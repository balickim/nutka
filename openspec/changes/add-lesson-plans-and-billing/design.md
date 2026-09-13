## Context

See `proposal.md` for motivation and `specs/` for observable behavior.

The backend currently stores teacher–learner assignments, availability rules, dated exceptions, lessons, and lesson events in PocketBase. Pure interval and timezone rules live in `internal/scheduling`, while HTTP handlers still orchestrate most lifecycle decisions directly against records. Lessons support configurable assignment durations, two schedule states, unrestricted future learner changes, and generic cancellation counters.

The React application reads aggregate calendar responses through one Fetch boundary and TanStack Query cache. It has teacher availability and assignment controls plus learner slot booking. The public Astro landing owns separate Polish pricing copy and is intentionally outside this change.

This change introduces cross-cutting commercial invariants, money and entitlement effects, recurring contract materialization, immutable corrections, and multi-record atomic mutations. The repository limits hand-written Go and TypeScript files to 250 NLOC and functions to CCN 10. Every modified hand-written non-test source file requires a short purpose-and-boundary overview.

## Goals / Non-Goals

**Goals:**

- Keep every business decision in pure, framework-independent Go domain code.
- Make HTTP handlers translate commands and DTOs without reimplementing policy.
- Make PocketBase adapters persist domain decisions atomically and enforce relation integrity.
- Represent token, contract, lesson, and financial transitions explicitly enough to audit and correct.
- Give the React application one authenticated policy source and role-scoped read models.
- Preserve UTC instants while explicitly supporting teacher-local business dates and wall-clock schedules.
- Keep feature modules small enough to meet repository code budgets without hiding complexity in generic helpers.

**Non-Goals:**

- Share executable business logic with `apps/landing`.
- Add online payments, invoices, accounting exports, tax rules, or currency conversion.
- Add external notifications, public invitations, registration, guardians, instruments, teaching notes, or statutory-holiday data.
- Add arbitrary products, prices, durations, plan composition, or a generic rules engine.
- Add automatic sanctions for unpaid records or ad hoc cancellation and no-show events.
- Add a general administrator realm or timezone migration workflow.
- Record actual learner or teacher arrival times or measure delivered lesson minutes.

## Decisions

### Use explicit Go domain types instead of framework records as business objects

Create focused packages under `apps/backend/internal`:

```text
businesspolicy   global Policy and immutable PolicySnapshot values
commercial      PlanType, Money, package and contract aggregates
lesson          lesson state, outcome, cancellation, and reschedule decisions
ledger          charge, settlement, credit, refund, and adjustment decisions
history         event vocabulary and role-safe event projections
scheduling      interval, availability, timezone, grid, buffer, and conflict rules
```

Aggregate methods accept values such as actor, current instant, policy snapshot, lesson state, and entitlement state. They return a decision containing the next state and immutable events. They do not call PocketBase, HTTP, clocks, or frontend code.

Application services load records, construct aggregates, call one domain operation, and persist its decision inside one transaction. Handlers authenticate, decode English DTOs, call the service, and encode role-safe DTOs.

Alternative: place rules in PocketBase hooks or endpoint handlers. Rejected because booking, availability reconciliation, lifecycle, settlement, and correction would duplicate the same precedence and entitlement knowledge.

Alternative: build a generic rule engine or interface hierarchy for future plan types. Rejected by YAGNI. The domain uses concrete aggregates and exhaustive plan enums for the three required plans.

### Define one immutable global policy value

`businesspolicy.Current()` returns one validated `Policy` assembled from named English constants. The policy contains:

| Field | Initial value |
| --- | --- |
| Currency | `PLN` |
| Ad hoc price | `8000` minor units |
| Package price | `26000` minor units |
| Regular lesson price | `5000` minor units |
| Lesson duration | 45 minutes |
| Start grid | 15 minutes |
| Participant buffer | 5 minutes |
| Learner booking minimum | 24 hours |
| Booking horizon | 14 days |
| Package token count | 4 |
| Package validity | 60 local calendar days |
| Teacher cancellation extension | 7 local calendar days |
| Contract monthly reschedules | 1 |
| Contract free cancellations | 2 |
| Contract replacement deadline | 30 local calendar days |
| Monthly payment due day | 5 |
| Contract ordinary end | 30 June |

The policy exposes a stable English JSON document through one authenticated read endpoint. Money uses integer minor units plus ISO currency. Durations use integer minutes or hours only at the DTO boundary and `time.Duration` inside Go.

