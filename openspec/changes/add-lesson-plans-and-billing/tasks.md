## 1. Contract Documentation

- [x] 1.1 Update `docs/constitutions/scheduling.md` with plan precedence, fixed duration, learner timing, contract occurrences, token effects, teacher-local business dates, outcomes, payments, and immutable corrections.
- [x] 1.2 Update `docs/constitutions/frontend-query-state.md` with policy, commercial summary, contract series, financial work, and history query keys plus compound mutation invalidation.
- [x] 1.3 Update `docs/api/assignments.md` to remove duration configuration, expose commercial summaries where appropriate, and document obligation-aware deactivation.
- [x] 1.4 Update `docs/api/availability.md` with policy-backed boundaries and the preview/resolved-commit contract for near-term and distant lesson effects.
- [x] 1.5 Update `docs/api/booking.md` with teacher-on-behalf booking, fixed duration, plan selection, minimum notice, start-based horizon, entitlement fields, and atomicity.
- [x] 1.6 Update `docs/api/lifecycle.md` with distinct cancellation and rescheduling, exact cutoff classification, outcomes, plan consequences, and correction commands.
- [x] 1.7 Update `docs/api/calendar.md` with horizon-scoped learner lessons, teacher near/later partitions, commercial summaries, outcomes, and removal of generic cancellation counters.
- [x] 1.8 Add canonical API pages for business policy, packages, regular contracts, payments, business history, and teacher unresolved work.
- [x] 1.9 Update `docs/api/errors.md` with English stable codes for plan precedence, timing, package, contract allowance, unresolved obligation, stale preview, and correction failures.
- [x] 1.10 Update `docs/README.md` links and verify every endpoint has one canonical page without duplicated endpoint definitions.

## 2. Global Policy and Shared Domain Values

- [x] 2.1 Add the `businesspolicy` package with one validated global `Policy`, compact `PolicySnapshot`, stable version, PLN minor-unit prices, and all agreed scheduling and commercial constants.
- [x] 2.2 Add focused policy tests for exact initial values, snapshot stability, and validation of internally inconsistent durations, horizons, limits, dates, and money.
- [x] 2.3 Add shared English domain values for money, actor, teacher-local date, plan type, schedule state, outcome, settlement state, and correction reason without PocketBase dependencies.
- [x] 2.4 Add teacher-timezone calendar helpers for inclusive 60-day validity, seven-day extensions, monthly boundaries, next 30 June, notice end, and 30-day replacements.
- [x] 2.5 Extend scheduling interval rules to accept injected policy grid, duration, buffer, booking minimum, and start horizon while preserving UTC and DST behavior.
- [x] 2.6 Add lowest-layer tests for exact 24-hour and 14-day boundaries, start-versus-end horizon behavior, local date calculations, and DST transitions.
- [x] 2.7 Add the authenticated business-policy read endpoint and role authorization tests with English DTO fields and no landing integration.

## 3. Persistence Schema and Data Migration

- [x] 3.1 Add centralized schema names and fields for packages, tokens, contracts, amendments, contract months, charges, financial entries, and business events.
- [x] 3.2 Add PocketBase migrations for `lesson_packages` and `package_tokens`, including relation rules, token-state validation, package ordinal uniqueness, and one active lesson relation per token.
- [x] 3.3 Add PocketBase migrations for `regular_contracts`, `contract_amendments`, and `contract_months`, including unique contract occurrence and month identities.
- [x] 3.4 Add PocketBase migrations for `charges`, `financial_entries`, and append-only `business_events`, including assignment and source indexes needed by authorized reads.
- [x] 3.5 Extend lessons with plan, entitlement, contract occurrence, policy snapshot, unit price, schedule state, outcome, and omission fields.
- [x] 3.6 Add migration hooks that reject ordinary business-event updates and deletes and validate cross-record assignment and participant ownership.
- [x] 3.7 Implement the migration audit and backfill for existing lessons, assignments, and lesson events, including 45-minute normalization and awaiting-outcome classification.
- [x] 3.8 Add migration tests for fresh databases, populated databases, non-45-minute audit behavior, event preservation, unique indexes, and reversible pre-write rollback.

## 4. Business Event History

- [x] 4.1 Add pure English event types and event constructors for every scheduling, package, contract, availability, outcome, payment, and correction transition.
- [x] 4.2 Add a persistence adapter that writes events in the caller's PocketBase transaction and never exposes an independent mutable event operation.
- [x] 4.3 Add compensating-event resolution that identifies corrected events and derives current effects from uncompensated history.
- [x] 4.4 Add teacher and learner history queries scoped through assignments, with server-side removal of internal teacher notes from learner DTOs.
- [x] 4.5 Add authorization and regression tests that prove unrelated accounts cannot read history and learners cannot receive internal notes.

