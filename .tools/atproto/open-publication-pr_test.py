#!/usr/bin/env python3
"""Exercise the exact GitHub handoff offline, including interrupted reruns."""

import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location(
    "publication_pr", Path(__file__).with_name("open-publication-pr.py")
)
SCRIPT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SCRIPT)
REAL_RUN = subprocess.run
URI = "at://did:plc:aj77r5uwt72o6oimdjfplqoz/site.standard.publication/3mgl7uesp65kn"


class PublicationProposalTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="publication-proposal-test-")
        self.old_directory = Path.cwd()
        os.chdir(self.temporary.name)
        Path(".zas").mkdir()
        Path(".well-known").mkdir()
        self.git("init", "--quiet", "--initial-branch=main", "--template=")
        Path("README.md").write_text("Original fixture.\n")
        self.git("add", "README.md")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "fixture")
        self.base = self.git("rev-parse", "HEAD").stdout.decode().strip()
        self.state = {
            "uri": URI, "cid": "bafyreicysix5fuqjf3x7bpaiqo5fhemgqdyvaww6ftu4o67jbha4milkxe",
            "did": "did:plc:aj77r5uwt72o6oimdjfplqoz",
            "pds": "https://eurosky.social", "url": "https://dario.cat",
        }
        Path(SCRIPT.FILES[0]).write_text(json.dumps(self.state) + "\n")
        Path(SCRIPT.FILES[1]).write_text(URI)
        Path("README.md").write_text("UNRELATED_TRACKED_CHANGE\n")
        Path("private-untracked.txt").write_text("UNRELATED_PRIVATE_DATA\n")
        self.calls = []
        self.branch = None
        self.contents = {}
        self.prs = []
        self.remote_files = list(SCRIPT.FILES)

    def tearDown(self):
        os.chdir(self.old_directory)
        self.temporary.cleanup()

    def git(self, *arguments):
        environment = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}
        return REAL_RUN(["git", *arguments], check=True, capture_output=True, env=environment)

    def execute(self, arguments, **options):
        if arguments[0] != "gh":
            return REAL_RUN(arguments, **options)
        args = arguments[1:]
        payload = json.loads(options["input"]) if options.get("input") else None
        self.calls.append((args, payload))
        response = None
        if args[0] == "api":
            method = args[args.index("--method") + 1] if "--method" in args else "GET"
            endpoint = next(arg for arg in args if arg.startswith("repos/"))
            self.assertTrue(endpoint.startswith("repos/darccio/darccio/"))
            if "/git/matching-refs/heads/" in endpoint:
                response = [] if self.branch is None else [{"ref": "refs/heads/" + self.branch, "object": {"sha": "publication-commit"}}]
            elif "/compare/" in endpoint:
                self.assertTrue(endpoint.endswith(self.base + "...publication-commit"))
                response = {"total_commits": 1, "files": [{"filename": filename} for filename in self.remote_files]}
            elif "/contents/" in endpoint:
                filename = endpoint.split("/contents/", 1)[1].split("?ref=", 1)[0]
                content = self.contents[filename].encode()
                response = {"sha": hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()}
            elif endpoint.endswith("/git/commits/" + self.base):
                response = {"tree": {"sha": "original-tree"}}
            elif endpoint.endswith("/git/trees"):
                self.assertEqual(method, "POST")
                self.assertEqual(payload["base_tree"], "original-tree")
                self.assertEqual({entry["path"] for entry in payload["tree"]}, set(SCRIPT.FILES))
                self.assertTrue(all(entry["mode"] == "100644" and entry["type"] == "blob" for entry in payload["tree"]))
                self.contents = {entry["path"]: entry["content"] for entry in payload["tree"]}
                response = {"sha": "publication-tree"}
            elif endpoint.endswith("/git/commits"):
                self.assertEqual(method, "POST")
                self.assertEqual(payload, {"message": "feat(atproto): verify site publication", "tree": "publication-tree", "parents": [self.base]})
                response = {"sha": "publication-commit"}
            elif endpoint.endswith("/git/refs"):
                self.assertEqual(payload["sha"], "publication-commit")
                self.branch = payload["ref"].removeprefix("refs/heads/")
                response = {}
            else:
                self.fail("Unexpected GitHub endpoint: " + endpoint)
        elif args[:2] == ["pr", "list"]:
            self.assertEqual(args[args.index("--head") + 1], self.branch)
            response = self.prs
        elif args[:2] == ["pr", "create"]:
            self.assertEqual(args[args.index("--base") + 1], "main")
            self.assertEqual(args[args.index("--head") + 1], self.branch)
            self.assertIn(URI, Path(args[args.index("--body-file") + 1]).read_text())
            self.prs = [{"number": 99999, "state": "OPEN"}]
        elif args[:2] == ["workflow", "run"]:
            self.assertEqual(args, ["workflow", "run", "site-checks.yml", "--repo", "darccio/darccio", "--ref", self.branch])
        else:
            self.fail("Unexpected gh command: " + str(args))
        return subprocess.CompletedProcess(arguments, 0, json.dumps(response).encode(), b"")

    def invoke(self):
        output = io.StringIO()
        with patch.object(SCRIPT.subprocess, "run", self.execute), contextlib.redirect_stdout(output):
            SCRIPT.propose()
        return output.getvalue()

    def test_atomic_outputs_and_repeat(self):
        self.invoke()
        commits = [payload for args, payload in self.calls if any(arg.endswith("/git/commits") for arg in args)]
        self.assertEqual(len(commits), 1)
        self.assertEqual(self.git("rev-parse", "HEAD").stdout.decode().strip(), self.base)
        self.assertNotIn("UNRELATED", json.dumps(self.calls))
        self.calls.clear()
        self.invoke()
        self.assertFalse(any("POST" in args for args, _ in self.calls))
        self.assertFalse(any(args[:2] == ["pr", "create"] for args, _ in self.calls))
        self.assertEqual(self.calls[-1][0][:3], ["workflow", "run", "site-checks.yml"])

    def test_recover_after_branch_creation(self):
        self.invoke()
        self.prs = []
        self.calls.clear()
        self.invoke()
        self.assertEqual(sum(args[:2] == ["pr", "create"] for args, _ in self.calls), 1)
        self.assertFalse(any("POST" in args for args, _ in self.calls))

    def test_closed_proposal_is_not_recreated(self):
        self.invoke()
        self.prs[0]["state"] = "CLOSED"
        self.calls.clear()
        self.assertIn("already reviewed", self.invoke())
        self.assertFalse(any(args[:2] in (["pr", "create"], ["workflow", "run"]) for args, _ in self.calls))

    def test_matching_committed_files_need_no_api(self):
        self.git("add", *SCRIPT.FILES)
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "verified fixture")
        self.assertIn("already committed", self.invoke())
        self.assertEqual(self.calls, [])

    def test_updated_cid_with_unchanged_uri_can_repeat(self):
        self.git("add", *SCRIPT.FILES)
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "original publication")
        self.base = self.git("rev-parse", "HEAD").stdout.decode().strip()
        self.state["cid"] = self.state["cid"][:-1] + "a"
        Path(SCRIPT.FILES[0]).write_text(json.dumps(self.state))
        self.remote_files = [SCRIPT.FILES[0]]
        self.invoke()
        self.calls.clear()
        self.invoke()
        self.assertFalse(any("POST" in args for args, _ in self.calls))
        self.assertEqual(self.calls[-1][0][:3], ["workflow", "run", "site-checks.yml"])

    def test_invalid_or_mismatched_uri_fails_before_api(self):
        for value in (URI + "\n", URI.replace("site.standard.publication", "site.standard.document"), "<html>fallback</html>"):
            with self.subTest(value=value):
                Path(SCRIPT.FILES[1]).write_text(value)
                with self.assertRaises(ValueError):
                    self.invoke()
                self.assertEqual(self.calls, [])
        Path(SCRIPT.FILES[1]).write_text(URI)
        self.state["uri"] = "different"
        Path(SCRIPT.FILES[0]).write_text(json.dumps(self.state))
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertEqual(self.calls, [])

    def test_existing_branch_changes_are_rejected(self):
        self.invoke()
        self.contents[SCRIPT.FILES[0]] = "unexpected remote change"
        self.calls.clear()
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertFalse(any("POST" in args for args, _ in self.calls))

    def test_existing_branch_extra_files_are_rejected(self):
        self.invoke()
        self.remote_files.append(".github/workflows/unrelated.yml")
        self.calls.clear()
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertFalse(any("POST" in args for args, _ in self.calls))
        self.assertFalse(any(args[:2] in (["pr", "create"], ["workflow", "run"]) for args, _ in self.calls))

    def test_symlink_is_rejected_before_api(self):
        Path(SCRIPT.FILES[1]).unlink()
        Path(SCRIPT.FILES[1]).symlink_to("../private-untracked.txt")
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertEqual(self.calls, [])

    def test_invalid_or_extra_state_is_rejected_before_api(self):
        for key, value in (("cid", "bafy-invalid"), ("did", "did:plc:other"),
                           ("pds", "https://other.invalid"), ("url", "https://rizoma.pckt.blog"),
                           ("password", "unexpected-private-field")):
            with self.subTest(key=key):
                altered = dict(self.state)
                altered[key] = value
                Path(SCRIPT.FILES[0]).write_text(json.dumps(altered))
                with self.assertRaises(ValueError):
                    self.invoke()
                self.assertEqual(self.calls, [])


if __name__ == "__main__":
    unittest.main()
