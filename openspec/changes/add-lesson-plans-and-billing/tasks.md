## 1. Contract Documentation

- [ ] 1.1 Update `docs/constitutions/scheduling.md` with plan precedence, fixed duration, learner timing, contract occurrences, token effects, teacher-local business dates, outcomes, payments, and immutable corrections.
- [ ] 1.2 Update `docs/constitutions/frontend-query-state.md` with policy, commercial summary, contract series, financial work, and history query keys plus compound mutation invalidation.
- [ ] 1.3 Update `docs/api/assignments.md` to remove duration configuration, expose commercial summaries where appropriate, and document obligation-aware deactivation.
- [ ] 1.4 Update `docs/api/availability.md` with policy-backed boundaries and the preview/resolved-commit contract for near-term and distant lesson effects.
- [ ] 1.5 Update `docs/api/booking.md` with teacher-on-behalf booking, fixed duration, plan selection, minimum notice, start-based horizon, entitlement fields, and atomicity.
- [ ] 1.6 Update `docs/api/lifecycle.md` with distinct cancellation and rescheduling, exact cutoff classification, outcomes, plan consequences, and correction commands.
- [ ] 1.7 Update `docs/api/calendar.md` with horizon-scoped learner lessons, teacher near/later partitions, commercial summaries, outcomes, and removal of generic cancellation counters.
- [ ] 1.8 Add canonical API pages for business policy, packages, regular contracts, payments, business history, and teacher unresolved work.
- [ ] 1.9 Update `docs/api/errors.md` with English stable codes for plan precedence, timing, package, contract allowance, unresolved obligation, stale preview, and correction failures.
- [ ] 1.10 Update `docs/README.md` links and verify every endpoint has one canonical page without duplicated endpoint definitions.

## 2. Global Policy and Shared Domain Values

- [ ] 2.1 Add the `businesspolicy` package with one validated global `Policy`, compact `PolicySnapshot`, stable version, PLN minor-unit prices, and all agreed scheduling and commercial constants.
- [ ] 2.2 Add focused policy tests for exact initial values, snapshot stability, and validation of internally inconsistent durations, horizons, limits, dates, and money.
- [ ] 2.3 Add shared English domain values for money, actor, teacher-local date, plan type, schedule state, outcome, settlement state, and correction reason without PocketBase dependencies.
- [ ] 2.4 Add teacher-timezone calendar helpers for inclusive 60-day validity, seven-day extensions, monthly boundaries, next 30 June, notice end, and 30-day replacements.
- [ ] 2.5 Extend scheduling interval rules to accept injected policy grid, duration, buffer, booking minimum, and start horizon while preserving UTC and DST behavior.
- [ ] 2.6 Add lowest-layer tests for exact 24-hour and 14-day boundaries, start-versus-end horizon behavior, local date calculations, and DST transitions.
- [ ] 2.7 Add the authenticated business-policy read endpoint and role authorization tests with English DTO fields and no landing integration.

## 3. Persistence Schema and Data Migration

- [ ] 3.1 Add centralized schema names and fields for packages, tokens, contracts, amendments, contract months, charges, financial entries, and business events.
- [ ] 3.2 Add PocketBase migrations for `lesson_packages` and `package_tokens`, including relation rules, token-state validation, package ordinal uniqueness, and one active lesson relation per token.
- [ ] 3.3 Add PocketBase migrations for `regular_contracts`, `contract_amendments`, and `contract_months`, including unique contract occurrence and month identities.
- [ ] 3.4 Add PocketBase migrations for `charges`, `financial_entries`, and append-only `business_events`, including assignment and source indexes needed by authorized reads.
- [ ] 3.5 Extend lessons with plan, entitlement, contract occurrence, policy snapshot, unit price, schedule state, outcome, and omission fields.
- [ ] 3.6 Add migration hooks that reject ordinary business-event updates and deletes and validate cross-record assignment and participant ownership.
- [ ] 3.7 Implement the migration audit and backfill for existing lessons, assignments, and lesson events, including 45-minute normalization and awaiting-outcome classification.
- [ ] 3.8 Add migration tests for fresh databases, populated databases, non-45-minute audit behavior, event preservation, unique indexes, and reversible pre-write rollback.

## 4. Business Event History

- [ ] 4.1 Add pure English event types and event constructors for every scheduling, package, contract, availability, outcome, payment, and correction transition.
- [ ] 4.2 Add a persistence adapter that writes events in the caller's PocketBase transaction and never exposes an independent mutable event operation.
- [ ] 4.3 Add compensating-event resolution that identifies corrected events and derives current effects from uncompensated history.
- [ ] 4.4 Add teacher and learner history queries scoped through assignments, with server-side removal of internal teacher notes from learner DTOs.
- [ ] 4.5 Add authorization and regression tests that prove unrelated accounts cannot read history and learners cannot receive internal notes.

