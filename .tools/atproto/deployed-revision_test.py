#!/usr/bin/env python3
"""Production gating fixtures never contact GitHub or use credentials."""

import copy
import importlib.util
from pathlib import Path
import sys
import unittest

sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location("gate", Path(__file__).with_name("deployed-revision.py"))
GATE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GATE)
SHA = "a" * 40


class DeploymentGateTest(unittest.TestCase):
    def setUp(self):
        self.event = {"repository": {"full_name": GATE.REPOSITORY}, "workflow_run": {
            "name": "Vanity after deploy", "conclusion": "success", "head_sha": SHA,
            "head_branch": "main", "head_repository": {"full_name": GATE.REPOSITORY}}}
        app = {"id": 85455, "slug": "cloudflare-workers-and-pages"}
        self.check = {"id": 10, "head_sha": SHA, "name": "Cloudflare Pages", "app": app,
                      "details_url": "https://dash.cloudflare.com/?to=/account/pages/view/dario/deployments/123",
                      "status": "completed", "conclusion": "success",
                      "output": {"title": "Deployed successfully"}, "check_suite": {"id": 20}}
        self.suite = {"head_sha": SHA, "head_branch": "main", "app": app}
        self.files = ["posts/test/index.md"]

    def fetch(self, endpoint):
        if endpoint == "commits/main":
            return {"sha": SHA}
        if endpoint == "commits/" + SHA:
            return {"files": [{"filename": filename} for filename in self.files]}
        if "check-runs?" in endpoint:
            return {"check_runs": [self.check]}
        if endpoint == "check-suites/20":
            return self.suite
        self.fail("Unexpected read: " + endpoint)

    def select(self):
        return GATE.select("workflow_run", self.event, self.fetch)

    def test_current_production_and_repeated_delivery(self):
        self.assertEqual(self.select(), SHA)
        self.assertEqual(self.select(), SHA)
        self.suite["head_branch"] = None
        self.assertEqual(self.select(), SHA)

    def test_superseded_and_state_only_merge_skip(self):
        self.event["workflow_run"]["head_sha"] = "b" * 40
        self.assertIsNone(self.select())
        self.event["workflow_run"]["head_sha"] = SHA
        self.files = [".zas/atproto.json"]
        self.assertIsNone(self.select())
        self.files.append("posts/test/index.md")
        self.assertEqual(self.select(), SHA)

    def test_fork_preview_failure_and_wrong_workflow_reject(self):
        original = copy.deepcopy(self.event)
        for key, value in (("head_branch", "feature"), ("conclusion", "failure"),
                           ("name", "Site checks"), ("head_repository", {"full_name": "fork/site"})):
            with self.subTest(key=key):
                self.event = copy.deepcopy(original)
                self.event["workflow_run"][key] = value
                with self.assertRaises(ValueError):
                    self.select()
        self.event = {"repository": {"full_name": "fork/site"}}
        with self.assertRaises(ValueError):
            GATE.select("workflow_dispatch", self.event, self.fetch)

    def test_provider_identity_latest_status_and_revision_reject(self):
        original = copy.deepcopy(self.check)
        for key, value in (("head_sha", "b" * 40), ("app", {"id": 1, "slug": "fake"}),
                           ("details_url", "https://example.invalid/pages/view/dario/"),
                           ("name", "Preview"), ("conclusion", "failure"), ("status", "in_progress"),
                           ("output", {"title": "Preview succeeded"})):
            with self.subTest(key=key):
                self.check = copy.deepcopy(original)
                self.check[key] = value
                with self.assertRaises(ValueError):
                    self.select()

    def test_suite_sha_branch_and_missing_branch_reject(self):
        original = copy.deepcopy(self.suite)
        for key, value in (("head_sha", "b" * 40), ("head_branch", "preview"),
                           ("app", {"id": 1, "slug": "fake"})):
            with self.subTest(key=key):
                self.suite = copy.deepcopy(original)
                self.suite[key] = value
                with self.assertRaises(ValueError):
                    self.select()
        self.suite = copy.deepcopy(original)
        del self.suite["head_branch"]
        with self.assertRaises(ValueError):
            self.select()

    def test_manual_dry_run_and_absent_event(self):
        self.assertEqual(GATE.select("workflow_dispatch", self.event, self.fetch), SHA)
        with self.assertRaises(ValueError):
            GATE.select("push", self.event, self.fetch)
        with self.assertRaises(ValueError):
            GATE.select("workflow_run", {"repository": self.event["repository"]}, self.fetch)


if __name__ == "__main__":
    unittest.main()
