#!/usr/bin/env python3
"""Propose only verified public document URIs, using one Git tree/commit."""

import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

REPOSITORY = "darccio/darccio"
FILES = (".zas/atproto.json",)


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("Duplicate JSON key in document proposal")
        result[key] = value
    return result


def git(*arguments):
    return subprocess.run(
        ["git", *arguments], check=True, capture_output=True
    ).stdout


def gh(*arguments, payload=None):
    result = subprocess.run(
        ["gh", *arguments, *(["--input", "-"] if payload is not None else [])],
        input=json.dumps(payload).encode() if payload is not None else None,
        check=True, capture_output=True,
    )
    return json.loads(result.stdout) if result.stdout.strip() else None


def current_main(base):
    revision = gh("api", "repos/" + REPOSITORY + "/commits/main")
    if revision.get("sha") != base:
        print("Production revision was superseded; no document proposal created.")
        return False
    return True


def propose():
    contents = {}
    for filename in FILES:
        path = Path(filename)
        if (path.is_symlink() or not path.is_file()
                or any(parent.is_symlink() for parent in path.parents)):
            raise ValueError("Missing regular publication output: " + filename)
        contents[filename] = path.read_text()
    manifest = json.loads(contents[FILES[0]], object_pairs_hook=unique_object)
    # Approval comes from the immutable checkout, never from a downloaded file.
    allowlist = json.loads(git("show", "HEAD:.tools/atproto/documents-allowlist.json"),
                          object_pairs_hook=unique_object)
    if (set(allowlist) != {"version", "documents"} or allowlist["version"] != 1
            or not isinstance(allowlist["documents"], list) or not isinstance(manifest, dict)):
        raise ValueError("Invalid reviewed document manifest")
    approved = {entry["path"] for entry in allowlist["documents"]}
    if len(approved) != len(allowlist["documents"]):
        raise ValueError("Duplicate reviewed document paths")
    seen = set()
    for path, uri in manifest.items():
        if (path not in approved or not isinstance(uri, str)
                or not re.fullmatch(r"/(?:posts|en/posts|es/posts|rizoma)/(?:[a-z0-9-]+/)+", path)
                or not re.fullmatch(r"at://did:plc:aj77r5uwt72o6oimdjfplqoz/site\.standard\.document/[234567a-j][234567a-z]{12}", uri)
                or uri in seen):
            raise ValueError("Unapproved, invalid or duplicate document URI")
        seen.add(uri)

    unchanged = True
    for filename, content in contents.items():
        previous = subprocess.run(
            ["git", "show", "HEAD:" + filename], capture_output=True
        )
        unchanged &= previous.returncode == 0 and previous.stdout == content.encode()
    if unchanged:
        print("Document verification is already committed.")
        return

    revision = hashlib.sha256(json.dumps(contents, sort_keys=True).encode()).hexdigest()[:16]
    branch = "bot/standard-site-documents-" + revision
    api = "repos/" + REPOSITORY
    base = git("rev-parse", "HEAD").decode().strip()
    expected = os.environ.get("DEPLOYED_SHA", base)
    if expected != base or not re.fullmatch(r"[0-9a-f]{40}", base):
        raise ValueError("Document proposal checkout differs from the deployed revision")
    if not current_main(base):
        return
    refs = gh("api", api + "/git/matching-refs/heads/" + branch)
    existing = next((ref for ref in refs if ref["ref"] == "refs/heads/" + branch), None)
    if existing:
        difference = gh("api", api + "/compare/" + base + "..." + existing["object"]["sha"])
        changed_files = {
            file["filename"] for file in difference.get("files", [])
        }
        if difference.get("total_commits") != 1 or not changed_files or not changed_files.issubset(FILES):
            raise ValueError("Existing bot branch changes files outside the atomic document proposal")
        for filename, content in contents.items():
            remote = gh("api", api + "/contents/" + filename + "?ref=" + branch)
            expected_blob = hashlib.sha1(
                b"blob " + str(len(content.encode())).encode() + b"\0" + content.encode()
            ).hexdigest()
            if remote.get("sha") != expected_blob:
                raise ValueError("Existing bot branch contains different document outputs")
    else:
        commit = gh("api", api + "/git/commits/" + base)
        tree = gh("api", "--method", "POST", api + "/git/trees", payload={
            "base_tree": commit["tree"]["sha"],
            "tree": [{"path": filename, "mode": "100644", "type": "blob", "content": content}
                     for filename, content in contents.items()],
        })
        commit = gh("api", "--method", "POST", api + "/git/commits", payload={
            "message": "chore(atproto): update verified document URIs",
            "tree": tree["sha"], "parents": [base],
        })
        gh("api", "--method", "POST", api + "/git/refs", payload={
            "ref": "refs/heads/" + branch, "sha": commit["sha"],
        })

    # A new push may revoke approval while the branch is being prepared. Leave
    # the immutable branch alone, but never propose it as current main's work.
    if not current_main(base):
        return
    pull_requests = gh("pr", "list", "--repo", REPOSITORY, "--head", branch,
                       "--state", "all", "--json", "number,state")
    if not pull_requests:
        with tempfile.TemporaryDirectory(dir=os.environ.get("RUNNER_TEMP")) as temporary:
            body = Path(temporary) / "documents-pr.md"
            body.write_text(
                "Add verified Standard.site document backlinks for the reviewed posts.\n\n"
                "The separate publisher verified the account, conditional record writes and public "
                "readback, then passed the guarded site build. This single commit contains only "
                "the public path-to-URI mapping. Merging deploys the backlinks through Cloudflare "
                "Pages and does not start another publishing cycle.\n\n"
                + "\n".join("- " + path + ": " + uri for path, uri in sorted(manifest.items())) + "\n"
            )
            subprocess.run([
                "gh", "pr", "create", "--repo", REPOSITORY, "--base", "main",
                "--head", branch, "--title", "Standard.site: update verified document URIs",
                "--body-file", str(body),
            ], check=True)
    elif not any(pr["state"] == "OPEN" for pr in pull_requests):
        print("This verification proposal was already reviewed; no duplicate PR created.")
        return

    # GITHUB_TOKEN writes do not start normal push/PR workflows.
    subprocess.run(["gh", "workflow", "run", "site-checks.yml", "--repo",
                    REPOSITORY, "--ref", branch], check=True)


if __name__ == "__main__":
    try:
        propose()
    except (ValueError, KeyError, subprocess.CalledProcessError) as error:
        # Never reproduce gh stderr or environment/authentication values.
        raise SystemExit("Document proposal failed: " + (
            "GitHub or Git command failed" if isinstance(error, subprocess.CalledProcessError)
            else str(error)
        )) from None
