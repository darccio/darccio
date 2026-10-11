#!/usr/bin/env bash
# Exercise the update-PR handoff without credentials or network access.
set -euo pipefail
script_directory=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
python3 - "$script_directory/open-update-pr.sh" <<'PY'
import base64
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


SCRIPT = Path(sys.argv[1]).resolve()
SITE = "repos/darccio/darccio"
SOURCE = "2" * 40
OLD_SOURCE = "1" * 40

# The stub records the exact CLI request and implements only the API surface
# used by the publisher. It rejects unexpected commands instead of reaching gh.
MOCK_GH = r'''#!/usr/bin/env python3
import base64
import hashlib
import json
import os
from pathlib import Path
import sys

root = Path(os.environ["MOCK_STATE"])
state = json.loads((root / "state.json").read_text())
args = sys.argv[1:]
call = {"args": args}
if "--input" in args:
    call["input"] = json.loads(Path(args[args.index("--input") + 1]).read_text())
if "--body-file" in args:
    call["body"] = Path(args[args.index("--body-file") + 1]).read_text()
with (root / "calls.jsonl").open("a") as out:
    out.write(json.dumps(call) + "\n")

def save():
    (root / "state.json").write_text(json.dumps(state))

if args[0] == "api":
    method = args[args.index("--method") + 1] if "--method" in args else "GET"
    endpoint = args[3] if "--method" in args else args[1]
    if endpoint.startswith("repos/darccio/darccio/git/ref/heads/"):
        assert method == "GET"
        if not state["branch_exists"]:
            sys.exit(1)
        print(json.dumps({"ref": "refs/heads/" + state["branch"]}))
    elif endpoint == "repos/darccio/darccio/git/refs":
        assert method == "POST"
        assert call["input"] == {"ref": "refs/heads/" + state["branch"], "sha": state["base"]}
        state["branch_exists"] = True
        save()
        print("{}")
    elif endpoint.startswith("repos/darccio/darccio/contents/.zas/projects.yml?ref="):
        assert method == "GET"
        assert endpoint.endswith("?ref=" + state["branch"])
        assert args[-2:] == ["--jq", ".sha"]
        print(state["blob"])
    elif endpoint == "repos/darccio/darccio/contents/.zas/projects.yml":
        assert method == "PUT"
        payload = call["input"]
        assert payload["branch"] == state["branch"]
        assert payload["sha"] == state["blob"]
        content = base64.b64decode(payload["content"], validate=True)
        state["blob"] = hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()
        save()
        print("{}")
    else:
        raise AssertionError("Unexpected API endpoint: " + endpoint)
elif args[:2] == ["pr", "list"]:
    assert args[2:] == ["--repo", "darccio/darccio", "--head", state["branch"], "--state", "open", "--json", "number", "--jq", "length"]
    print("1" if state["pr_exists"] else "0")
elif args[:2] == ["pr", "create"]:
    assert args[args.index("--repo") + 1] == "darccio/darccio"
    assert args[args.index("--base") + 1] == "main"
    assert args[args.index("--head") + 1] == state["branch"]
    state["pr_exists"] = True
    save()
    print("https://github.com/darccio/darccio/pull/99999")
elif args[:2] == ["workflow", "run"]:
    assert args[2:] == ["site-checks.yml", "--repo", "darccio/darccio", "--ref", state["branch"]]
else:
    raise AssertionError("Unexpected gh command: " + repr(args))
'''


def check(condition, message):
    if not condition:
        raise AssertionError(message)


def git(root, *args):
    clean = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}
    result = subprocess.run(["git", "-C", str(root), *args], env=clean, text=True, capture_output=True, check=True)
    return result.stdout.strip()


def fixture(base, project="mergo", changed=True):
    root = base / project
    root.mkdir()
    (root / ".zas").mkdir()
    old = "version: 1\nprojects:\n  - id: " + project + "\n    commit: " + OLD_SOURCE + "\n"
    proposed = old.replace(OLD_SOURCE, SOURCE)
    (root / ".zas/projects.yml").write_text(old)
    (root / "README.md").write_text("Original project fixture.\n")
    empty_template = base / (project + "-empty-template")
    empty_template.mkdir()
    git(root, "init", "--quiet", "--initial-branch=main", "--template=" + str(empty_template))
    git(root, "add", ".zas/projects.yml", "README.md")
    git(root, "-c", "user.name=Documentation fixture", "-c", "user.email=docs@example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "fixture")
    base_commit = git(root, "rev-parse", "HEAD")
    old_blob = git(root, "hash-object", ".zas/projects.yml")
    if changed:
        (root / ".zas/projects.yml").write_text(proposed)
    # These unrelated changes would be included by an unsafe broad Git commit.
    (root / "README.md").write_text("UNRELATED_TRACKED_EDIT\n")
    (root / "fetched-private-file.txt").write_text("UNRELATED_UNTRACKED_DATA\n")
    mock = base / (project + "-mock")
    mock.mkdir()
    branch = "bot/docs-" + project + "-" + SOURCE
    (mock / "state.json").write_text(json.dumps({"branch_exists": False, "pr_exists": False, "branch": branch, "base": base_commit, "blob": old_blob}))
    (mock / "calls.jsonl").write_text("")
    bin_dir = mock / "bin"
    bin_dir.mkdir()
    (bin_dir / "gh").write_text(MOCK_GH)
    (bin_dir / "gh").chmod(0o700)
    runner = mock / "runner"
    runner.mkdir()
    return {"root": root, "mock": mock, "bin": bin_dir, "runner": runner, "branch": branch, "base": base_commit, "old_blob": old_blob, "proposed": proposed}


def invoke(f, repo=None, commit=None, succeeds=True):
    (f["mock"] / "calls.jsonl").write_text("")
    env = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}
    env.update({"PATH": str(f["bin"]) + os.pathsep + env["PATH"], "MOCK_STATE": str(f["mock"]), "RUNNER_TEMP": str(f["runner"]), "SOURCE_REPOSITORY": repo if repo is not None else "darccio/" + f["root"].name, "SOURCE_COMMIT": commit if commit is not None else SOURCE, "GH_TOKEN": "offline-fixture-only"})
    result = subprocess.run(["bash", str(SCRIPT)], cwd=f["root"], env=env, text=True, capture_output=True)
    check((result.returncode == 0) == succeeds, "Unexpected script result:\n" + result.stdout + result.stderr)
    return [json.loads(line) for line in (f["mock"] / "calls.jsonl").read_text().splitlines()], result


