# Frontend query state constitution

This document defines how the Nutka React application owns server state.
It does not change any backend contract in `docs/api/`.

## Authority

- The shared TanStack Query cache is the single authority for frontend server state.
- Every authentication read and scheduling read uses a query.
- Every authentication write and scheduling write uses a mutation.
- The application creates one QueryClient in `apps/app/src/query/client.ts`.
- One `QueryClientProvider` wraps the router.
- Route guards and components use the same QueryClient, so they share one request.
- No module keeps a second copy of session, calendar, or slot data.

## Query keys

- The file `apps/app/src/query/keys.ts` defines every query key.
- No other file builds a query-key array.
- Each key contains every variable that changes the returned data.

| Data | Key |
| --- | --- |
| Persona session | `["nutka", role, "session"]` |
| Persona calendar | `["nutka", role, "calendar", accountId]` |
| Learner assignment slots | `["nutka", "learner", "slots", learnerId, assignmentId]` |

- The session key uses the role because the account is unknown before the first read.
- Each owned data key contains the authenticated account identifier.
- A different account for one role therefore reads a different key.

## Cache rules

- The registry `queryRules` defines the cache effect of every mutation.
- A caller applies a rule with `applyCacheEffect` and never builds its own filter.
- A successful mutation awaits its rule before the mutation settles.

| Rule | Effect |
| --- | --- |
| `availabilityWrite` | Invalidates both calendar prefixes and every learner slot query. |
| `assignmentWrite` | Invalidates both calendar prefixes and the slot query of the changed assignment. |
| `lessonWrite` | Invalidates both calendar prefixes and every learner slot query. |
| `personaCleared` | Cancels every request of one role and removes the owned data of that role. |

- A lesson change invalidates every learner slot query because a participant conflict crosses assignments.
- Logout, session expiry, cross-tab logout, and account replacement all apply `personaCleared`.
- Each caller then writes the replacing session result into the session key.
- The rule keeps the session key so an active reader receives the replacement instead of a new request.
- A cleared persona does not affect the other persona.

## Transport boundary

- The file `apps/app/src/api/transport.ts` is the only module that calls the Fetch API.
- The transport builds the URL, sends `credentials: "include"`, and decodes JSON.
- The transport sends the header `X-Requested-With: fetch` on every mutation.
- The transport forwards the AbortSignal that the query cache supplies.
- The transport raises `ApiRequestError` with the stable error code and the HTTP status.
- Endpoint functions return promises and hold no cache knowledge.

## Retries

- A read retries at most one time.
- A read retries only a network failure or a `5xx` response.
- A read does not retry a `4xx` response.
- A mutation never retries.

## Session lifecycle

- The session query returns `authenticated` or `unauthenticated` data.
- A `401` from the session endpoint is `unauthenticated` data and not an error.
- A network failure or a `5xx` response is a query error, and the panel offers a retry control.
- The session query stays fresh until a lifecycle effect replaces it.
- Every resolved session result arms or clears the expiry timer of its role.
- The file `apps/app/src/auth/lifecycle.ts` owns the expiry timers and the cross-tab channels.
- The lifecycle module holds no account data and publishes event types only.
- Logout clears the persona cache even when the server request fails.
- The frontend signs out at the server-reported expiry, or 12 hours after login without one.

## Presentation

- A panel renders its loading state only when the query is pending and holds no data.
- A background refresh keeps the last successful data until the refresh settles.
- A failed scheduling request renders the existing Polish message for its stable code.
- Scheduling writes use no optimistic update, because the server revalidates each request.
