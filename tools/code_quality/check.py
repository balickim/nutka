#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["lizard==1.24.0"]
# ///
"""Enforces ratcheted complexity and file-size budgets for production source code."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from collections import defaultdict
from pathlib import Path
from typing import Any

import lizard


ROOT = Path(__file__).resolve().parents[2]
BASELINE_PATH = Path(__file__).with_name("baseline.json")
CCN_LIMIT = 10
NLOC_LIMIT = 250
TOOL = "lizard==1.24.0"
SOURCE_SUFFIXES = {".cjs", ".go", ".js", ".jsx", ".mjs", ".ts", ".tsx"}
SOURCE_ROOTS = ("apps/app/", "apps/backend/", "apps/landing/")
GENERATED_FILES: set[str] = set()
EXCLUDED_DIRECTORIES = {
    "__tests__",
    ".astro",
    ".vite",
    "build",
    "coverage",
    "dist",
    "dist-ssr",
    "fixtures",
    "node_modules",
    "tests",
    "vendor",
}


Report = dict[str, Any]


def run_git(*args: str) -> bytes:
    return subprocess.check_output(["git", *args], cwd=ROOT, stderr=subprocess.DEVNULL)


def is_generated_migration(path: str) -> bool:
    if not path.startswith("apps/backend/database/migrations/"):
        return False
    name = Path(path).name
    return "_collections_snapshot.go" in name or any(
        marker in name for marker in ("_updated_", "_created_", "_deleted_")
    )


def is_test(path: str) -> bool:
    file = Path(path)
    if set(file.parts) & {"__tests__", "fixtures", "tests"}:
        return True
    if file.name.endswith("_test.go"):
        return True
    return any(file.name.endswith(f".{kind}{suffix}") for kind in ("test", "spec") for suffix in SOURCE_SUFFIXES)


def is_source(path: str) -> bool:
    file = Path(path)
    return (
        file.suffix in SOURCE_SUFFIXES
        and path.startswith(SOURCE_ROOTS)
        and not set(file.parts) & EXCLUDED_DIRECTORIES
        and path not in GENERATED_FILES
        and not is_generated_migration(path)
        and not is_test(path)
    )


def tracked_sources() -> list[str]:
    paths = run_git("ls-files", "-z").decode().split("\0")
    return sorted(path for path in paths if path and is_source(path))


def function_ids(functions: list[Any]) -> list[tuple[str, Any]]:
    counts: defaultdict[str, int] = defaultdict(int)
    identified = []
    for function in sorted(functions, key=lambda item: (item.start_line, item.long_name)):
        signature = function.long_name.strip() or function.name.strip() or "<anonymous>"
        counts[signature] += 1
        identified.append((f"{signature}#{counts[signature]}", function))
    return identified


def measure() -> Report:
    files: dict[str, Any] = {}
    for relative_path in tracked_sources():
        result = lizard.analyze_file(str(ROOT / relative_path))
        functions = {}
        for function_id, function in function_ids(result.function_list):
            if function.cyclomatic_complexity <= CCN_LIMIT:
                continue
            functions[function_id] = function.cyclomatic_complexity
        entry = {}
        if result.nloc > NLOC_LIMIT:
            entry["nloc"] = result.nloc
        if functions:
            entry["functions"] = dict(sorted(functions.items()))
        if entry:
            files[relative_path] = entry
    report = {
        "version": 1,
        "tool": TOOL,
        "limits": {"ccn": CCN_LIMIT, "nloc": NLOC_LIMIT},
        "files": dict(sorted(files.items())),
    }
    return report


def load_baseline(path: Path = BASELINE_PATH) -> Report:
    if not path.exists():
        raise SystemExit(f"Missing {path.relative_to(ROOT)}. Run '{Path(__file__).relative_to(ROOT)} update'.")
    return json.loads(path.read_text())


def load_base_baseline(ref: str) -> Report | None:
    baseline = BASELINE_PATH.relative_to(ROOT).as_posix()
    try:
        run_git("rev-parse", "--verify", f"{ref}^{{commit}}")
    except subprocess.CalledProcessError as error:
        raise SystemExit(f"Unknown Git base ref: {ref}") from error
    try:
        raw = run_git("show", f"{ref}:{baseline}")
    except subprocess.CalledProcessError:
        return None
    return json.loads(raw)


def baseline_differences(stored: Report, current: Report) -> list[str]:
    if stored == current:
        return []
    differences = []
    stored_files = stored.get("files", {})
    current_files = current["files"]
    for path in sorted(set(stored_files) | set(current_files)):
        if stored_files.get(path) != current_files.get(path):
            differences.append(path)
    if any(stored.get(key) != current[key] for key in ("version", "tool", "limits")):
        differences.insert(0, "budget metadata")
    return differences


def increase_errors(base: Report, current: Report) -> list[str]:
    errors = []
    if current["version"] != base.get("version"):
        errors.append(f"baseline format changed from {base.get('version')} to {current['version']}")
    if current["tool"] != base.get("tool"):
        errors.append(f"analyzer changed from {base.get('tool')} to {current['tool']}")
    for metric, limit in current["limits"].items():
        old_limit = base.get("limits", {}).get(metric)
        if old_limit is None or limit > old_limit:
            errors.append(f"{metric} budget increased from {old_limit} to {limit}")
    base_files = base.get("files", {})
    for path, entry in current["files"].items():
        previous = base_files.get(path, {})
        if "nloc" in entry:
            old_nloc = previous.get("nloc")
            if old_nloc is None:
                errors.append(f"{path}: NLOC {entry['nloc']} exceeds {NLOC_LIMIT} in a new baseline entry")
            elif entry["nloc"] > old_nloc:
                errors.append(f"{path}: NLOC increased from {old_nloc} to {entry['nloc']}")
        old_functions = previous.get("functions", {})
        for function_id, ccn in entry.get("functions", {}).items():
            old_ccn = old_functions.get(function_id)
            if old_ccn is None:
                errors.append(f"{path}: {function_id} has new CCN {ccn}, above {CCN_LIMIT}")
            elif ccn > old_ccn:
                errors.append(f"{path}: {function_id} CCN increased from {old_ccn} to {ccn}")
    return errors


def print_summary(report: Report) -> None:
    files = report["files"]
    function_count = sum(len(entry.get("functions", {})) for entry in files.values())
    nloc_count = sum("nloc" in entry for entry in files.values())
    print(f"Code budget baseline: {function_count} complex functions and {nloc_count} oversized files.")


def update() -> int:
    report = measure()
    BASELINE_PATH.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    print_summary(report)
    return 0


def check(base_ref: str | None) -> int:
    current = measure()
    stored = load_baseline()
    differences = baseline_differences(stored, current)
    if differences:
        print("The code budget baseline is stale. Run './tools/code_quality/check.py update'.", file=sys.stderr)
        for difference in differences[:25]:
            print(f"  {difference}", file=sys.stderr)
        return 1
    if base_ref:
        base = load_base_baseline(base_ref)
        if base:
            errors = increase_errors(base, current)
            if errors:
                print(f"Code budgets regressed relative to {base_ref}:", file=sys.stderr)
                for error in errors:
                    print(f"  {error}", file=sys.stderr)
                return 1
        else:
            print(f"No baseline exists at {base_ref}; accepting the initial baseline.")
    print_summary(current)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("check", "update"))
    parser.add_argument("--base", help="Git ref whose baseline must not increase")
    args = parser.parse_args()
    if args.command == "update":
        return update()
    return check(args.base)


if __name__ == "__main__":
    raise SystemExit(main())