## 5. Plan Eligibility and Assignment Guardrails

- [x] 5.1 Add the pure commercial eligibility service for `regular_contract`, `package`, then `ad_hoc` precedence.
- [x] 5.2 Add domain tests for contract rejection, forced ad hoc rejection, package token selection, expired packages, teacher–learner scoping, and no-higher-plan ad hoc fallback.
- [x] 5.3 Add overlap validation for contract activation, package purchase, future package lessons, and unresolved ad hoc lessons.
- [x] 5.4 Add atomic conversion decisions for selected future ad hoc lessons during package purchase or contract activation.
- [x] 5.5 Replace assignment duration with fixed policy duration in schema DTOs, seed commands, API validation, and application reads.
- [x] 5.6 Add assignment deactivation guards for contracts, open token balances, package-backed future lessons, and all future scheduled lessons while retaining historical debt visibility.
- [x] 5.7 Add API tests for plan override rejection, atomic conversion failure, cross-assignment entitlement isolation, and obligation-aware deactivation.

## 6. Package Aggregate and Workflows

- [x] 6.1 Add the package aggregate with purchase, validity, available/reserved/used/expired/invalidated token states, and deterministic lowest-ordinal reservation.
- [x] 6.2 Add domain tests for current and backdated purchase, future-date rejection, 60 inclusive local days, final-date lesson starts, no-refund expiry, and DST-spanning validity.
- [x] 6.3 Add package booking decisions that reserve one token atomically and prevent concurrent over-reservation.
- [x] 6.4 Add package lifecycle decisions for completion, no-show, timely cancellation, late cancellation, repeated reschedule, and expiration after token return.
- [x] 6.5 Add teacher cancellation decisions that return a token and extend validity by seven local days per cancellation without extending on reschedule.
- [x] 6.6 Add package renewal validation that permits a new purchase only when no prior token is available while retaining old future lesson links.
- [x] 6.7 Add early package closure with required reason, future-lesson guard, token invalidation, optional refund information, and compensating corrections.
- [x] 6.8 Add teacher package purchase, close, correction, and assignment summary endpoints with strict request decoding and authorization tests.

## 7. Regular Contract Aggregate and Series

- [x] 7.1 Add the regular contract aggregate with explicit teacher-local start and end dates, weekly weekday and time, fixed price snapshot, status, notice, and allowance queries.
- [x] 7.2 Add activation validation for assigned teacher authority, effective availability, grid, fixed duration, participant conflicts, next-30-June end, and prohibited commercial overlaps.
- [x] 7.3 Add controlled backdating with required reason and explicit resolution for every past generated occurrence.
- [x] 7.4 Add DST-safe full-series materialization with stable original local dates, UTC intervals, unique occurrence identities, and contract links.
- [x] 7.5 Add teacher full-series and learner horizon-scoped queries that preserve distant conflict protection and classify near-term versus later starts.
- [x] 7.6 Add event-derived learner allowance checks for one reschedule per original month, one learner reschedule per occurrence, and two free cancellations per contract.
- [x] 7.7 Add domain tests for cross-month replacements, non-accumulating monthly allowances, repeat reschedule rejection, exhausted monthly limits, timely free cancellation, late billable cancellation, exhausted free cancellation, and no-show.
- [x] 7.8 Add 30-local-day replacement validation, including post-30-June replacements and teacher placement beyond the learner horizon.
- [x] 7.9 Add teacher occurrence reschedule and cancellation decisions that bypass learner cutoffs and allowances while preserving the proper financial effects.
- [x] 7.10 Add permanent schedule-change reconciliation that updates only unmodified future occurrences and preserves individually rescheduled lessons.
- [x] 7.11 Add learner and teacher ordinary notice with computed next-month end, plus teacher-recorded earlier mutual end with a required reason.
- [x] 7.12 Add explicit renewal that proposes the prior time, revalidates availability, snapshots current policy, and creates fresh allowance history.
- [x] 7.13 Add prospective price amendments effective only on the first day of a future month without changing earlier charges.
- [x] 7.14 Add contract activation, series, schedule, amendment, notice, renewal, early-end, and correction endpoints with participant authorization tests.

## 8. Plan-Aware Booking and Lesson Lifecycle

