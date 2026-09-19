# Frontend query state constitution

This document defines how the Nutka React application owns server state.
It does not change any backend contract in `docs/api/`.

## Authority

- The shared TanStack Query cache is the single authority for frontend server state.
- Every authentication, policy, scheduling, commercial, payment, history, and material read uses a query.
- Every authentication, scheduling, commercial, payment, correction, and material write uses a mutation.
- The application creates one QueryClient in `apps/app/src/query/client.ts`.
- One QueryClientProvider wraps the router.
- Route guards and components use the same QueryClient.
- No module keeps a second copy of session, calendar, slot, plan, or ledger data.

## Query keys

- The file `apps/app/src/query/keys.ts` defines every query key.
- No other file builds a query-key array.
- Each key contains every variable that changes returned data.

| Data | Key |
| --- | --- |
| Persona session | `["nutka", role, "session"]` |
| Authenticated policy | `["nutka", role, "policy"]` |
| Persona calendar | `["nutka", role, "calendar", accountId]` |
| Learner assignment slots | `["nutka", "learner", "slots", learnerId, assignmentId]` |
| Assignment commercial summary | `["nutka", role, "assignment-summary", assignmentId]` |
| Assignment packages | `["nutka", role, "assignment-summary", accountId, assignmentId, "packages"]` |
| Assignment contracts | `["nutka", role, "assignment-summary", accountId, assignmentId, "contracts"]` |
| Teacher contract series | `["nutka", "teacher", "contract-series", assignmentId, contractId]` |
| Teacher contract months | `["nutka", "teacher", "contract-series", assignmentId, contractId, "months"]` |
| Financial work | `["nutka", role, "financial-work", accountId]` |
| Assignment history | `["nutka", role, "history", assignmentId]` |
| Teacher unresolved work | `["nutka", "teacher", "unresolved-work", teacherId]` |
| Assignment materials | `["nutka", role, "materials", accountId, assignmentId]` |

- The session key uses the role because the account is unknown before the first read.
- Each owned data key contains the authenticated account identifier or assignment identifier.
- A different account reads a different key for the same role.
- A learner key cannot address teacher-owned contract, financial, or history data.
- Policy data is keyed by role to prevent cross-realm cache reuse.
- A package key, a contract key, and a contract month key extend the key of the read they belong to.
- An extended key inherits the cache rule of its prefix.

## Cache rules

- The registry `queryRules` defines the cache effect of every mutation.
- A caller applies a rule with `applyCacheEffect` and never builds its own filter.
- A successful mutation awaits its rule before the mutation settles.

| Rule | Effect |
| --- | --- |
| `availabilityWrite` | Invalidates both calendar prefixes, every learner slot query, affected summaries, contract series, financial work, and history. |
| `assignmentWrite` | Invalidates both calendar prefixes, the changed summary, and the changed assignment slots. |
| `bookingWrite` | Invalidates both calendars, affected summary, slots, financial work, and history. |
| `lifecycleWrite` | Invalidates both calendars, affected summary, slots, financial work, unresolved work, and history. |
| `planWrite` | Invalidates both calendars, affected summary, slots, contract series, financial work, unresolved work, and history. |
| `outcomeWrite` | Invalidates calendars, affected summary, unresolved work, financial work, and history. |
| `settlementWrite` | Invalidates calendars, affected summary, financial work, unresolved work, and history. |
| `correctionWrite` | Invalidates calendars, affected summary, slots, contract series, financial work, unresolved work, and history. |
| `materialWrite` | Invalidates the teacher and learner materials of the changed assignment. |
| `personaCleared` | Cancels every request of one role and removes the owned data of that role. |

- A lesson change invalidates every learner slot query because conflicts cross assignments.
- A compound commercial mutation never uses an optimistic update.
- A successful compound mutation refreshes server-owned read models after commit.
- Logout, session expiry, cross-tab logout, and account replacement apply `personaCleared`.
- Each caller then writes the replacing session result into the session key.
- The rule keeps the session key so an active reader receives the replacement.
- A cleared persona does not affect the other persona.

## Transport boundary

- The file `apps/app/src/api/transport.ts` is the only module that calls the Fetch API.
- The transport builds the URL, sends `credentials: "include"`, and decodes JSON.
- The transport sends a `FormData` body as multipart and lets the browser set its content type.
- The transport sends `X-Requested-With: fetch` on every mutation.
- The transport forwards the AbortSignal supplied by the query cache.
- The transport raises ApiRequestError with the stable error code and HTTP status.
- Endpoint functions return promises and hold no cache knowledge.

## Retries

- A read retries at most one time.
- A read retries only a network failure or a `5xx` response.
- A read does not retry a `4xx` response.
- A mutation never retries.

## Session lifecycle

- The session query returns `authenticated` or `unauthenticated` data.
- A `401` from the session endpoint is unauthenticated data and not an error.
- A network failure or a `5xx` response is a query error.
- The session query stays fresh until a lifecycle effect replaces it.
- Every resolved session result arms or clears the expiry timer of its role.
- The file `apps/app/src/auth/lifecycle.ts` owns expiry timers and cross-tab channels.
- The lifecycle module holds no account data and publishes event types only.
- Logout clears persona cache even when the server request fails.
- The frontend signs out at the server-reported expiry or 12 hours after login without one.

## Presentation

- A panel renders loading state only when the query is pending and holds no data.
- A background refresh keeps the last successful data until refresh settles.
- A failed request renders the existing Polish message for its stable code.
- Frontend maps English plans, states, outcomes, event types, and errors to Polish copy.
- Frontend sends English machine values and never persists translated values.
- Policy values drive price, duration, grid, buffer, timing, horizon, package, and contract controls.
