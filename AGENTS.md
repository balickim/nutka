# AI Engineering Instructions

You are a senior engineer writing composable and loosely coupled code.

## Priorities

- Readability and clarity over cleverness.
- Maintainability and explicitness over magic.
- Low coupling and high cohesion.
- Stable behavior and clear boundaries over hidden complexity.

## Instruction Order

When instructions overlap, follow this order:

1. `docs/AGENTS.md`
2. The most relevant constitution file(s)
3. `docs/README.md`

## Required Workflow

- Read `docs/README.md` and any relevant constitution files before changing code.
- When you change flows, data contracts, authorization assumptions, shared APIs, or important UI behavior, update the relevant constitution file(s).
- Keep code, docs, and constitutions aligned.
- When resolving a `docs/product-debt/*.md` or `docs/tech-debt/*.md` item, set `status: resolved`, add `resolved: <date>`, and move the file to that debt folder's `archive/` directory.

## Coding Guardrails

- Avoid unnecessary abstractions, premature optimization, and deeply nested control flow.
- Prefer pure functions and keep side effects explicit.
- Prefer improving the shape of the system over patching the nearest call site when the issue is structural.
- When logic is repeated prefer extracting a single authoritative abstraction or source of truth.
- Optimize for composability.
- Keep orchestration separate from business rules and policy decisions.
- Reduce coupling by depending on stable interfaces and shared domain primitives rather than spreading the same knowledge across many files.
- Prefer solutions that make the next related change easier, not just the current change pass.
- Do not introduce backward-compatibility layers, fallbacks, or legacy-preserving behavior unless explicitly requested.
- Don’t add defense-in-depth checks for things TypeScript already guarantees.
- Persist datetime values as UTC instants by default; convert to marina/user-local dates or times only at UI/read boundaries.
- If you find a touched persistence path storing dates or datetimes in a non-UTC form without an explicit constitution-backed exception, flag it as a finding and align it if the task scope allows.

## Code Budgets

- Apply these budgets to hand-written, non-test JavaScript, TypeScript, and Go.
- Keep cyclomatic complexity (CCN) at 10 or less per function.
- Keep NLOC at 250 or less per file.
- Do not increase an existing baseline or raise a budget.

## Verification

- After each batch of backend changes, run a backend build to catch compile errors.
- After UI changes, run the most relevant tests or checks for the touched area.
- Do not finish a change with broken types, failing builds, or stale constitutions.
- Run `./tools/code_quality/check.py check` after changing hand-written, non-test JavaScript, TypeScript, or Go.
- `pnpm audit --audit-level=high --prod` must pass with zero advisories. The `Frontend Build` CI check enforces this on every PR. When adding or updating dependencies, run the audit locally first and resolve any high/critical findings before pushing.

## Code Review Graph (MCP)

The `code-review-graph` MCP server maintains a persistent knowledge graph of this repo used for impact analysis, architecture overviews, and review context. A `PostToolUse` hook in `.claude/settings.local.json` runs an incremental update after every Write/Edit.

Slash commands:

- `/graph:stats` — node/edge counts, languages, last update time.
- `/graph:rebuild` — full rebuild. Run after a large merge from `main` (incremental diffs against `HEAD~1`, so big merges leave the graph stale).

Updating the server itself: `uvx --refresh code-review-graph serve` forces uv to pull the latest version from PyPI.
