## Why

Nutka has learner authentication but no shared scheduling model. Teachers cannot publish recurring availability, learners cannot book assigned teachers, and neither persona has a calendar workflow or durable lesson history. This change establishes the scheduling foundation now so later cancellation policy can build on retained events and initiator attribution.

## What Changes

- Add a teacher auth realm while preserving learner auth as a separate realm.
- Rename the existing custom `users` auth collection to `teachers` when it exists, and create `teachers` on a fresh database.
- Add independent teacher and learner route spaces under `/teachers/*` and `/learners/*`.
- Issue independent HttpOnly JWT session cookies named `__Host-nutka_teacher_session` and `__Host-nutka_learner_session` from the first deployment.
- Add many-to-many teacher–learner assignments.
- Add indefinitely recurring weekly teacher availability in a validated teacher IANA timezone, defaulting migrated and new teachers to `Europe/Warsaw`.
- Add dated available and unavailable exceptions, with unavailable exceptions taking precedence.
- Generate 15-minute booking slots within a rolling 14-day horizon.
- Support a teacher-configured 45-minute default per assignment and teacher-only per-lesson duration changes.
- Make learner booking always use the assignment default without a learner duration override.
- Require every lesson duration to be a positive multiple of 15 minutes.
- Apply five-minute protected buffers before and after every lesson for both participants.
- Permit learners to book only assigned teachers through authorized APIs.
- Make booking, rescheduling, and conflict checks atomic for teacher and learner calendars.
- Permit either persona to reschedule or cancel future lessons.
- Make started and past lessons immutable.
- Retain cancelled lessons and record the initiating persona and event history.
- Attribute cancellation counters to the persona that initiated each cancellation.
- Do not increment cancellation counters for reschedules.
- Reject availability blocks that conflict with scheduled lessons.
- Persist concrete datetimes as UTC instants and localize them at UI boundaries.
- Add teacher and learner calendar panels with availability, lesson, reschedule, cancellation, and counter views.
- Keep public registration and invitation workflows out of scope.

## Capabilities

### New Capabilities

- `dual-persona-auth`: Independent teacher and learner authentication realms, cookies, route guards, and session APIs.
- `teacher-availability`: Recurring weekly availability, IANA timezone handling, dated exceptions, slot generation, buffers, and blocking rules.
- `teacher-learner-assignments`: Many-to-many assignments and teacher-specific default lesson durations.
- `lesson-booking`: Authorized learner booking within the rolling horizon with duration validation and atomic conflict protection.
- `lesson-lifecycle`: Future lesson rescheduling and cancellation, immutable started or past lessons, retained history, and initiator counters.
- `calendar-panels`: Teacher and learner calendar panels that consume the authorized scheduling APIs and localize UTC datetimes.

### Modified Capabilities

<!-- No existing OpenSpec capabilities exist in this repository. -->

## Impact

- `apps/backend`: PocketBase migrations, auth middleware, session cookies, scheduling domain services, authorized HTTP APIs, and business-rule tests.
- `apps/app`: Separate teacher and learner route trees, auth stores, calendar panels, API clients, and browser-flow tests.
- `docs`: Authentication contract and scheduling constitution/API references.
- PocketBase schema: `teachers`, `learners`, `teacher_learners`, `availability_rules`, `availability_exceptions`, `lessons`, and `lesson_events`.
- Existing learner auth behavior remains authoritative. The implementation must reuse it rather than copy the Sailormoon mechanism.