## 5. Plan Eligibility and Assignment Guardrails

- [ ] 5.1 Add the pure commercial eligibility service for `regular_contract`, `package`, then `ad_hoc` precedence.
- [ ] 5.2 Add domain tests for contract rejection, forced ad hoc rejection, package token selection, expired packages, teacher–learner scoping, and no-higher-plan ad hoc fallback.
- [ ] 5.3 Add overlap validation for contract activation, package purchase, future package lessons, and unresolved ad hoc lessons.
- [ ] 5.4 Add atomic conversion decisions for selected future ad hoc lessons during package purchase or contract activation.
- [ ] 5.5 Replace assignment duration with fixed policy duration in schema DTOs, seed commands, API validation, and application reads.
- [ ] 5.6 Add assignment deactivation guards for contracts, open token balances, package-backed future lessons, and all future scheduled lessons while retaining historical debt visibility.
- [ ] 5.7 Add API tests for plan override rejection, atomic conversion failure, cross-assignment entitlement isolation, and obligation-aware deactivation.

## 6. Package Aggregate and Workflows

- [ ] 6.1 Add the package aggregate with purchase, validity, available/reserved/used/expired/invalidated token states, and deterministic lowest-ordinal reservation.
- [ ] 6.2 Add domain tests for current and backdated purchase, future-date rejection, 60 inclusive local days, final-date lesson starts, no-refund expiry, and DST-spanning validity.
- [ ] 6.3 Add package booking decisions that reserve one token atomically and prevent concurrent over-reservation.
- [ ] 6.4 Add package lifecycle decisions for completion, no-show, timely cancellation, late cancellation, repeated reschedule, and expiration after token return.
- [ ] 6.5 Add teacher cancellation decisions that return a token and extend validity by seven local days per cancellation without extending on reschedule.
- [ ] 6.6 Add package renewal validation that permits a new purchase only when no prior token is available while retaining old future lesson links.
- [ ] 6.7 Add early package closure with required reason, future-lesson guard, token invalidation, optional refund information, and compensating corrections.
- [ ] 6.8 Add teacher package purchase, close, correction, and assignment summary endpoints with strict request decoding and authorization tests.

## 7. Regular Contract Aggregate and Series

- [ ] 7.1 Add the regular contract aggregate with explicit teacher-local start and end dates, weekly weekday and time, fixed price snapshot, status, notice, and allowance queries.
- [ ] 7.2 Add activation validation for assigned teacher authority, effective availability, grid, fixed duration, participant conflicts, next-30-June end, and prohibited commercial overlaps.
- [ ] 7.3 Add controlled backdating with required reason and explicit resolution for every past generated occurrence.
- [ ] 7.4 Add DST-safe full-series materialization with stable original local dates, UTC intervals, unique occurrence identities, and contract links.
- [ ] 7.5 Add teacher full-series and learner horizon-scoped queries that preserve distant conflict protection and classify near-term versus later starts.
- [ ] 7.6 Add event-derived learner allowance checks for one reschedule per original month, one learner reschedule per occurrence, and two free cancellations per contract.
- [ ] 7.7 Add domain tests for cross-month replacements, non-accumulating monthly allowances, repeat reschedule rejection, exhausted monthly limits, timely free cancellation, late billable cancellation, exhausted free cancellation, and no-show.
- [ ] 7.8 Add 30-local-day replacement validation, including post-30-June replacements and teacher placement beyond the learner horizon.
- [ ] 7.9 Add teacher occurrence reschedule and cancellation decisions that bypass learner cutoffs and allowances while preserving the proper financial effects.
- [ ] 7.10 Add permanent schedule-change reconciliation that updates only unmodified future occurrences and preserves individually rescheduled lessons.
- [ ] 7.11 Add learner and teacher ordinary notice with computed next-month end, plus teacher-recorded earlier mutual end with a required reason.
- [ ] 7.12 Add explicit renewal that proposes the prior time, revalidates availability, snapshots current policy, and creates fresh allowance history.
- [ ] 7.13 Add prospective price amendments effective only on the first day of a future month without changing earlier charges.
- [ ] 7.14 Add contract activation, series, schedule, amendment, notice, renewal, early-end, and correction endpoints with participant authorization tests.

## 8. Plan-Aware Booking and Lesson Lifecycle

