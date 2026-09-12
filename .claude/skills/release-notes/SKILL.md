---
name: release-notes
description: Writes human-readable, user-facing release notes for MarinaKeeper from a release-please release PR or a tag range, in any requested language, using the product's own terminology from sailormoon.ui/src/locale/*.json. Use when the user asks for release notes, changelog prose, "what shipped in this release", notes for a "chore(main): release x.y.z" PR, or a translated version of release notes.
---

# Release notes

Turn a release's commits into notes a marina operator can read. The generated
CHANGELOG is a list of commit subjects; these notes explain what changed and why
it matters.

## Workflow

1. **Collect the raw material.**
   Run from the repo root; `$SKILL` is this skill's directory.
   ```bash
   $SKILL/scripts/release-changes.sh --pr 388     # release-please PR ("chore(main): release 0.52.0")
   $SKILL/scripts/release-changes.sh --range v0.51.0..HEAD
   ```
   Prints each change's commit subject, full body (squashed PRs keep their sub-commit
   messages — the best source of detail), and changed files.

2. **Read the diff where the subject is vague.** A subject like
   `feat(dates): add locale-aware date controls` does not say what a user sees. Check
   the new files, the new locale keys, and the validation messages before writing.

3. **Fix the terminology *before* drafting** — see [TERMINOLOGY.md](TERMINOLOGY.md).
   Every user-visible noun must match the UI in the target language:
   ```bash
   $SKILL/scripts/locale-terms.py pl_PL 'handoff|reconcil'   # find by key path
   $SKILL/scripts/locale-terms.py pl_PL 'armator' --values   # confirm a term is the one in use
   ```
   Never translate a feature name yourself when the UI already names it.

4. **Draft.** One `##` section per user-visible change, ordered by how much users
   care, not by commit order. Group pure-maintenance commits under `## Poprawki` /
   `## Fixes`. Skip release plumbing (version bumps, baselines, CI ratchets) unless it
   changes behaviour.

5. **Ask before publishing.** Offer to post as a PR comment (`gh pr comment`) or save
   to a file. Do not post unprompted.

## Writing rules

- Lead each section with what the user can now do, then the details that follow from it.
- Quote the actual UI string when naming a screen, tab, button, or validation message.
- Name the product **MarinaKeeper**. `sailormoon` is the repository, never user-facing.
- Cite each change as `(#<PR number>)`; keep the `v0.X.0 → v0.Y.0` compare link at the top.
- No marketing adjectives, no "we are excited". State the change.
- Internal refactors, code-budget baselines, and test-only changes do not get sections.
  A guardrail that prevents regressions can be one clause at the end of its section.

## Language

Default to the language the user asks in. The UI ships `en_GB`, `pl_PL`, `da_DK`,
`de_DE`, `nb_NO`, `sv_SE`, `uk_UA`, `ru_RU` — for any of these, source the terminology
from that locale file. For a language with no locale file, write the notes but say
which terms had no established translation.

See [EXAMPLES.md](EXAMPLES.md) for a finished set of notes and what each part is doing.
