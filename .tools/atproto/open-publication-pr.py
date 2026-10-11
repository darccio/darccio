#!/usr/bin/env python3
"""Propose only the two public bootstrap outputs, using one Git tree/commit."""

import base64
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

REPOSITORY = "darccio/darccio"
FILES = (".zas/atproto-publication.json", ".well-known/site.standard.publication")


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


def propose():
    contents = {}
    for filename in FILES:
        path = Path(filename)
        if path.is_symlink() or not path.is_file():
            raise ValueError("Missing regular publication output: " + filename)
        contents[filename] = path.read_text()
    state = json.loads(contents[FILES[0]])
    uri = contents[FILES[1]]
    if not re.fullmatch(
        r"at://did:plc:aj77r5uwt72o6oimdjfplqoz/site\.standard\.publication/[234567a-j][234567a-z]{12}", uri
    ):
        raise ValueError("Invalid publication verification URI")
    if set(state) != {"uri", "cid", "did", "pds", "url"} or any(
        state.get(key) != value for key, value in {
            "uri": uri, "did": "did:plc:aj77r5uwt72o6oimdjfplqoz",
            "pds": "https://eurosky.social", "url": "https://dario.cat",
        }.items()
    ):
        raise ValueError("Publication state and verification file disagree")
    cid = state["cid"]
    if not isinstance(cid, str) or not re.fullmatch(r"b[a-z2-7]{58}", cid):
        raise ValueError("Invalid publication CID")
    decoded = base64.b32decode(cid[1:].upper() + "======")
    if (decoded[:4] != bytes((1, 0x71, 0x12, 0x20)) or
            base64.b32encode(decoded).decode().lower().rstrip("=") != cid[1:]):
        raise ValueError("Invalid publication CID")

    unchanged = True
    for filename, content in contents.items():
        previous = subprocess.run(
            ["git", "show", "HEAD:" + filename], capture_output=True
        )
        unchanged &= previous.returncode == 0 and previous.stdout == content.encode()
    if unchanged:
        print("Publication verification is already committed.")
        return

    revision = hashlib.sha256(json.dumps(contents, sort_keys=True).encode()).hexdigest()[:16]
    branch = "bot/standard-site-publication-" + revision
    api = "repos/" + REPOSITORY
    base = git("rev-parse", "HEAD").decode().strip()
    refs = gh("api", api + "/git/matching-refs/heads/" + branch)
    existing = next((ref for ref in refs if ref["ref"] == "refs/heads/" + branch), None)
    if existing:
        difference = gh("api", api + "/compare/" + base + "..." + existing["object"]["sha"])
        changed_files = {
            file["filename"] for file in difference.get("files", [])
        }
        if difference.get("total_commits") != 1 or not changed_files or not changed_files.issubset(FILES):
            raise ValueError("Existing bot branch changes files outside the atomic publication proposal")
        for filename, content in contents.items():
            remote = gh("api", api + "/contents/" + filename + "?ref=" + branch)
            expected_blob = hashlib.sha1(
                b"blob " + str(len(content.encode())).encode() + b"\0" + content.encode()
            ).hexdigest()
            if remote.get("sha") != expected_blob:
                raise ValueError("Existing bot branch contains different publication outputs")
    else:
        commit = gh("api", api + "/git/commits/" + base)
        tree = gh("api", "--method", "POST", api + "/git/trees", payload={
            "base_tree": commit["tree"]["sha"],
            "tree": [{"path": filename, "mode": "100644", "type": "blob", "content": content}
                     for filename, content in contents.items()],
        })
        commit = gh("api", "--method", "POST", api + "/git/commits", payload={
            "message": "feat(atproto): verify site publication",
            "tree": tree["sha"], "parents": [base],
        })
        gh("api", "--method", "POST", api + "/git/refs", payload={
            "ref": "refs/heads/" + branch, "sha": commit["sha"],
        })

    pull_requests = gh("pr", "list", "--repo", REPOSITORY, "--head", branch,
                       "--state", "all", "--json", "number,state")
    if not pull_requests:
        with tempfile.TemporaryDirectory(dir=os.environ.get("RUNNER_TEMP")) as temporary:
            body = Path(temporary) / "publication-pr.md"
            body.write_text(
                "Verify the dario.cat Standard.site publication with its public AT URI.\n\n"
                "The manual bootstrap verified the account, publication record and CID, "
                "then passed the guarded site build. This single commit contains only "
                "the public publication state and exact verification URI. Merging deploys "
                "the backlink through the existing Cloudflare Pages Git integration.\n\n"
                "Publication: " + uri + "\n\n"
                "This does not publish articles. The existing Rizoma publication is preserved.\n"
            )
            subprocess.run([
                "gh", "pr", "create", "--repo", REPOSITORY, "--base", "main",
                "--head", branch, "--title", "Standard.site: verify site publication",
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
        raise SystemExit("Publication proposal failed: " + (
            "GitHub or Git command failed" if isinstance(error, subprocess.CalledProcessError)
            else str(error)
        )) from None