- [ ] 8.1 Refactor learner booking to call the shared eligibility and scheduling domain services and atomically create plan source, snapshot, entitlement, obligation, and events.
- [ ] 8.2 Add teacher-on-behalf flexible booking with the same 14-day upper horizon and explicit short-notice confirmation below the learner 24-hour minimum.
- [ ] 8.3 Remove learner and teacher duration override payloads and return fixed 45-minute lesson DTOs with plan and entitlement references.
- [ ] 8.4 Refactor rescheduling into one domain command that preserves lesson identity and atomically applies replacement, token, allowance, billing, and history effects.
- [ ] 8.5 Refactor cancellation into a distinct command with exact timely or late classification, reject learner rescheduling inside the cutoff, and apply package, contract, ad hoc, or teacher consequences.
- [ ] 8.6 Add ended-lesson outcome commands for `completed` and `learner_no_show`, with derived `awaiting_outcome`, no automatic attendance assertion, and unchanged scheduled state for operational lateness.
- [ ] 8.7 Apply the agreed no-penalty rule to late ad hoc cancellation and ad hoc no-show while retaining immutable events.
- [ ] 8.8 Add teacher correction commands that require reasons and restore token or contract allowance effects through compensating events.
- [ ] 8.9 Replace generic cancellation counter reads with plan-specific balances and history projections.
- [ ] 8.10 Extend HTTP tests for minimum and horizon boundaries, teacher short notice, plan-specific reschedule and cancellation, outcomes, concurrent slot and token races, and atomic rollback.

## 9. Availability Preview and Reconciliation

- [ ] 9.1 Add a pure availability impact planner that compares proposed changes with near-term lessons and distant contract occurrences.
- [ ] 9.2 Add preview DTOs that separate required near-term resolutions from automatic distant omissions and restorations and include an opaque state version.
- [ ] 9.3 Add preview endpoints for recurring-rule and dated-exception create, update, enable, disable, and delete operations.
- [ ] 9.4 Add resolved commit services that re-evaluate state inside one transaction and reject stale, incomplete, or invalid resolution sets.
- [ ] 9.5 Apply teacher cancel or reschedule decisions to every near-term conflict and planned omission or restoration decisions to distant contract occurrences.
- [ ] 9.6 Ensure planned distant omissions remove billing without package extension or learner allowance effects and that distant restoration reinstates eligible billing.
- [ ] 9.7 Preserve the rule that restored near-term availability never silently recreates a cancelled lesson.
- [ ] 9.8 Add domain and HTTP regression tests for partial resolution rejection, stale preview, all-or-nothing persistence, distant omission and restoration, and no inferred holidays.
- [ ] 9.9 Block ordinary teacher timezone changes while active contracts or future lessons exist and add an authorization and integrity regression test.

## 10. Ledger and Monthly Reconciliation

- [ ] 10.1 Add the ledger domain model for integer minor-unit charges, financial entries, payment states, derived overdue state, credits, refunds, and corrections.
- [ ] 10.2 Add ad hoc settlement decisions for pending settlement, explicit paid default submission, intentional nonpayment, later payment, and immutable transition history.
- [ ] 10.3 Ensure ad hoc cancellation and no-show resolve settlement as `not_applicable` without creating debt and that unpaid ad hoc never blocks booking.
- [ ] 10.4 Add idempotent contract month forecasts and first-day charge reconciliation, including immediate current-month creation for mid-month activation.
- [ ] 10.5 Compute monthly amount from billable occurrence snapshots and distinguish planned omissions, free cancellations, teacher cancellations, late cancellations, exhausted allowances, and no-shows.
- [ ] 10.6 Add unpaid-charge reductions, paid-charge credits, oldest-first next-charge credit application, and teacher-recorded refund alternatives.
- [ ] 10.7 Add package purchase payment entries in the same transaction as package and token creation.
- [ ] 10.8 Add payment, refund, correction, forecast, monthly charge, and unresolved-work endpoints with teacher and learner role-safe DTOs.
- [ ] 10.9 Add core tests for five-lesson and four-lesson months, mid-month starts, day-five overdue derivation, post-payment cancellation credits, refunds, later ad hoc payment, and non-enforcement.

## 11. Backend Read Models and API Integration

- [ ] 11.1 Define concise assignment commercial summary DTOs with active plan, token balance, contract allowances, package expiry, and payment summary.
- [ ] 11.2 Update teacher calendar reads to return near-term lessons separately from later contract reservations and exclude unbounded event and ledger detail.
- [ ] 11.3 Update learner calendar reads to return only lesson starts inside the current horizon while retaining owned commercial summaries and payment information.
- [ ] 11.4 Add focused paginated reads for full teacher contract series, financial history, business history, and unresolved teacher work.
- [ ] 11.5 Ensure every DTO builder strips raw PocketBase fields, internal notes, unrelated identities, and unauthorized financial data.
- [ ] 11.6 Register all new routes under the correct persona realm and preserve HttpOnly session identity, intent headers, strict JSON decoding, and stable English errors.
- [ ] 11.7 Add end-to-end backend authorization tests across policy, packages, contracts, payments, history, teacher-on-behalf actions, and learner-owned reads.

