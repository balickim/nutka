#!/usr/bin/env python3
"""Search a MarinaKeeper locale file so release notes reuse the product's own wording.

Usage:
  locale-terms.py <locale> <regex> [--values] [--limit N]

  <locale>  locale code or file, e.g. pl_PL, pl, src/locale/pl_PL.json
  <regex>   case-insensitive regex matched against the flattened key path,
            or against the translated text when --values is set

Examples:
  locale-terms.py pl_PL 'handoff|reconcil'          # keys about accounting handoffs
  locale-terms.py pl_PL '^translation\\.dashboard\\.setup\\.'
  locale-terms.py pl_PL 'armator' --values          # where a Polish term is used
"""

import argparse
import json
import pathlib
import re
import subprocess
import sys


def locale_dir() -> pathlib.Path:
    """Locale files live in the repo, so resolve them from the repo root, not the cwd."""
    root = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        capture_output=True, text=True, check=True,
    ).stdout.strip()
    return pathlib.Path(root) / "src/locale"


LOCALE_DIR = locale_dir()


def resolve(locale: str) -> pathlib.Path:
    path = pathlib.Path(locale)
    if path.is_file():
        return path
    candidate = LOCALE_DIR / f"{locale}.json"
    if candidate.is_file():
        return candidate
    matches = sorted(LOCALE_DIR.glob(f"{locale}*.json"))
    if matches:
        return matches[0]
    sys.exit(f"no locale file for {locale!r}; available: {sorted(p.stem for p in LOCALE_DIR.glob('*.json'))}")


def flatten(node, prefix=""):
    for key, value in node.items():
        path = f"{prefix}.{key}" if prefix else key
        if isinstance(value, dict):
            yield from flatten(value, path)
        else:
            yield path, value


def main() -> None:
    parser = argparse.ArgumentParser(add_help=True)
    parser.add_argument("locale")
    parser.add_argument("pattern")
    parser.add_argument("--values", action="store_true", help="match the translated text instead of the key")
    parser.add_argument("--limit", type=int, default=60)
    args = parser.parse_args()

    path = resolve(args.locale)
    pattern = re.compile(args.pattern, re.IGNORECASE)
    shown = 0
    for key, value in flatten(json.loads(path.read_text(encoding="utf-8"))):
        haystack = str(value) if args.values else key
        if not pattern.search(haystack):
            continue
        print(f"{key} = {value}")
        shown += 1
        if shown >= args.limit:
            print(f"... limit {args.limit} reached", file=sys.stderr)
            break
    if shown == 0:
        print("no matches", file=sys.stderr)


if __name__ == "__main__":
    main()
