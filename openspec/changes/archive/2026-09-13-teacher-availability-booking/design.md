## Context

The backend is a PocketBase application with one custom learner auth flow. It already uses a server-managed JWT cookie and disables native learner token refresh. The frontend has one learner auth store and one root/login route pair. No scheduling collections, API boundary, or time model exists yet. See `proposal.md` for motivation and the capability specs for the observable contract.

## Goals / Non-Goals

**Goals:**

- Keep teacher and learner authentication isolated while reusing the existing learner session mechanism.
- Represent recurring availability, dated exceptions, assignments, lessons, and immutable event history in PocketBase.
- Centralize slot, duration, buffer, horizon, timezone, and conflict rules in a backend domain package.
- Make every booking and lifecycle mutation atomic at the persistence boundary.
- Expose small authorized APIs that the two route spaces can consume without direct collection access.
- Provide focused teacher and learner calendar panels with independent in-memory auth stores.

**Non-Goals:**

- Public registration, invitation delivery, invitation management, password recovery, MFA, or account import.
- Payment, reminders, attendance, conferencing, recurring lesson records, or cancellation policy fees.
- General-purpose calendar integrations or a framework migration.

## Decisions

### Reuse the existing session bridge

Extend the existing PocketBase auth hooks and cookie helpers for a second realm. Add role-specific cookie names and `me` and `logout` handlers, but retain the current HttpOnly, no-token-response, intent-header, and verified-account behavior. Do not copy the Sailormoon implementation.

The request middleware resolves at most one matching persona per cookie. Scheduling handlers resolve identity from the cookie and ignore caller-supplied identity fields. Native PocketBase collection routes remain unavailable to non-superusers.

### Use migration-safe auth collection setup

The first migration checks the collection set in this order:

1. Rename custom `users` to `teachers` when `teachers` does not exist.
2. Create `teachers` with the same custom auth fields when neither collection exists.
3. Leave `learners` unchanged.

The migration preserves record IDs and credentials during rename. The `teachers` schema defaults a missing timezone to `Europe/Warsaw`. New and migrated teacher records accept only valid IANA timezone identifiers. It fails clearly if both `users` and `teachers` exist with incompatible data rather than guessing which realm is authoritative.

### Store scheduling data as explicit collections

Use these PocketBase collections:

- `teacher_learners`: teacher, learner, active state, and default duration. Enforce one row per pair and default the duration to 45 minutes.
- `availability_rules`: teacher, weekday, local start and end wall-clock values, and enabled state. Rules have no end date.
- `availability_exceptions`: teacher, UTC start and end, kind (`available` or `unavailable`), and optional note.
- `lessons`: teacher, learner, assignment, UTC start and end, duration, status, cancellation initiator role and ID, and cancellation UTC instant.
- `lesson_events`: lesson, event kind, initiator role and ID, event UTC instant, prior interval, new interval, and duration snapshots.

Use PocketBase relation fields for ownership and assignment links. Add indexes for teacher and learner lesson intervals, active assignments, and exception ranges. Treat cancelled lessons as retained records and exclude them from active conflict queries.
Only teacher handlers may change an assignment default or a future lesson duration. Learner booking and rescheduling payloads contain no duration control.

### Keep business rules pure and persistence orchestration explicit

The scheduling domain receives a clock, teacher timezone, recurring rules, exceptions, assignments, and active lessons. It returns slots or a typed validation or conflict result without HTTP or PocketBase dependencies. Handlers load the required records, call the domain, then persist inside one transaction.

Use half-open intervals `[start, end)` for lessons and buffers. A proposed lesson conflicts when its protected interval intersects another active protected interval. A five-minute buffer may extend outside a teacher availability window and participates only in participant conflict checks, while the lesson itself must fit inside the effective available interval.

### Resolve local recurring rules before converting to UTC

Store weekly rule times as local wall-clock values plus weekday and store the teacher's validated IANA timezone on the teacher record. Default missing values to `Europe/Warsaw`. Expand only the requested rolling horizon. Convert each occurrence to a UTC instant using one shared timezone resolver. The resolver documents deterministic handling for nonexistent and ambiguous local times and is covered by DST tests.

Dated exceptions store concrete UTC instants. An unavailable interval subtracts from effective availability after all available sources are merged. Concrete lesson and exception responses use RFC3339 UTC values. The client performs display localization only.

### Enforce atomic writes with a transaction and re-read

Booking and lifecycle mutations run in one PocketBase transaction. The transaction re-reads active assignment, target lesson, and conflicting active lessons before writing. It writes the lesson and its event in the same transaction. A failed validation or conflict rolls back all writes.

Availability mutations use the same transaction boundary. An unavailable exception is rejected if its interval intersects a scheduled lesson interval. Rescheduling validates the new interval before replacing the old one and appends a history event.

### Expose custom API resources

Register role-specific routes for session status, assignments, availability, slot queries, lesson lists, booking, rescheduling, cancellation, and cancellation counters. Route handlers return stable JSON errors for unauthenticated, unauthorized, validation, horizon, and conflict outcomes. They do not expose PocketBase collection rules as the public contract.

Use an intent header on browser mutations, consistent with the existing auth contract. Return UTC datetimes and enough lesson metadata for either panel to render status and buffers without direct collection reads.

### Separate frontend auth stores and route trees

Keep learner auth lifecycle semantics from the existing store while extracting shared cookie-session helpers. Create a teacher store with its own PocketBase auth store, bootstrap request, expiry timer, logout request, and BroadcastChannel name. Messages contain event types only.

Build route guards for `/learners/login`, `/learners/calendar`, `/teachers/login`, `/teachers`, and `/teachers/availability`. Keep `/` as the persona entry point or redirect. Each panel calls custom APIs and formats concrete UTC values with `Intl.DateTimeFormat`.

## Risks / Trade-offs

- [Risk] PocketBase collection rename behavior differs across existing installations. → Inspect collection type and fields before rename, preserve IDs, and fail on incompatible dual collections.
- [Risk] Concurrent SQLite writes can produce lock errors. → Keep transactions short, use indexed conflict queries, and return a retryable conflict response for lock failures.
- [Risk] DST transitions can create nonexistent or ambiguous occurrences. → Centralize resolution, document the selected policy, and test both transition types.
- [Risk] PocketBase native route hooks may expose new collections accidentally. → Add a post-auth route guard and integration tests for guest, wrong-realm, and superuser access.
- [Risk] Two browser sessions can drift after expiry or logout. → Keep separate channels and timers, and re-bootstrap on visibility changes for the matching realm.
- [Risk] UI and API disagree on intervals at timezone boundaries. → Persist only UTC instants for concrete records and test round trips through each panel's formatter.

## Migration Plan

1. Add an idempotent schema migration for `teachers` and scheduling collections.
2. Deploy backend auth hooks and custom APIs with the two role-specific session cookies from the first deployment.
3. Deploy the frontend route trees and independent stores.
4. Verify fresh and existing databases, auth isolation, migration preservation, booking conflicts, DST behavior, and critical browser flows.
5. Roll back application binaries if needed. Do not reverse the `users` rename automatically because a destructive reverse can collide with newly created teacher records. Restore a database backup or run an explicit, reviewed reverse migration.