Each commercial record stores a compact immutable policy snapshot containing only values that can affect its later lifecycle. A stable policy version identifies the full source policy. Runtime validation always uses the record snapshot for historical obligations and the current policy for new obligations.

Alternative: let the app hard-code display constants. Rejected because the user requires backend authority.

Alternative: store editable policy rows in PocketBase. Rejected because the agreed first version has one deployment-controlled global policy and no administration UI.

### Use English throughout backend contracts and storage

Go identifiers, package names, database fields, enum values, event types, API fields, stable error codes, and technical error text use English. The React application owns Polish user copy and maps stable codes and values at the presentation boundary.

Do not persist Polish labels in snapshots or events. Store values such as `regular_contract`, `pending_settlement`, `learner_no_show`, `lesson_rescheduled`, and `intentionally_unpaid`.

Alternative: store localized event descriptions for direct display. Rejected because they cannot support consistent localization or stable machine processing.

### Model plan precedence in one commercial eligibility service

`commercial.Eligibility` evaluates one active assignment in this order:

```text
active regular contract -> reject flexible booking
available valid package token -> package booking
neither -> ad_hoc booking
```

Callers cannot request a lower-priority plan. Teacher-on-behalf booking uses the same service and differs only in the minimum-notice override. Package creation and contract activation call the same overlap validator. Assignment deactivation calls an obligation validator from the same package.

Conversions are explicit domain commands. Package purchase can convert selected future ad hoc lessons up to its token capacity. Contract activation can convert eligible future ad hoc occurrences. The transaction rejects activation if any prohibited future ad hoc or earlier-plan obligation remains.

Alternative: add a mutable `current_plan` field to assignments and trust it. Rejected because package expiry, token exhaustion, contract dates, and unresolved future lessons make plan eligibility time-dependent.

### Add dedicated commercial collections and relations

Add these PocketBase collections through versioned migrations:

```text
lesson_packages
  assignment, status, purchased_on, valid_through, closed_at,
  price_minor, currency, policy_version, policy_snapshot

package_tokens
  package, ordinal, state, lesson, changed_at

regular_contracts
  assignment, status, start_on, end_on, weekday, start_time,
  price_minor, currency, notice_at, effective_end_on,
  policy_version, policy_snapshot

contract_amendments
  contract, effective_on, price_minor, currency, created_at

contract_months
  contract, month, forecast_amount_minor, charge, generated_at

charges
  assignment, source_type, source_id, period,
  original_amount_minor, current_amount_minor, currency,
  settlement_state, due_on, paid_at

financial_entries
  charge, entry_type, amount_minor, currency, effective_on,
  related_lesson, related_entry, event_at

business_events
  assignment, aggregate_type, aggregate_id, event_type,
  actor_role, actor_id, event_at, prior_state, new_state,
  reason, internal_note, corrects_event
```

Extend `lessons` with:

```text
plan_type, package_token, contract, original_local_date,
original_start_at, policy_version, policy_snapshot,
unit_price_minor, currency, schedule_state, outcome,
outcome_at, individually_rescheduled, omission_reason
```

Use unique indexes for package token ordinal, one token per active lesson, contract occurrence identity, one contract month projection, and one charge source. Use relation validation to ensure every child matches the same assignment and participants.

Local business dates such as `purchased_on`, `valid_through`, `start_on`, `end_on`, `effective_on`, `month`, and `due_on` intentionally persist as normalized date strings. They are not instants. Their interpretation always uses the stored teacher timezone. Concrete lesson, actor, payment, cancellation, and event times persist as UTC instants.

Alternative: encode all commercial state into JSON fields on assignments or lessons. Rejected because uniqueness, authorization, concurrent token reservation, and queryable unresolved work require explicit relations.

Alternative: use only a generic event store without current projections. Rejected because PocketBase list queries and calendar reads need efficient, explicit current state. Immutable events remain the audit authority while transactional projections serve reads.

### Represent tokens as explicit stateful records

Create four token rows per package with an ordinal and one of `available`, `reserved`, `used`, `expired`, or `invalidated`. A reserved token has exactly one lesson relation. Selection uses deterministic lowest ordinal within the only eligible package.

The package aggregate owns allowed transitions:

```text
purchase                    -> 4 available
book                        -> available to reserved
complete/no-show/late cancel-> reserved to used
timely learner cancel       -> reserved to available or expired
reschedule                  -> remains reserved on same lesson
teacher cancel              -> reserved to available plus 7-day extension
validity reconciliation     -> available to expired
early close                 -> available to invalidated
correction                  -> compensating transition
```

