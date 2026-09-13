## Context

The React app currently owns server state in a custom authentication store and view-level `useEffect` loaders. Two modules call the Fetch API directly. See `proposal.md` for the motivation and `specs/frontend-query-state/spec.md` for required behavior.

Teacher and learner sessions can coexist in one browser. Their HttpOnly cookies, expiry timers, and cross-tab logout events must remain independent. Scheduling writes can affect calendars and slot availability across both personas.

## Goals / Non-Goals

**Goals:**

- Use one QueryClient for all frontend server state.
- Make cache identity and mutation effects explicit and testable.
- Remove duplicate loading, error, refresh, and single-flight state.
- Preserve route guards, session expiry, localized errors, and current UI states.

**Non-Goals:**

- Change backend routes, DTOs, authorization, or cookie behavior.
- Add optimistic scheduling writes.
- Persist, broadcast, or hydrate the query cache.
- Add pagination, offline mutation queues, or a query-key factory dependency.
- Replace Formik or TanStack Router.

## Decisions

### Use one application QueryClient

Create one QueryClient in `src/query/client.ts`. Mount one `QueryClientProvider` above the router. Route guards and React hooks use the same client, so concurrent requests share one cache and one promise.

Use explicit query options for session, calendar, and slot reads. Use mutation hooks for login, logout, assignment changes, availability changes, booking, rescheduling, and cancellation.

Alternative: keep the auth store beside TanStack Query. Rejected because two server-state authorities can disagree about authentication and duplicate request coordination.

### Keep keys and cache rules in one pure registry

Create `src/query/keys.ts` as the only source of query-key arrays and mutation cache rules. Do not place inline query-key arrays in views, hooks, route guards, or tests outside registry tests.

Use this hierarchy:

```text
["nutka", role, "session"]
["nutka", role, "calendar", accountId]
["nutka", "learner", "slots", learnerId, assignmentId]
```

Place stable prefixes and exact-key factories in `queryKeys`. Place parameterized invalidation, cancellation, and removal filters in `queryRules`. Keep the registry free of network and React dependencies.

The registry defines these mutation effects:

| Effect | Invalidate | Remove |
| --- | --- | --- |
| Teacher availability write | Both calendar prefixes and all learner slots | None |
| Teacher assignment write | Both calendar prefixes and the assignment slot key | None |
| Lesson booking or lifecycle write | Both calendar prefixes and all learner slots | None |
| Persona replacement | None | The previous cache for that role |
| Persona logout, expiry, or `401` | None | The matching role prefix after cancellation |

Alternative: colocate keys with each feature. Rejected because the requested single authority would disappear and cross-feature invalidation would remain implicit.

### Include the account identifier in owned data keys

Calendar and slot keys include the authenticated account identifier. The session key uses the role because the account is unknown before bootstrap. Login clears the existing role prefix before writing the new session result.

This hierarchy provides defense in depth against showing cached data after switching accounts. Logout first cancels role-scoped requests and then removes the role prefix, preventing an old in-flight request from restoring removed data.

Alternative: key owned data by role only. Rejected because correctness would depend entirely on perfect logout cleanup.

### Keep one Fetch API transport boundary

Create `src/api/transport.ts` as the only module that calls `fetch`. It owns URL construction, `credentials: "include"`, mutation intent headers, JSON decoding, and typed HTTP errors. It accepts an AbortSignal from query functions.

Endpoint functions remain plain promise-returning functions. Query options and mutation hooks are their only application consumers. TanStack Query owns execution, caching, retries, cancellation, and invalidation rather than replacing the browser transport.

Alternative: call `fetch` inside every query function. Rejected because credentials, headers, decoding, and error handling would duplicate.

### Model authentication as query data plus lifecycle effects

Define one session query per persona. Route guards use QueryClient fetch APIs with the same query options that components use. Login writes the returned session data into the matching key. Logout clears local cache in `finally`, even if the server request fails.

Keep expiry timers and BroadcastChannel listeners in a small session lifecycle module. They operate on QueryClient keys and publish event types only. They do not own a second copy of account data.

A `401` is an unauthenticated session result, not a retryable error. Network and `5xx` failures remain errors. Queries retry a transient failure at most once. Mutations never retry automatically.

Alternative: treat every `401` as a thrown query error. Rejected because unauthenticated state is an expected route decision, not a retryable failure.

### Derive panels from query and mutation state

Teacher views use the teacher calendar query. Learner views use the learner calendar query and parallel slot queries for active assignments. The slot query uses both learner and assignment identifiers and stays disabled without an authenticated learner or active assignment.

Successful mutations apply `queryRules` and await relevant invalidation before settling. Initial pending states use existing loading UI. Background fetching retains cached content and does not replace the panel with an empty loading screen.

Avoid optimistic updates for scheduling. The server atomically revalidates availability, participant conflicts, duration, buffers, and horizon, so the response and subsequent query refresh remain authoritative.

## Risks / Trade-offs

- [Broad lesson invalidation can refetch several slot queries] → Participant conflicts cross assignments, so correctness takes precedence over narrower invalidation.
- [Two active personas share one QueryClient] → Every owned key includes its role and account identifier.
- [Expired cache data could appear after account replacement] → Cancel and remove the role prefix before storing the new session.
- [Route guards can duplicate component requests] → Both use the same query key and QueryClient.
- [Default retries could repeat deterministic failures] → Central retry policy rejects `4xx` responses and disables mutation retries.
- [Migration can change loading timing] → Preserve existing visible states and cover route, auth, and scheduling flows with existing Playwright tests.

## Migration Plan

1. Add TanStack Query and mount the application QueryClient provider.
2. Add the transport boundary and the tested query-key registry.
3. Migrate session reads, login, logout, expiry, cross-tab events, and route guards.
4. Migrate calendar and slot reads to query options and hooks.
5. Migrate scheduling writes to mutation hooks and central cache rules.
6. Remove the custom auth store, view loaders, direct Fetch API calls, and obsolete tests.
7. Run unit, route, E2E, build, quality, OpenSpec, and production audit checks.

Rollback reverts the frontend change. No backend or database rollback is required.
