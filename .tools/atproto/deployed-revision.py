#!/usr/bin/env python3
"""Select the immutable, current production revision before publishing."""

import json
import os
from pathlib import Path
import re
import subprocess

REPOSITORY = "darccio/darccio"
STATE_FILES = {".zas/atproto.json"}


def api(endpoint):
    result = subprocess.run(["gh", "api", "repos/" + REPOSITORY + "/" + endpoint],
                            check=True, capture_output=True, timeout=30)
    return json.loads(result.stdout)


def select(event_name, event, fetch):
    if event.get("repository", {}).get("full_name") != REPOSITORY:
        raise ValueError("Unexpected repository")
    main = fetch("commits/main")["sha"]
    if not re.fullmatch(r"[0-9a-f]{40}", main):
        raise ValueError("Invalid production revision")
    if event_name == "workflow_run":
        run = event.get("workflow_run", {})
        if (run.get("name") != "Vanity after deploy" or run.get("conclusion") != "success"
                or run.get("head_branch") != "main"
                or run.get("head_repository", {}).get("full_name") != REPOSITORY):
            raise ValueError("Untrusted deployment workflow event")
        if run.get("head_sha") != main:
            return None
        # The backlink-only merge deploys through Pages but must not start
        # another publish/state-PR cycle. The initial implementation contains
        # more files and remains eligible for its harmless empty-list run.
        revision = fetch("commits/" + main)
        files = revision.get("files", [])
        if files and {file["filename"] for file in files}.issubset(STATE_FILES):
            return None
    elif event_name != "workflow_dispatch":
        raise ValueError("Unsupported publishing event")
    checks = fetch("commits/" + main + "/check-runs?check_name=Cloudflare%20Pages&filter=latest&per_page=100")
    trusted = [check for check in checks.get("check_runs", []) if (
        check.get("head_sha") == main and check.get("name") == "Cloudflare Pages"
        and check.get("app", {}).get("id") == 85455
        and check.get("app", {}).get("slug") == "cloudflare-workers-and-pages"
        and (check.get("details_url") or "").startswith("https://dash.cloudflare.com/?to=/")
        and "/pages/view/dario/" in (check.get("details_url") or "")
    )]
    if not trusted:
        raise ValueError("No production Pages deployment for current main")
    check = max(trusted, key=lambda item: item["id"])
    if (check.get("status") != "completed" or check.get("conclusion") != "success"
            or check.get("output", {}).get("title") != "Deployed successfully"):
        raise ValueError("Production Pages deployment has not succeeded")
    suite = fetch("check-suites/" + str(check["check_suite"]["id"]))
    if ("head_branch" not in suite or suite["head_branch"] not in (None, "main")
            or suite.get("head_sha") != main or suite.get("app", {}).get("id") != 85455
            or suite.get("app", {}).get("slug") != "cloudflare-workers-and-pages"):
        raise ValueError("Pages check suite does not identify production main")
    return main


def main():
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
    revision = select(os.environ["GITHUB_EVENT_NAME"], event, api)
    with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
        output.write("ready=" + str(revision is not None).lower() + "\n")
        if revision:
            output.write("sha=" + revision + "\n")
    print("Confirmed production revision " + revision if revision else "Skipped superseded or backlink-only revision.")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError, OSError, subprocess.SubprocessError):
        raise SystemExit("Production deployment verification failed; no publishing credentials were used.") from None