- [x] 8.1 Refactor learner booking to call the shared eligibility and scheduling domain services and atomically create plan source, snapshot, entitlement, obligation, and events.
- [x] 8.2 Add teacher-on-behalf flexible booking with the same 14-day upper horizon and explicit short-notice confirmation below the learner 24-hour minimum.
- [x] 8.3 Remove learner and teacher duration override payloads and return fixed 45-minute lesson DTOs with plan and entitlement references.
- [x] 8.4 Refactor rescheduling into one domain command that preserves lesson identity and atomically applies replacement, token, allowance, billing, and history effects.
- [x] 8.5 Refactor cancellation into a distinct command with exact timely or late classification, reject learner rescheduling inside the cutoff, and apply package, contract, ad hoc, or teacher consequences.
- [x] 8.6 Add ended-lesson outcome commands for `completed` and `learner_no_show`, with derived `awaiting_outcome`, no automatic attendance assertion, and unchanged scheduled state for operational lateness.
- [x] 8.7 Apply the agreed no-penalty rule to late ad hoc cancellation and ad hoc no-show while retaining immutable events.
- [x] 8.8 Add teacher correction commands that require reasons and restore token or contract allowance effects through compensating events.
- [x] 8.9 Replace generic cancellation counter reads with plan-specific balances and history projections.
- [x] 8.10 Extend HTTP tests for minimum and horizon boundaries, teacher short notice, plan-specific reschedule and cancellation, outcomes, concurrent slot and token races, and atomic rollback.

## 9. Availability Preview and Reconciliation

- [x] 9.1 Add a pure availability impact planner that compares proposed changes with near-term lessons and distant contract occurrences.
- [x] 9.2 Add preview DTOs that separate required near-term resolutions from automatic distant omissions and restorations and include an opaque state version.
- [x] 9.3 Add preview endpoints for recurring-rule and dated-exception create, update, enable, disable, and delete operations.
- [x] 9.4 Add resolved commit services that re-evaluate state inside one transaction and reject stale, incomplete, or invalid resolution sets.
- [x] 9.5 Apply teacher cancel or reschedule decisions to every near-term conflict and planned omission or restoration decisions to distant contract occurrences.
- [x] 9.6 Ensure planned distant omissions remove billing without package extension or learner allowance effects and that distant restoration reinstates eligible billing.
- [x] 9.7 Preserve the rule that restored near-term availability never silently recreates a cancelled lesson.
- [x] 9.8 Add domain and HTTP regression tests for partial resolution rejection, stale preview, all-or-nothing persistence, distant omission and restoration, and no inferred holidays.
- [x] 9.9 Block ordinary teacher timezone changes while active contracts or future lessons exist and add an authorization and integrity regression test.

## 10. Ledger and Monthly Reconciliation

- [x] 10.1 Add the ledger domain model for integer minor-unit charges, financial entries, payment states, derived overdue state, credits, refunds, and corrections.
- [x] 10.2 Add ad hoc settlement decisions for pending settlement, explicit paid default submission, intentional nonpayment, later payment, and immutable transition history.
- [x] 10.3 Ensure ad hoc cancellation and no-show resolve settlement as `not_applicable` without creating debt and that unpaid ad hoc never blocks booking.
- [x] 10.4 Add idempotent contract month forecasts and first-day charge reconciliation, including immediate current-month creation for mid-month activation.
- [x] 10.5 Compute monthly amount from billable occurrence snapshots and distinguish planned omissions, free cancellations, teacher cancellations, late cancellations, exhausted allowances, and no-shows.
- [x] 10.6 Add unpaid-charge reductions, paid-charge credits, oldest-first next-charge credit application, and teacher-recorded refund alternatives.
- [x] 10.7 Add package purchase payment entries in the same transaction as package and token creation.
- [x] 10.8 Add payment, refund, correction, forecast, monthly charge, and unresolved-work endpoints with teacher and learner role-safe DTOs.
- [x] 10.9 Add core tests for five-lesson and four-lesson months, mid-month starts, day-five overdue derivation, post-payment cancellation credits, refunds, later ad hoc payment, and non-enforcement.

## 11. Backend Read Models and API Integration

- [x] 11.1 Define concise assignment commercial summary DTOs with active plan, token balance, contract allowances, package expiry, and payment summary.
- [x] 11.2 Update teacher calendar reads to return near-term lessons separately from later contract reservations and exclude unbounded event and ledger detail.
- [x] 11.3 Update learner calendar reads to return only lesson starts inside the current horizon while retaining owned commercial summaries and payment information.
- [x] 11.4 Add focused paginated reads for full teacher contract series, financial history, business history, and unresolved teacher work.
- [x] 11.5 Ensure every DTO builder strips raw PocketBase fields, internal notes, unrelated identities, and unauthorized financial data.
- [x] 11.6 Register all new routes under the correct persona realm and preserve HttpOnly session identity, intent headers, strict JSON decoding, and stable English errors.
- [x] 11.7 Add end-to-end backend authorization tests across policy, packages, contracts, payments, history, teacher-on-behalf actions, and learner-owned reads.

