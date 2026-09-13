## MODIFIED Requirements

### Requirement: Both personas manage future lessons

An authenticated teacher or the assigned learner SHALL reschedule or cancel a scheduled lesson before its start. The learner SHALL satisfy the 24-hour cutoff for a timely change and every plan-specific allowance. The teacher SHALL have no cutoff and SHALL use ordinary or administrative plan-aware actions. Neither persona SHALL change lesson duration.

Every reschedule SHALL apply assignment, plan, entitlement, availability, fixed duration, grid, buffer, horizon where applicable, package expiry, replacement deadline, and participant conflict rules.

#### Scenario: Learner reschedules eligible future lesson
- **WHEN** the assigned learner submits a valid replacement allowed by the lesson plan
- **THEN** the system atomically moves the lesson and records the plan-specific effect

#### Scenario: Learner submits duration override
- **WHEN** the learner includes a duration field while rescheduling
- **THEN** the system rejects the request and leaves the lesson unchanged

#### Scenario: Teacher submits duration override
- **WHEN** the teacher includes a duration field while rescheduling
- **THEN** the system rejects the request because commercial lesson duration is fixed

#### Scenario: Teacher changes lesson without learner cutoff
- **WHEN** the assigned teacher reschedules or cancels before lesson start
- **THEN** the operation applies teacher consequences without consuming learner allowances

### Requirement: Started and past lessons are immutable

The system SHALL reject reschedule and cancellation requests at or after a lesson's start instant. After lesson end, only the assigned teacher SHALL record or correct a lesson outcome and related settlement. Outcome and administrative correction events SHALL preserve the scheduled interval and prior history.

#### Scenario: Started lesson changes
- **WHEN** either persona attempts to reschedule or cancel a lesson at or after its start instant
- **THEN** the system rejects the request and leaves the lesson unchanged

#### Scenario: Teacher closes ended lesson
- **WHEN** an assigned teacher records an allowed outcome after lesson end
- **THEN** the system retains the scheduled interval and appends the outcome and entitlement effects

### Requirement: Cancellation is retained with initiator attribution

Cancelling a lesson SHALL retain the lesson record and SHALL record the initiating persona, initiating account identifier, UTC cancellation instant, cutoff classification, plan consequences, and released calendar interval. A cancelled lesson SHALL not block future availability.

#### Scenario: Learner cancels lesson
- **WHEN** an assigned learner cancels a future lesson
- **THEN** the lesson remains queryable with learner attribution, timely or late classification, and its plan effect

#### Scenario: Teacher cancels lesson
- **WHEN** an assigned teacher cancels a future lesson
- **THEN** the lesson remains queryable with teacher attribution and teacher-specific entitlement or settlement effects

### Requirement: Reschedule history is separate from cancellation

Every successful reschedule SHALL preserve one lesson identity and append an event with prior and new UTC intervals, original local schedule identity when applicable, plan, entitlement, initiator, and event time. A reschedule SHALL NOT masquerade as an ordinary cancellation.

#### Scenario: Reschedule keeps package token
- **WHEN** either persona validly reschedules a package lesson
- **THEN** the same reserved token remains linked and history identifies a reschedule rather than a token-returning cancellation

#### Scenario: Replacement slot becomes unavailable concurrently
- **WHEN** the target slot conflicts before the reschedule commits
- **THEN** the system leaves the original lesson and entitlement unchanged

### Requirement: Lifecycle writes are authorized and atomic

Only the assigned teacher or learner SHALL mutate a lesson. Reschedule, cancellation, outcome, settlement, entitlement, charge adjustment, and all required history SHALL commit atomically or update none.

#### Scenario: Unrelated account changes lesson
- **WHEN** an account outside the lesson participants submits a lifecycle mutation
- **THEN** the system rejects the request without changing the lesson, entitlement, settlement, or history

#### Scenario: Entitlement update fails
- **WHEN** a cancellation cannot apply its required token or charge transition
- **THEN** the system retains the original scheduled lesson and writes no partial event

