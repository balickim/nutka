## Why

Nutka schedules generic lessons but does not model the commercial rules promised by the pricing page and lesson regulations. The backend must become the single authority for ad hoc lessons, four-lesson packages, regular-plan contracts, payments, entitlements, and their auditable lifecycle.

## What Changes

- Add three mutually prioritized commercial lesson plans: regular-plan contract, active package, and ad hoc.
- Keep the free trial outside the invite-only application and its learner onboarding flow.
- Fix every commercial lesson at 45 minutes and preserve the existing 15-minute slot grid and five-minute participant buffers.
- Add a global backend business-policy configuration and expose its current values to the authenticated React application.
- Add teacher-recorded purchases for prepaid four-token packages that remain valid for 60 teacher-local calendar days.
- Reserve package tokens during booking and settle, return, expire, extend, or correct them through explicit domain transitions.
- Add fixed weekly contracts selected by the teacher, ending on 30 June unless terminated earlier.
- Materialize contract lessons, reserve their slots beyond the learner booking horizon, and derive monthly charges from billable occurrences.
- Add learner self-service and teacher-on-behalf booking, rescheduling, cancellation, contract notice, and plan-specific eligibility checks.
- Add teacher workflows that resolve every near-term lesson conflict before an availability mutation commits.
- Add lesson completion and learner no-show outcomes.
- Add payment tracking for ad hoc lessons and monthly contract charges without payment processing or automated enforcement.
- Add immutable, authorized business history for scheduling, plans, tokens, charges, payments, and administrative corrections.
- Keep external notifications, public registration, guardian accounts, instrument tracking, statutory-holiday calendars, and online payments outside this change.
- **BREAKING** Remove configurable assignment and lesson durations. All supported lessons use the backend-configured 45-minute duration.
- **BREAKING** Replace generic cancellation counters and unrestricted future lifecycle actions with plan-aware entitlements, a 24-hour learner change cutoff, and the configurable booking horizon.

## Capabilities

### New Capabilities

- `business-policy-config`: Defines the global backend authority and authenticated application contract for current prices, currency, durations, buffers, horizons, cutoffs, package terms, and contract limits.
- `commercial-lesson-plans`: Defines the three lesson plans, their strict precedence, lesson classification, purchase and activation constraints, and exclusion of trials.
- `lesson-packages`: Defines package purchase, token reservation and settlement, validity, renewal, extension, closure, and corrections.
- `regular-plan-contracts`: Defines fixed weekly contracts, generated occurrences, availability effects, amendments, notice, renewal, rescheduling allowances, cancellation allowances, and monthly forecasts.
- `payment-ledger`: Defines ad hoc settlement, contract monthly charges, overdue state, credits, refunds, corrections, and non-enforcement.
- `business-event-history`: Defines immutable audit events, correction events, actor attribution, reasons, and role-scoped visibility.

### Modified Capabilities

- `lesson-booking`: Makes booking plan-aware, fixes duration, adds teacher-on-behalf booking, applies strict plan precedence, and changes the horizon to constrain lesson starts with a 24-hour learner minimum.
- `lesson-lifecycle`: Adds plan-aware rescheduling and cancellation, lesson outcomes, cutoff consequences, replacement links, entitlement effects, and administrative corrections.
- `teacher-availability`: Sources grid, buffer, and horizon from backend policy and resolves availability changes differently inside and outside the booking horizon.
- `teacher-learner-assignments`: Removes duration configuration and blocks deactivation while active commercial obligations or future lessons remain.
- `calendar-panels`: Adds plan, entitlement, payment, history, outcome, conflict-resolution, and clearly separated near-term and later-contract views.

## Impact

- Adds pure Go domain models and services under `apps/backend/internal` as the sole implementation of commercial and scheduling policy.
- Adds PocketBase collections and migrations for plans, packages, contracts, occurrences, charges, payments, credits, and immutable business events.
- Changes booking, lifecycle, availability, assignment, calendar, and error API contracts and adds authenticated policy, plan, payment, outcome, and history endpoints.
- Changes React DTOs, query keys, mutation effects, teacher workflows, learner booking, calendars, settlement views, and Polish presentation copy.
- Updates scheduling, frontend query-state, and API constitutions and canonical API reference pages.
- Removes assignment duration controls and teacher duration overrides from the backend and React application.
- Leaves `apps/landing`, its pricing configuration, and its static deployment unchanged.
- Adds no payment gateway, notification provider, holiday dependency, guardian model, or instrument model.
