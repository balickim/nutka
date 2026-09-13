## 1. Schema and migration

- [x] 1.1 Inspect the existing PocketBase migration and auth collection fields, then define the idempotent migration preconditions for `users`, `teachers`, and `learners`.
- [x] 1.2 Implement the auth migration that renames `users` to `teachers` with identifiers and credentials preserved, creates `teachers` on a fresh database, and rejects incompatible dual collections.
- [x] 1.3 Add the teacher IANA timezone field with validation and set the schema and migration default to `Europe/Warsaw`.
- [x] 1.4 Add `teacher_learners`, `availability_rules`, `availability_exceptions`, `lessons`, and `lesson_events` collections with ownership relations, validation, indexes, and retained-history fields.
- [x] 1.5 Add migration tests for fresh databases, existing `users`, reruns, preserved records, and incompatible `users` plus `teachers` collections.

## 2. Authentication and authorization

- [x] 2.1 Extract shared session-cookie behavior from the existing learner auth flow without changing its verified-account, intent-header, expiry, or no-token guarantees.
- [x] 2.2 Add teacher password authentication, session bootstrap, logout, and role-specific middleware using `__Host-nutka_teacher_session`.
- [x] 2.3 Move learner session handling to `__Host-nutka_learner_session` and preserve independent learner behavior using the existing mechanism as the authority.
- [x] 2.4 Add wrong-realm route and API guards, including the native PocketBase collection route guard and caller-identity resolution from matching cookies.
- [x] 2.5 Verify that no public registration, invitation, token response, token storage, or cross-persona authorization path exists.
- [x] 2.6 Update `docs/auth.md` with both realms, cookie names, route spaces, custom session endpoints, and closed-account scope.

## 3. Scheduling domain rules

- [x] 3.1 Implement pure interval, duration, and five-minute protected-buffer value operations using half-open UTC intervals.
- [x] 3.2 Implement teacher-timezone recurring-rule expansion with a documented deterministic policy for nonexistent and ambiguous DST wall-clock times.
- [x] 3.3 Implement available and unavailable exception merging with unavailable precedence and UTC conversion at the boundary.
- [x] 3.4 Implement fifteen-minute slot generation for the rolling fourteen-day horizon, excluding past starts and candidates whose lesson interval does not fit effective availability.
- [x] 3.5 Implement assignment default validation and teacher-only lesson-duration override validation for positive multiples of fifteen minutes.
- [x] 3.6 Implement teacher and learner protected-interval conflict detection, including buffers that extend beyond availability windows.
- [x] 3.7 Add focused domain tests for defaults, exception precedence, grid alignment, horizon edges, DST transitions, durations, buffers, and conflicts.

## 4. Assignment and availability APIs

- [x] 4.1 Add authorized assignment query and mutation APIs for the many-to-many teacher–learner relationship, including teacher-controlled default duration.
- [x] 4.2 Add authorized teacher availability rule and exception APIs with local recurring fields and UTC concrete exception fields.
- [x] 4.3 Reject unavailable-interval mutations that conflict with scheduled lesson intervals in one transaction, while keeping buffers in participant conflict checks only.
- [x] 4.4 Add teacher and learner calendar read APIs that return only authorized assignments, active availability, retained lessons, and RFC3339 UTC instants.
- [x] 4.5 Add stable JSON errors for unauthenticated, unauthorized, invalid-duration, invalid-grid, horizon, and conflict outcomes.

## 5. Booking and lesson lifecycle APIs

- [x] 5.1 Implement atomic learner booking that always derives duration from the active assignment, rejects learner duration fields, and re-reads availability, horizon, and both participant calendars before writing the lesson and creation event.
- [x] 5.2 Implement atomic future rescheduling for either participant, allowing only teachers to change duration while learners retain it, with complete revalidation, prior and new interval snapshots, and a reschedule event.
- [x] 5.3 Implement atomic future cancellation for either participant with retained status, initiator role and ID, cancellation UTC instant, and cancellation event.
- [x] 5.4 Reject mutations for started or past lessons and exclude cancelled lessons from active conflict queries.
- [x] 5.5 Implement initiator-attributed cancellation counters and expose them through both dashboard API views without counting reschedules.
- [x] 5.6 Add backend HTTP and transaction tests for assignment authorization, concurrent booking, participant conflicts, lifecycle immutability, retained history, and counters.

## 6. Frontend route spaces and panels

- [x] 6.1 Split frontend auth state into independent teacher and learner stores with separate bootstrap, expiry, logout, and BroadcastChannel lifecycles.
- [x] 6.2 Add `/learners/login`, `/learners/calendar`, `/teachers/login`, `/teachers`, and `/teachers/availability` routes with matching guards and safe redirects.
- [x] 6.3 Add shared scheduling API types and UTC parsing helpers that keep timezone conversion at display boundaries.
- [x] 6.4 Build the teacher panel for weekly availability, exceptions, future lessons, duration changes, reschedule, cancellation, and teacher cancellation counter.
- [x] 6.5 Build the learner panel for assigned teachers, fourteen-day slots, booking with assignment defaults, future reschedule without duration controls, cancellation, and learner cancellation counter.
- [x] 6.6 Render API errors and loading states without exposing collection internals, tokens, or public registration controls.
- [x] 6.7 Add frontend unit tests for independent auth state, route guards, UTC localization, horizon display, and initiator counters.

## 7. Documentation and verification

- [x] 7.1 Add the scheduling constitution and canonical API reference pages for availability, assignments, booking, lifecycle, and calendar reads.
- [x] 7.2 Add the UTC persistence and recurring wall-clock timezone exception to the relevant documentation with consistent terminology.
- [x] 7.3 Add the critical Playwright flow covering teacher setup, learner booking, independent sessions, reschedule, cancellation, and both counters.
- [x] 7.4 Run backend tests, frontend unit tests, the production build, code-quality checks, OpenSpec validation, and the production dependency audit.