## ADDED Requirements

### Requirement: Cancel and reschedule remain distinct actions

Cancellation SHALL release a lesson without selecting a replacement. Rescheduling SHALL require an eligible replacement and move the same lesson atomically. The UI and API SHALL preserve this distinction for allowance, token, settlement, and history decisions.

#### Scenario: Teacher lacks a replacement time
- **WHEN** the teacher resolves an availability conflict by cancelling
- **THEN** the lesson ends without a replacement and receives cancellation consequences

#### Scenario: Teacher knows a replacement time
- **WHEN** the teacher resolves an availability conflict by rescheduling
- **THEN** the original interval and replacement commit as one operation

### Requirement: Learner cutoff uses the exact start difference

A learner change SHALL be timely when submitted at least 24 hours before the lesson start. A change submitted less than 24 hours before start SHALL be late. The system SHALL use concrete instants for this comparison.

Learner self-service rescheduling SHALL require a timely source lesson and a replacement at least 24 hours after the current instant. Inside the cutoff, the learner MAY cancel with the plan-specific late consequence but SHALL NOT reschedule.

#### Scenario: Change occurs exactly at cutoff
- **WHEN** the learner submits a cancellation exactly 24 hours before lesson start
- **THEN** the system applies timely plan consequences

#### Scenario: Change occurs one instant after cutoff
- **WHEN** the learner submits less than 24 hours before lesson start
- **THEN** the system applies late plan consequences

#### Scenario: Learner attempts late reschedule
- **WHEN** the learner submits a replacement less than 24 hours before the source lesson starts
- **THEN** the system rejects rescheduling and leaves the learner free to submit a late cancellation

### Requirement: Teacher closes every ended lesson outcome

After lesson end, the teacher SHALL explicitly record `completed` or `learner_no_show`. Until submission, the lesson SHALL remain `awaiting_outcome` and appear in unresolved teacher work. The outcome form SHALL default to `completed` but SHALL require submission.

#### Scenario: Lesson end passes
- **WHEN** no teacher outcome was submitted
- **THEN** the application shows the lesson as awaiting outcome rather than assuming attendance

#### Scenario: Teacher confirms completed
- **WHEN** the teacher submits the default completed outcome
- **THEN** the lesson becomes completed and its plan entitlement settles

#### Scenario: Teacher records no-show
- **WHEN** the teacher selects learner no-show
- **THEN** the lesson records that outcome and applies the plan-specific consequence

### Requirement: Late arrival does not rewrite the scheduled lesson

The system SHALL NOT automatically move the scheduled end, price, token, charge, or contract allowance because the learner arrived late. Teacher lateness SHALL NOT reduce the learner's fixed 45-minute service entitlement. This change SHALL NOT record actual arrival or delivery times.

#### Scenario: Learner arrives after scheduled start
- **WHEN** the teacher records the lesson as completed
- **THEN** its original scheduled interval and ordinary completed plan effects remain unchanged

#### Scenario: Teacher delivers later because of teacher delay
- **WHEN** the teacher provides the full lesson after the scheduled start
- **THEN** the application records the agreed scheduled interval and completed outcome without shortening the 45-minute entitlement

### Requirement: Ordinary corrections preserve history

Only the assigned teacher SHALL correct an erroneous lifecycle outcome, settlement, or entitlement. A correction SHALL require a reason, restore applicable allowance or token state, and append a compensating event without deleting the original.

#### Scenario: Teacher reverses an erroneous learner cancellation
- **WHEN** the teacher submits a valid correction and reason
- **THEN** applicable token or contract allowance returns and both events remain visible

## REMOVED Requirements

### Requirement: Cancellation counters belong to initiating persona

**Reason**: Generic actor cancellation totals do not express package token effects or contract-specific cancellation allowances.

**Migration**: Replace dashboard totals with plan-specific token balances, contract allowance balances, and filtered immutable event history.