for command in ("bash", "git", "jq", "base64"):
    check(shutil.which(command) is not None, "Missing test prerequisite: " + command)

with tempfile.TemporaryDirectory(prefix="dario-docs-update-pr-") as temporary:
    base = Path(temporary)
    for project in ("mergo", "zas", "darna"):
        f = fixture(base, project)
        calls, _ = invoke(f)
        check(len(calls) == 7, "New update should make seven exact gh calls")
        check(calls[0]["args"] == ["api", SITE + "/git/ref/heads/" + f["branch"]], "Unexpected branch lookup")
        check(calls[1]["args"][:4] == ["api", "--method", "POST", SITE + "/git/refs"], "Missing branch creation")
        puts = [call for call in calls if call["args"][:3] == ["api", "--method", "PUT"]]
        check(len(puts) == 1, "Expected exactly one Contents API commit")
        put = puts[0]
        check(put["args"][3] == SITE + "/contents/.zas/projects.yml", "Commit targets more than the source manifest")
        check(set(put["input"]) == {"branch", "sha", "message", "content"}, "Unexpected Contents API payload")
        check(put["input"]["sha"] == f["old_blob"], "Update does not compare the current branch blob")
        check(put["input"]["message"] == "build(docs): update " + project + " source pin", "Incorrect commit message")
        check(base64.b64decode(put["input"]["content"], validate=True) == f["proposed"].encode(), "Commit content differs from the reviewed manifest")
        pr = [call for call in calls if call["args"][:2] == ["pr", "create"]]
        check(len(pr) == 1 and SOURCE in pr[0]["body"], "Missing pin-update PR")
        check("UNRELATED" not in json.dumps(calls), "Unrelated source data reached the API")
        check(calls[-1]["args"] == ["workflow", "run", "site-checks.yml", "--repo", "darccio/darccio", "--ref", f["branch"]], "Site checks were not explicitly dispatched on the update branch")
        check(git(f["root"], "rev-parse", "HEAD") == f["base"], "Handoff changed the local Git history")
        repeat, _ = invoke(f)
        check(len(repeat) == 4, "Unchanged bot revision should only read existing state and dispatch checks")
        check(not any(call["args"][:3] in (["api", "--method", "PUT"], ["api", "--method", "POST"]) for call in repeat), "Repeat wrote an unchanged pin")
        check(not any(call["args"][:2] == ["pr", "create"] for call in repeat), "Repeat created a duplicate PR")
        check(repeat[-1]["args"][:3] == ["workflow", "run", "site-checks.yml"], "Repeat omitted explicit checks")
        print(project + ": new update and idempotent repeat passed")

    current_base = base / "current"
    current_base.mkdir()
    current = fixture(current_base, changed=False)
    calls, result = invoke(current, commit=OLD_SOURCE)
    check(calls == [], "Already-current main pin made API calls")
    check("already current" in result.stdout, "Already-current pin did not explain the skip")
    print("Already-current main pin skips all gh calls")

    invalid_base = base / "invalid"
    invalid_base.mkdir()
    invalid = fixture(invalid_base)
    for repo in ("attacker/project", "darccio/mergo; touch injected", "", "darccio/mergo\n"):
        calls, _ = invoke(invalid, repo=repo, succeeds=False)
        check(calls == [], "Unsupported source repository reached gh")
    marker = invalid["root"] / "injected"
    for commit in ("", "abc", "A" * 40, "g" * 40, SOURCE + "\n", "$(touch injected)", SOURCE + "; touch injected"):
        calls, _ = invoke(invalid, commit=commit, succeeds=False)
        check(calls == [], "Malformed source commit reached gh")
        check(not marker.exists(), "Source commit was interpreted as shell input")
    print("Unsupported sources and malformed/injection commits fail before gh calls")

print("All offline docs-update PR tests passed.")
PY