Transactions lock lesson mutations with the existing process mutex and re-read token state inside `RunInTransaction`. Database uniqueness protects against accidental double linkage. Do not rely on a computed integer balance as the only integrity control.

The 60-day calculation uses teacher-local dates with purchase day as day one. `valid_through` equals `purchased_on + 59 calendar days`. Each teacher cancellation adds seven calendar days to the current `valid_through` value. Booking validates the proposed lesson start's teacher-local date against that inclusive boundary.

Alternative: decrement a package counter. Rejected because reserved versus used state, reversals, lesson linkage, and audit reconstruction would become implicit.

### Materialize contract occurrences through the explicit end date

Contract activation expands the selected local weekday and start time from `start_on` through `end_on`. It uses existing deterministic DST resolution and stores every concrete UTC interval. `end_on` defaults to the next applicable 30 June and remains explicit on the record.

Contract occurrence identity combines contract and original local date. Permanent schedule changes preserve that identity and original date while updating eligible intervals. Individual reschedules set `individually_rescheduled` and do not participate in later permanent schedule rewrites.

Use schedule states `scheduled`, `cancelled`, and `omitted`. Distant planned unavailability changes `scheduled` to `omitted` with a reason and makes it non-conflicting and non-billable. Reopening distant availability can restore that occurrence. Near-term cancellation uses `cancelled` and never restores silently.

Teacher calendar queries return the entire active contract series partitioned by whether start is inside the current horizon. Learner calendar queries return only starts inside the horizon. Contract occurrence conflicts still protect teacher and learner calendars beyond that read boundary.

Alternative: store only the recurrence and generate future lessons lazily. Rejected because teacher visibility, distant reservation conflicts, monthly forecasts, availability removal, and immutable occurrence history require stable concrete identities.

### Reconcile availability with a preview and atomic commit protocol

Availability create, update, enable, disable, and delete operations first call a preview endpoint with the proposed mutation. The server returns:

- the normalized proposed availability change
- all affected near-term lessons requiring `cancel` or `reschedule`
- all distant contract occurrences that will be omitted or restored automatically
- a short-lived opaque preview version derived from current relevant records

The commit request sends the proposed change, preview version, and one resolution for each near-term conflict. The server re-runs the preview in the transaction. Any new conflict or changed state returns a stable stale-preview error and commits nothing.

Each reschedule resolution includes a replacement start. Each cancellation resolution uses ordinary teacher cancellation effects. Distant omission and restoration use planned-availability effects and do not create teacher cancellation penalties, package extensions, or learner allowance effects.

The application presents every affected lesson before commit. This preserves explicit teacher intent and prevents availability from silently contradicting scheduled lessons.

Alternative: save availability and cancel conflicts automatically. Rejected because near-term package, contract, and settlement effects require an explicit teacher decision.

Alternative: keep the current reject-only conflict behavior. Rejected because it cannot perform the agreed one-step availability workflow.

### Keep cancellation and rescheduling as separate domain commands

`CancelLesson` has no replacement and releases the interval. `RescheduleLesson` requires a replacement and updates the same lesson identity atomically. Both return plan-specific entitlement and financial decisions plus events.

Learner timing compares the command instant with lesson start as exact instants. Exactly 24 hours is timely. Less than 24 hours is late. Learner self-service rescheduling is unavailable inside the cutoff, while cancellation remains available with plan-specific late consequences. Teacher lifecycle commands ignore this cutoff.

Flexible learner reschedules must choose a start inside the current 14-day horizon and at least 24 hours ahead. Package replacement starts must also remain within package validity. Contract learner reschedules use one allowance tied to the original teacher-local lesson month, permit only one learner reschedule per occurrence, and enforce the 30-local-day replacement deadline. A teacher can place a contract replacement beyond the learner horizon when it remains within the deadline.

Alternative: implement reschedule as client-side cancel followed by booking. Rejected because another request could claim the replacement, and token, allowance, settlement, and lesson identity could split across partial operations.

### Derive contract allowances from uncompensated events

Do not store manually editable cancellation or reschedule counters. To decide eligibility inside the transaction:

- count uncompensated eligible learner cancellation events for the contract
- count uncompensated learner reschedule events for the original local month
- check whether the occurrence already has an uncompensated learner reschedule

A correction appends an event pointing to the corrected event. Queries exclude compensated effects from current balances while returning both events in history. The expected volume is small enough for indexed assignment, contract, month, and event-type queries.

Alternative: increment and decrement integer counters. Rejected because corrections can drift from history and obscure why a balance changed.

