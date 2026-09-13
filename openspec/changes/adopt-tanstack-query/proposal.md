## Why

The frontend duplicates server-state loading, error, refresh, and mutation orchestration across authentication and scheduling views. TanStack Query can make cache ownership and invalidation explicit while preserving isolated teacher and learner sessions.

## What Changes

- Add TanStack Query as the frontend server-state authority.
- Define all query keys and their invalidation rules in one query-key registry.
- Move teacher and learner session bootstrap to persona-scoped queries.
- Move login, logout, scheduling writes, and lesson lifecycle writes to mutations.
- Replace manual calendar and slot loading state with query state.
- Invalidate only the persona, calendar, slot, or session data affected by each successful mutation.
- Keep one low-level Fetch API transport boundary with credentials and intent headers.
- Preserve Polish scheduling errors, HttpOnly cookie sessions, UTC localization, and route behavior.

## Capabilities

### New Capabilities

- `frontend-query-state`: Defines query ownership, persona isolation, query keys, invalidation, mutations, and transport boundaries for frontend server state.

### Modified Capabilities

None.

## Impact

- Adds `@tanstack/react-query` to `apps/app`.
- Adds a root `QueryClientProvider` and one application `QueryClient`.
- Replaces custom calendar loading and auth bootstrap coordination in frontend views and route guards.
- Refactors `apps/app/src/api`, `apps/app/src/auth`, and scheduling components.
- Does not change backend endpoints, payloads, cookies, UTC persistence, or authorization rules.
