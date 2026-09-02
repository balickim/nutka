#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["lizard==1.24.0"]
# ///
"""Tests the code-budget ratchet without reading or changing the repository baseline."""

from __future__ import annotations

import unittest
from types import SimpleNamespace

import check


def report(files: dict) -> dict:
    return {
        "version": 1,
        "tool": check.TOOL,
        "limits": {"ccn": check.CCN_LIMIT, "nloc": check.NLOC_LIMIT},
        "files": files,
    }


class RatchetTests(unittest.TestCase):
    def test_new_violations_fail(self) -> None:
        current = report({"new.ts": {"nloc": 251, "functions": {"work()#1": 11}}})
        errors = check.increase_errors(report({}), current)
        self.assertEqual(
            [
                "new.ts: NLOC 251 exceeds 250 in a new baseline entry",
                "new.ts: work()#1 has new CCN 11, above 10",
            ],
            errors,
        )

    def test_metric_increases_fail(self) -> None:
        base = report({"old.go": {"nloc": 300, "functions": {"work()#1": 12}}})
        current = report({"old.go": {"nloc": 301, "functions": {"work()#1": 13}}})
        errors = check.increase_errors(base, current)
        self.assertEqual(2, len(errors))

    def test_budget_increases_fail(self) -> None:
        base = report({})
        current = report({})
        current["limits"]["ccn"] = 11
        self.assertEqual(["ccn budget increased from 10 to 11"], check.increase_errors(base, current))

    def test_reductions_and_removals_pass(self) -> None:
        base = report({"old.go": {"nloc": 300, "functions": {"work()#1": 12, "gone()#1": 11}}})
        current = report({"old.go": {"nloc": 299, "functions": {"work()#1": 11}}})
        self.assertEqual([], check.increase_errors(base, current))

    def test_stale_entries_are_reported(self) -> None:
        stored = report({"old.go": {"nloc": 300}})
        current = report({"old.go": {"nloc": 299}})
        self.assertEqual(["old.go"], check.baseline_differences(stored, current))

    def test_removed_violation_requires_baseline_update(self) -> None:
        stored = report({"old.go": {"nloc": 300}})
        self.assertEqual(["old.go"], check.baseline_differences(stored, report({})))

    def test_duplicate_function_names_get_stable_occurrences(self) -> None:
        functions = [
            SimpleNamespace(start_line=20, long_name="work()", name="work"),
            SimpleNamespace(start_line=10, long_name="work()", name="work"),
        ]
        identifiers = [identifier for identifier, _ in check.function_ids(functions)]
        self.assertEqual(["work()#1", "work()#2"], identifiers)

    def test_test_and_generated_files_are_excluded(self) -> None:
        self.assertFalse(check.is_source("apps/backend/domain_test.go"))
        self.assertFalse(check.is_source("apps/backend/internal/fixtures/sample.go"))
        self.assertFalse(check.is_source("apps/app/dist/chunk.js"))
        self.assertFalse(check.is_source("apps/app/node_modules/library/index.js"))
        self.assertFalse(check.is_source("apps/backend/database/migrations/123_created_users.go"))
        self.assertTrue(check.is_source("apps/app/src/auth/auth.ts"))
        self.assertTrue(check.is_source("apps/backend/auth.go"))
        self.assertTrue(check.is_source("apps/landing/src/scripts/reveal.ts"))


if __name__ == "__main__":
    unittest.main()