### Separate schedule state, outcome, and settlement

A scheduled lesson reaching its end does not automatically assert attendance. The API derives `awaiting_outcome` when an ended non-cancelled lesson has no outcome. The teacher submits `completed` or `learner_no_show`; the form defaults visually to completed but no backend write occurs before submission.

Schedule state answers whether the interval remains reserved. Outcome answers what happened after an uncancelled start. Settlement answers whether money was acknowledged. This prevents one overloaded lesson status from conflating calendar, attendance, entitlement, and payment.

Package completion and no-show settle the reserved token as used. Contract no-show remains billable. Ad hoc no-show is logged without creating a penalty. Outcome correction uses a compensating history event and recalculates any entitlement projection in the same transaction.

Learner lateness does not change the scheduled end or settlement. Teacher lateness does not reduce the promised 45-minute entitlement, but actual arrival and delivery times remain an operational matter outside application state.

Alternative: automatically mark every past lesson completed. Rejected because it creates false attendance and payment assertions.

### Use a small ledger with immutable entries and mutable read projections

Store money as signed integer minor units and ISO currency. `charges` provide current query state. `financial_entries` record each creation, adjustment, payment, credit application, or refund. `business_events` record the broader actor-visible transition.

Ad hoc settlement belongs to the lesson and its charge source. It begins as `pending_settlement`, but only a completed lesson requires teacher payment resolution. Cancellation and no-show resolve it as `not_applicable` without creating debt. The teacher can record `paid` or `intentionally_unpaid`, later change unpaid to paid, and never loses the earlier event.

Contract monthly projections are idempotently reconciled on relevant authenticated reads and writes. Before a month begins, `contract_months` exposes a forecast. On or after the first teacher-local day, the first reconciliation creates the unique monthly charge and records its logical effective date as the first. A mid-month activation creates the current charge immediately.

Open unpaid charges adjust in place through immutable entries. An adjustment after payment creates an assignment credit. The next charge applies available credits deterministically oldest-first. A teacher can mark a credit refunded instead. `overdue` is derived from unpaid state and the teacher-local day-five deadline rather than stored as an irreversible state.

Package purchase writes a paid financial entry in the same transaction as the package and tokens. No external provider is involved.

Alternative: store only booleans such as `paid`. Rejected because pending versus intentionally unpaid, later payment, original amounts, credits, refunds, and corrections require explicit transitions.

### Store one append-only business event stream with protected notes

All domain decisions emit typed `business_events` inside their state transaction. Each event stores UTC `event_at`, aggregate identity, assignment, English event type, actor role, actor ID when present, prior and new state snapshots, and correction relation. System-created expiry and month projection events use actor role `system` with no account ID.

Ordinary business-event update and delete APIs do not exist. Administrative correction commands append compensating events and require an English backend reason code plus an optional teacher note.

Teacher history endpoints return all events for that teacher's assignments. Learner history endpoints return only events for that learner's assignments and omit `internal_note` and teacher-only correction detail. DTO builders, not frontend filtering, enforce this distinction.

Alternative: rely on PocketBase record timestamps and application logs. Rejected because they do not preserve semantic before/after state, correction linkage, authorization, or user-visible history.

### Use feature-specific command and read APIs

Keep teacher routes under `/api/teachers/*` and learner routes under `/api/learners/*`. Add endpoint groups for:

```text
business-policy
assignment commercial summary
package purchase, close, and correction
contract activate, amend, schedule, notice, renew, and correction
teacher-on-behalf booking
lesson outcome and settlement
monthly charge payment, refund, and correction
availability preview and resolved commit
role-scoped business history
```

Do not expose raw PocketBase collection records. Mutation payloads reject unknown fields and caller-supplied identity. Every browser mutation retains `X-Requested-With: fetch`. Stable errors use English codes, including plan precedence, token exhaustion, package expiry, cutoff, allowance exhaustion, unresolved obligations, stale preview, and correction reason errors.

Prefer focused read models over one unbounded calendar payload. The calendar response contains the current horizon view and concise commercial summaries. Full teacher contract series, ledger history, and business events use assignment-scoped endpoints with explicit pagination if current PocketBase conventions require it.

Alternative: continue expanding one calendar response with every event and financial record. Rejected because teacher full-series and immutable history grow independently and would make routine calendar refreshes expensive.

### Extend TanStack Query ownership without adding a second frontend store

Add keys for authenticated policy, assignment commercial summary, teacher full contract series, financial work, and role-scoped history to the existing centralized query-key registry. Add feature endpoint modules that continue to use the single transport boundary.

