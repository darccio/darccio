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
    "documents_pr", Path(__file__).with_name("open-documents-pr.py")
)
SCRIPT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SCRIPT)
REAL_RUN = subprocess.run
URI = "at://did:plc:aj77r5uwt72o6oimdjfplqoz/site.standard.document/3mgl7uesp65kn"


class DocumentsProposalTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="documents-proposal-test-")
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
        Path(".tools/atproto").mkdir(parents=True)
        Path(".tools/atproto/documents-allowlist.json").write_text(json.dumps({
            "version": 1, "documents": [{"path": "/posts/reviewed/", "publishedAt": "2025-01-01T10:00:00+01:00"}]
        }))
        self.git("add", ".tools/atproto/documents-allowlist.json")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "approved fixture")
        self.base = self.git("rev-parse", "HEAD").stdout.decode().strip()
        self.state = {"/posts/reviewed/": URI}
        Path(SCRIPT.FILES[0]).write_text(json.dumps(self.state) + "\n")
        Path("README.md").write_text("UNRELATED_TRACKED_CHANGE\n")
        Path("private-untracked.txt").write_text("UNRELATED_PRIVATE_DATA\n")
        self.calls = []
        self.branch = None
        self.contents = {}
        self.prs = []
        self.remote_files = list(SCRIPT.FILES)
        self.main = self.base
        self.main_revisions = []
        self.expected = self.base

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
            if endpoint.endswith("/commits/main"):
                response = {"sha": self.main_revisions.pop(0) if self.main_revisions else self.main}
            elif "/git/matching-refs/heads/" in endpoint:
                response = [] if self.branch is None else [{"ref": "refs/heads/" + self.branch, "object": {"sha": "documents-commit"}}]
            elif "/compare/" in endpoint:
                self.assertTrue(endpoint.endswith(self.base + "...documents-commit"))
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
                response = {"sha": "documents-tree"}
            elif endpoint.endswith("/git/commits"):
                self.assertEqual(method, "POST")
                self.assertEqual(payload, {"message": "chore(atproto): update verified document URIs", "tree": "documents-tree", "parents": [self.base]})
                response = {"sha": "documents-commit"}
            elif endpoint.endswith("/git/refs"):
                self.assertEqual(payload["sha"], "documents-commit")
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
        with patch.object(SCRIPT.subprocess, "run", self.execute), \
                patch.dict(os.environ, {"DEPLOYED_SHA": self.expected}), \
                contextlib.redirect_stdout(output):
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

    def test_invalid_or_unapproved_uri_fails_before_api(self):
        for state in ({"/posts/reviewed/": URI + "\n"},
                      {"/posts/reviewed/": URI.replace("site.standard.document", "site.standard.publication")},
                      {"/posts/not-approved/": URI},
                      {"/posts/reviewed/": "<html>fallback</html>"}):
            with self.subTest(state=state):
                Path(SCRIPT.FILES[0]).write_text(json.dumps(state))
                with self.assertRaises(ValueError):
                    self.invoke()
                self.assertEqual(self.calls, [])

    def test_download_cannot_change_the_reviewed_allowlist(self):
        Path(".tools/atproto/documents-allowlist.json").write_text(json.dumps({
            "version": 1, "documents": [{"path": "/posts/not-approved/"}]
        }))
        Path(SCRIPT.FILES[0]).write_text(json.dumps({"/posts/not-approved/": URI}))
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
        Path(SCRIPT.FILES[0]).unlink()
        Path(SCRIPT.FILES[0]).symlink_to("../private-untracked.txt")
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertEqual(self.calls, [])

    def test_symlinked_parent_is_rejected_before_api(self):
        Path(".zas").rename("downloaded")
        Path(".zas").symlink_to("downloaded", target_is_directory=True)
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertEqual(self.calls, [])

    def test_duplicate_json_keys_are_rejected_before_api(self):
        Path(SCRIPT.FILES[0]).write_text(
            '{"/posts/reviewed/":' + json.dumps(URI) + ',"/posts/reviewed/":' + json.dumps(URI) + '}')
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertEqual(self.calls, [])

    def test_checkout_must_match_verified_deployment(self):
        self.expected = "f" * 40
        with self.assertRaises(ValueError):
            self.invoke()
        self.assertEqual(self.calls, [])

    def test_superseded_revision_makes_no_branch_or_pr(self):
        self.main = "f" * 40
        self.assertIn("superseded", self.invoke())
        self.assertEqual(len(self.calls), 1)
        self.assertTrue(self.calls[0][0][-1].endswith("/commits/main"))

    def test_main_moving_during_branch_creation_makes_no_pr(self):
        self.main_revisions = [self.base, "f" * 40]
        self.assertIn("superseded", self.invoke())
        self.assertIsNotNone(self.branch)
        self.assertFalse(any(args[:2] in (["pr", "create"], ["workflow", "run"]) for args, _ in self.calls))



if __name__ == "__main__":
    unittest.main()
