# AI Engineering Instructions

## Documentation Workflow

- Read `docs/README.md` and the relevant constitution files before changing code; `docs/README.md` takes precedence when they conflict.
- When you change flows, data contracts, authorization assumptions, shared APIs, or important UI behavior, update the relevant constitution file(s).
- When resolving a `docs/product-debt/*.md` or `docs/tech-debt/*.md` item, set `status: resolved`, add `resolved: <date>`, and move the file to that debt folder's `archive/` directory.

## Documentation Style

Constitutions and API references use a compressed, controlled-language style derived from ASD-STE100. Match it when you edit them.

- One rule per bullet. Cap a bullet at about 25 words; split rather than extend.
- Active voice, simple tenses. No phrasal verbs — "remove", not "take off".
- State the rule; do not argue for it. Rationale belongs in `docs/adr/`, operations in runbooks.
- One term per concept, every time. Do not rotate synonyms.
- No nominalization — "analyze", not "perform an analysis of". No marketing adjectives.
- Noun clusters stay at three words or fewer.
- Give each endpoint one canonical page in `docs/api/`; constitutions link to it rather than restate it.
- Constitutions and API reference pages contain no semicolons.
- Constitutions and API reference pages use explicit subjects, verbs, and articles.
- Do not claim dictionary compliance.
- Write each Markdown document so readers can understand it without prior conversations or threads.

## Design Rules

- Prefer the smallest readable, explicit implementation and structural fixes. Avoid cleverness, deep nesting, premature optimization, and patches that preserve structural problems.
- Apply YAGNI: add only capabilities required by the current task and established contracts, including options, extension points, and frameworks.
- YAGNI does not defer security, data integrity, tests, or refactoring required for the current change.
- Before adding code, inspect existing implementations and documented contracts. Reuse components and services whose responsibilities match the requirement.
- Share code for the same rule and reason to change. Keep similar code separate when it represents different rules.
- Keep one authoritative implementation of each rule. Consumers call it instead of duplicating its knowledge.
- Before extraction, simplify logic and remove unnecessary lines, branches, states, and abstractions.
- Do not relocate complexity into helpers or files to satisfy metrics.
- Use composable components with explicit dependencies, outputs usable as other components' inputs, and tests independent of unrelated application state.
- Depend only on stable public contracts so implementation changes do not require consumer changes.
- Expose only interfaces consumers need, without framework, storage, or transport details.
- Keep business rules pure. Place orchestration, persistence, network access, time, and other side effects at explicit boundaries.
- Add compatibility layers, fallbacks, or legacy behavior only when explicitly requested.
- Do not check guarantees already enforced by TypeScript.
- Use comments only as a last resort for non-obvious intent, invariants, tradeoffs, or external constraints. Place each above related code on one physical line.
- Begin each new or modified hand-written, non-test source file with a purpose-and-boundary overview (maximum 40 words), after required shebangs, build tags, directives, or legal headers.
- Do not add overviews to untouched files or automate overview and comment checks.
- Persist datetimes as UTC instants by default. Convert to marina/user-local dates or times only at UI/read boundaries.
- Flag non-UTC date or datetime persistence in touched paths unless a constitution explicitly permits it. Align it with UTC when scope allows.

## Testing Rules

- Add tests only for core business rules, meaningful regressions, authorization, data integrity, and critical user flows.
- Before adding a test, inspect existing coverage and identify the distinct rule or failure it protects. Extend an existing test when possible.
- Prefer backend tests at the lowest layer that can reproduce the failure. Use database or HTTP tests only when that boundary matters.
- Do not repeat the same scenario across unit, integration, and browser tests unless each layer protects a distinct failure.
- Do not add tests for trivial wiring, framework behavior, getters, implementation details, or guarantees enforced by types.
- Do not add tests merely because a file or function changed, or to increase coverage percentages.
- Keep fixtures minimal. Reuse established database templates instead of replaying migrations for each test.

## Code Budgets

- Apply these budgets to hand-written, non-test JavaScript, TypeScript, and Go.
- Keep cyclomatic complexity (CCN) at 10 or less per function.
- Keep NLOC at 250 or less per file.
- Do not increase an existing baseline or raise a budget.

## Verification

- Build after backend changes and run the most relevant checks after UI changes. Do not finish with broken types, failing builds, or stale constitutions.
- Run `./tools/code_quality/check.py check` after changing hand-written, non-test JavaScript, TypeScript, or Go.
- `pnpm audit --audit-level=high --prod` must pass with zero advisories. The `Frontend Build` CI check enforces this on every PR. When adding or updating dependencies, run the audit locally first and resolve any high/critical findings before pushing.

## Code Review Graph (MCP)

- Use `code-review-graph` for impact analysis and review context when available. Run `/graph:rebuild` after a large merge from `main`; use `/graph:stats` to check freshness.
- Refresh the server with `uvx --refresh code-review-graph serve`.