## 12. React Data Contracts and Cache Ownership

- [ ] 12.1 Replace legacy scheduling DTOs with English typed policy, plan, token, contract, lesson outcome, payment, preview, and event contracts.
- [ ] 12.2 Add endpoint functions through the existing sole Fetch transport for all new reads and mutations.
- [ ] 12.3 Add centralized query keys for policy, commercial summaries, full contract series, financial work, and role-scoped history.
- [ ] 12.4 Add centralized mutation effects for booking, lifecycle, plan, availability, outcome, settlement, and correction changes without optimistic compound writes.
- [ ] 12.5 Add Polish copy maps for every English backend plan, state, event, and stable error code, including a generic unknown-code fallback.
- [ ] 12.6 Add frontend contract and query tests for policy authority, persona-scoped cache isolation, compound invalidation, and protected-field absence.

## 13. Teacher Application Workflows

- [ ] 13.1 Replace assignment duration controls with an assignment commercial workspace and valid plan transition actions.
- [ ] 13.2 Add package purchase, existing ad hoc conversion, token balance, renewal, closure, refund note, and correction interfaces.
- [ ] 13.3 Add contract activation with weekly term selection, series preview, conflict feedback, controlled backdating, and existing ad hoc resolution.
- [ ] 13.4 Add contract full-series view with clear near-term and later sections, permanent schedule change, notice, early end, amendment, and renewal controls.
- [ ] 13.5 Add teacher-on-behalf ad hoc and package booking with explicit short-notice warning confirmation.
- [ ] 13.6 Add lesson cancellation and atomic reschedule controls that display plan-specific effects.
- [ ] 13.7 Add ended-lesson close forms that default visually to completed but persist nothing until submission.
- [ ] 13.8 Add ad hoc settlement forms that default visually to paid, distinguish pending from intentional nonpayment, and support later payment.
- [ ] 13.9 Add unresolved outcomes, unpaid and overdue charges, credits, refunds, and correction work queues.
- [ ] 13.10 Replace availability direct writes with conflict preview, complete per-lesson resolution, distant-effect review, and atomic commit.
- [ ] 13.11 Add full teacher history display with correction relationships and internal notes.

## 14. Learner Application Workflows

- [ ] 14.1 Show the learner's active plan, package expiry and token states, contract allowances, payment summary, and owned history.
- [ ] 14.2 Limit detailed learner lessons and controls to starts inside the policy horizon while preserving teacher-local business date labels.
- [ ] 14.3 Filter flexible slots to the policy minimum and horizon and remove booking controls during active contracts.
- [ ] 14.4 Let eligible package bookings consume the authoritative token automatically and use ad hoc only when no higher plan applies.
- [ ] 14.5 Add consequence previews and confirmation for timely and late cancellation and rescheduling across all three plans.
- [ ] 14.6 Add contract notice submission and show the computed notice period and effective end.
- [ ] 14.7 Display original and correction events without teacher internal notes and translate all English machine values into Polish.

## 15. Critical Flow and Quality Verification

- [ ] 15.1 Extend existing Playwright scheduling flows for ad hoc booking and settlement, package purchase through four tokens, and strict plan precedence.
- [ ] 15.2 Add a critical regular-contract browser flow covering activation, full teacher series, limited learner view, monthly forecast, timely reschedule, free cancellation, and payment.
- [ ] 15.3 Add a critical availability browser flow covering preview, required conflict resolution, atomic commit, distant omission, and teacher near/later visual separation.
- [ ] 15.4 Add a correction and authorization browser or HTTP flow proving history retention and learner note redaction at the boundary that matters.
- [ ] 15.5 Run `go test ./...` in `apps/backend` and resolve every failure.
- [ ] 15.6 Run the React type check and relevant unit tests and resolve every failure.
- [ ] 15.7 Run the most relevant Playwright tests and resolve every failure.
- [ ] 15.8 Run `go build ./...` in `apps/backend` and `npm run build` at the repository root and resolve every build failure.
- [ ] 15.9 Run `./tools/code_quality/check.py check` and keep every modified hand-written Go and TypeScript file within existing NLOC and CCN baselines.
- [ ] 15.10 Run `pnpm audit --audit-level=high --prod` and resolve every high or critical production advisory.
- [ ] 15.11 Run strict OpenSpec validation, documentation link checks, and a release comparison of independent landing prices against backend policy.
