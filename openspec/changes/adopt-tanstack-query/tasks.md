## 1. Query foundation

- [ ] 1.1 Add `@tanstack/react-query` to `apps/app`, update the lockfile, and confirm the production dependency audit remains clean.
- [ ] 1.2 Create one application QueryClient with query retry rules and disabled mutation retries.
- [ ] 1.3 Mount one QueryClientProvider above the router and expose the same QueryClient to route guards.
- [ ] 1.4 Create one transport module for URL construction, credentials, intent headers, JSON decoding, typed errors, and AbortSignal forwarding.
- [ ] 1.5 Add transport tests for read cancellation, mutation headers, cookie credentials, JSON responses, `4xx`, `5xx`, and network failures.

## 2. Query keys and cache rules

- [ ] 2.1 Create `src/query/keys.ts` with role, session, account calendar, learner slot, and stable prefix factories.
- [ ] 2.2 Add pure rules for availability, assignment, lesson, account replacement, logout, expiry, and `401` cache effects.
- [ ] 2.3 Add registry tests for deterministic keys, persona isolation, account isolation, assignment variables, and every mutation effect.
- [ ] 2.4 Remove inline query-key arrays outside the registry and verify every query obtains its key from that file.

## 3. Authentication migration

- [ ] 3.1 Move teacher and learner session requests, login, and logout onto the shared transport boundary.
- [ ] 3.2 Add persona session query options and use the shared QueryClient in route guards and rendered session consumers.
- [ ] 3.3 Add login and logout mutations that replace, cancel, or remove only the matching persona cache.
- [ ] 3.4 Adapt expiry timers and BroadcastChannel events to QueryClient state without storing account data separately.
- [ ] 3.5 Preserve unauthenticated `401`, retryable network and `5xx` behavior, safe redirects, and logout cleanup after request failure.
- [ ] 3.6 Remove the custom auth store, subscriptions, bootstrap promise, and view adapters after all consumers migrate.
- [ ] 3.7 Update auth and route tests for shared request deduplication, dual-session isolation, account replacement, expiry, and cross-tab logout.

## 4. Scheduling query migration

- [ ] 4.1 Add teacher and learner calendar query options that include the authenticated account identifier.
- [ ] 4.2 Add learner slot query options and parallel active-assignment queries keyed by learner and assignment identifiers.
- [ ] 4.3 Replace teacher and learner manual loader state, refresh callbacks, and initial effects with query state.
- [ ] 4.4 Add assignment and availability mutations that await their central invalidation rules.
- [ ] 4.5 Add booking, reschedule, and cancellation mutations that invalidate both calendar prefixes and all learner slots.
- [ ] 4.6 Preserve existing loading, retry, empty, Polish error, busy, and background-refresh presentation.
- [ ] 4.7 Update scheduling unit and component tests for cached data retention, mutation failures, and cache refresh behavior.

## 5. Cleanup and documentation

- [ ] 5.1 Remove direct Fetch API calls outside the transport module and remove obsolete request and loading helpers.
- [ ] 5.2 Update the authentication lifecycle documentation to identify the shared query cache as the frontend session authority.
- [ ] 5.3 Add a frontend query-state document for the key hierarchy, cache rules, transport boundary, retries, and persona cleanup.
- [ ] 5.4 Link the query-state document from `docs/README.md` without changing backend API contracts.

## 6. Verification

- [ ] 6.1 Run focused unit tests for transport, query keys, auth lifecycle, scheduling queries, and mutation invalidation.
- [ ] 6.2 Run the complete frontend unit suite and Playwright teacher–learner lifecycle suite.
- [ ] 6.3 Run the monorepo check, production build, code-quality budget check, and `git diff --check`.
- [ ] 6.4 Run strict OpenSpec validation and `npm audit --audit-level=high --omit=dev` with zero high or critical advisories.
