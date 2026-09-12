#!/usr/bin/env bash
# Collect the raw material for release notes: every change in a release, with its
# commit message body and changed files.
#
# Usage:
#   release-changes.sh --pr <release-please PR number>
#   release-changes.sh --range <old-tag>..<new-ref>     e.g. v0.51.0..HEAD
#
# Reads the release-please PR body to find the commits, so it covers exactly what
# the release ships. Falls back to git log for an explicit range.
set -euo pipefail

mode=""
arg=""
case "${1:-}" in
  --pr)    mode=pr;    arg="${2:?missing PR number}" ;;
  --range) mode=range; arg="${2:?missing range}" ;;
  *) echo "usage: $0 --pr <number> | --range <old-tag>..<new-ref>" >&2; exit 2 ;;
esac

if [[ $mode == pr ]]; then
  body=$(gh pr view "$arg" --json body --jq .body)
  printf '=== release-please PR #%s body ===\n%s\n\n' "$arg" "$body"
  # release-please links each entry as .../commit/<sha>
  shas=$(grep -oE 'commit/[0-9a-f]{7,40}' <<<"$body" | cut -d/ -f2 | awk '!seen[$0]++')
else
  shas=$(git log --format=%h --no-merges "$arg" | awk '!seen[$0]++')
fi

[[ -n ${shas:-} ]] || { echo "no commits found" >&2; exit 1; }

while read -r sha; do
  [[ -n $sha ]] || continue
  printf '\n=== %s ===\n' "$sha"
  git log -1 --format='%s%n%n%b' "$sha"
  printf -- '--- changed files ---\n'
  git show --stat --format='' "$sha"
done <<<"$shas"