## 12. React Data Contracts and Cache Ownership

- [x] 12.1 Replace legacy scheduling DTOs with English typed policy, plan, token, contract, lesson outcome, payment, preview, and event contracts.
- [x] 12.2 Add endpoint functions through the existing sole Fetch transport for all new reads and mutations.
- [x] 12.3 Add centralized query keys for policy, commercial summaries, full contract series, financial work, and role-scoped history.
- [x] 12.4 Add centralized mutation effects for booking, lifecycle, plan, availability, outcome, settlement, and correction changes without optimistic compound writes.
- [x] 12.5 Add Polish copy maps for every English backend plan, state, event, and stable error code, including a generic unknown-code fallback.
- [x] 12.6 Add frontend contract and query tests for policy authority, persona-scoped cache isolation, compound invalidation, and protected-field absence.

## 13. Teacher Application Workflows

- [x] 13.1 Replace assignment duration controls with an assignment commercial workspace and valid plan transition actions.
- [x] 13.2 Add package purchase, existing ad hoc conversion, token balance, renewal, closure, refund note, and correction interfaces.
- [x] 13.3 Add contract activation with weekly term selection, series preview, conflict feedback, controlled backdating, and existing ad hoc resolution.
- [x] 13.4 Add contract full-series view with clear near-term and later sections, permanent schedule change, notice, early end, amendment, and renewal controls.
- [x] 13.5 Add teacher-on-behalf ad hoc and package booking with explicit short-notice warning confirmation.
- [x] 13.6 Add lesson cancellation and atomic reschedule controls that display plan-specific effects.
- [x] 13.7 Add ended-lesson close forms that default visually to completed but persist nothing until submission.
- [x] 13.8 Add ad hoc settlement forms that default visually to paid, distinguish pending from intentional nonpayment, and support later payment.
- [x] 13.9 Add unresolved outcomes, unpaid and overdue charges, credits, refunds, and correction work queues.
- [x] 13.10 Replace availability direct writes with conflict preview, complete per-lesson resolution, distant-effect review, and atomic commit.
- [x] 13.11 Add full teacher history display with correction relationships and internal notes.

## 14. Learner Application Workflows

- [x] 14.1 Show the learner's active plan, package expiry and token states, contract allowances, payment summary, and owned history.
- [x] 14.2 Limit detailed learner lessons and controls to starts inside the policy horizon while preserving teacher-local business date labels.
- [x] 14.3 Filter flexible slots to the policy minimum and horizon and remove booking controls during active contracts.
- [x] 14.4 Let eligible package bookings consume the authoritative token automatically and use ad hoc only when no higher plan applies.
- [x] 14.5 Add consequence previews and confirmation for timely and late cancellation and rescheduling across all three plans.
- [x] 14.6 Add contract notice submission and show the computed notice period and effective end.
- [x] 14.7 Display original and correction events without teacher internal notes and translate all English machine values into Polish.

## 15. Critical Flow and Quality Verification

- [x] 15.1 Extend existing Playwright scheduling flows for ad hoc booking and settlement, package purchase through four tokens, and strict plan precedence.
- [x] 15.2 Add a critical regular-contract browser flow covering activation, full teacher series, limited learner view, monthly forecast, timely reschedule, free cancellation, and payment.
- [x] 15.3 Add a critical availability browser flow covering preview, required conflict resolution, atomic commit, distant omission, and teacher near/later visual separation.
- [x] 15.4 Add a correction and authorization browser or HTTP flow proving history retention and learner note redaction at the boundary that matters.
- [x] 15.5 Run `go test ./...` in `apps/backend` and resolve every failure.
- [x] 15.6 Run the React type check and relevant unit tests and resolve every failure.
- [x] 15.7 Run the most relevant Playwright tests and resolve every failure.
- [x] 15.8 Run `go build ./...` in `apps/backend` and `npm run build` at the repository root and resolve every build failure.
- [x] 15.9 Run `./tools/code_quality/check.py check` and keep every modified hand-written Go and TypeScript file within existing NLOC and CCN baselines.
- [x] 15.10 Run `npm audit --audit-level=high --omit=dev` and resolve every high or critical production advisory.
- [x] 15.11 Run strict OpenSpec validation, documentation link checks, and a release comparison of independent landing prices against backend policy.
- [x] 15.12 Add a forward repair migration and calendar regression coverage for databases that applied an earlier commercial schema revision.
- [x] 15.13 Preserve empty availability arrays and migrate legacy amount fields that reject valid zero-value settlement outcomes.