Mutation effect rules invalidate:

| Mutation | Required invalidation |
| --- | --- |
| Policy read | Stable for application session |
| Flexible booking or lifecycle | Both calendars, affected summary, slots, financial work, history |
| Package or contract mutation | Both calendars, affected summary, slots, full series, financial work, history |
| Availability resolved commit | Both calendars, all slots, affected series, summaries, financial work, history |
| Outcome or settlement | Calendars, affected summary, financial work, history |
| Persona clear | All resources owned by that persona |

Do not use optimistic updates for compound commercial mutations. Server transactions and refreshed read models remain authoritative.

### Organize React workflows around assignments and teacher work

Replace assignment duration controls with an assignment commercial workspace. The teacher dashboard has focused sections for:

- unresolved outcomes and ad hoc settlements
- unpaid and overdue charges
- near-term lessons inside the active horizon
- later regular-contract reservations
- assigned learners and plan actions

The learner calendar shows only near-term lesson details, available slots, current plan, token or allowance balance, charges, and owned history. Consequence confirmation dialogs use policy and server-provided previews, but the backend recomputes every decision.

Availability editing uses preview then commit. The teacher cannot submit a mutation until every near-term collision has a selected resolution. Distant automatic occurrence changes appear in the preview separately.

Keep Polish strings in frontend copy modules. Keep English enums and stable API error codes in DTO types and translation maps.

## Risks / Trade-offs

- [Full contract materialization increases records] → Contracts create roughly one lesson per week through June. Use indexed contract and start fields and avoid embedding full series in routine learner responses.
- [Availability recurrence changes can affect many contract lessons] → Preview all effects, partition near and distant changes, revalidate in one transaction, and keep reconciliation functions pure and bounded by explicit contract ends.
- [PocketBase transactions lack a portable row-lock API] → Retain the process-level mutation mutex, add unique indexes, re-read state in each transaction, and keep deployment single-writer until a database-level concurrency design replaces it.
- [Lazy monthly reconciliation means no stored charge exists before first post-boundary access] → Give projections a logical first-day effective date and call the idempotent reconciler from every relevant read and mutation. No external notification depends on wall-clock creation in this version.
- [Teacher-local dates are not UTC instants] → Document them as intentional wall-clock business fields in the scheduling constitution and keep every concrete timestamp in UTC.
- [English-only backend can leak technical text into Polish UI] → Map stable codes and enums in frontend copy modules and render generic Polish fallback text for unknown codes.
- [Immutable event snapshots can expose internal information] → Build separate teacher and learner DTOs and omit protected fields server-side.
- [Corrections can become complex] → Support only named compensating commands required by these specs. Do not expose arbitrary event or balance editing.
- [Breaking duration migration can alter unsupported existing data] → Audit existing non-45-minute records before migration, report them, and migrate only through an explicit documented rule.
- [One global policy requires deployment for changes] → Accept this deliberate first-version constraint and preserve obligation snapshots so deployment does not rewrite history.
- [Landing prices can diverge from backend policy] → Accept the explicit scope decision. Add a release checklist comparison without coupling the static landing to authenticated policy.

## Migration Plan

1. Audit current assignments and lessons for non-45-minute durations, future state, and event consistency.
2. Add new collections, indexes, relation fields, nullable lesson commercial fields, and event storage through reversible PocketBase migrations.
3. Backfill every current lesson as `ad_hoc` with the deployment policy snapshot. Preserve cancelled lessons. Mark ended non-cancelled lessons as awaiting teacher outcome rather than asserting attendance or payment.
4. Backfill existing lesson events into English business events with original UTC instants and migration actor attribution.
5. Change all assignments and lessons to 45 minutes, remove assignment duration mutation support, and remove teacher duration override support.
6. Make new lesson commercial fields required after backfill validation succeeds.
7. Deploy new backend reads and compound mutations together with updated canonical API documents and constitutions.
8. Deploy the React application in the same release because booking, duration, lifecycle, and calendar contracts are breaking.
9. Run backend migration tests, pure domain tests, relevant HTTP tests, React unit checks, critical Playwright flows, the full build, code-quality check, and production dependency audit.
10. Compare deployed backend prices and policy against the independent landing cennik as a release verification step.

Rollback before new commercial writes can use the down migrations. Rollback after new writes requires restoring the pre-deployment database backup and prior application version, because removing package, contract, payment, and event records would otherwise discard business history.

## Open Questions

None. The requirements interview resolved every decision needed for the specified behavior and task breakdown.
