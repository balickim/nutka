## Purpose

Provides an immutable, role-authorized record of every business transition and compensating correction without exposing teacher-only notes to learners.

## ADDED Requirements

### Requirement: Business transitions append immutable events

Every lesson creation, conversion, reschedule, cancellation, outcome, settlement change, package purchase, token transition, package extension, package closure, contract activation, schedule change, amendment, notice, renewal, charge, credit, refund, availability consequence, and administrative correction SHALL append an event.

Each event SHALL contain an English event type, UTC event instant, actor role, actor identifier when applicable, affected record identifiers, and sufficient before-and-after data to reconstruct the transition.

#### Scenario: Learner reschedules a package lesson
- **WHEN** the reschedule succeeds
- **THEN** one atomic history records the old interval, new interval, learner actor, package link, and unchanged reserved token

#### Scenario: System expires a token
- **WHEN** package validity ends with an unused token
- **THEN** a system-attributed event records the expiry transition and effective instant

### Requirement: Mutations and events are atomic

A business mutation and all required history events SHALL commit together or not at all. A failed event write SHALL leave the business state unchanged.

#### Scenario: Event persistence fails
- **WHEN** a lesson mutation cannot append its required event
- **THEN** the system rolls back the lesson, entitlement, and settlement changes

### Requirement: History cannot be edited or deleted

Ordinary APIs SHALL NOT update or delete business events. A correction SHALL append a compensating event that references the corrected event or record. It SHALL NOT erase or overwrite the earlier event.

#### Scenario: Teacher corrects token usage
- **WHEN** an authorized teacher reverses an erroneous token transition
- **THEN** both the original and compensating events remain queryable

### Requirement: Administrative corrections require reasons

Every administrative override, backdated contract, early contract end, ownership correction, token correction, or exceptional state repair SHALL include a non-empty reason. The event SHALL distinguish the correction from an ordinary business action.

#### Scenario: Teacher omits correction reason
- **WHEN** the teacher submits an administrative correction without a reason
- **THEN** the system rejects it and changes no business state

### Requirement: History visibility follows assignment ownership

The assigned teacher SHALL see the full history for that assignment. The assigned learner SHALL see only events that affect that learner's lessons, packages, contracts, charges, payments, credits, and notices. A learner SHALL NOT see teacher-internal notes. Unrelated accounts SHALL see no events.

#### Scenario: Learner opens own history
- **WHEN** an authenticated learner requests history for an owned assignment
- **THEN** the response includes relevant transitions and omits internal teacher notes

#### Scenario: Unrelated teacher requests history
- **WHEN** a teacher requests events for another teacher's assignment
- **THEN** the system rejects access without revealing event existence

### Requirement: Machine history uses English terms

Event types, actor roles, state values, field names, and stable error codes SHALL use English terms. Clients SHALL localize display copy and SHALL NOT persist translated machine values.

#### Scenario: Polish history view renders cancellation
- **WHEN** the API returns an event type such as `lesson_cancelled`
- **THEN** the application renders Polish copy while retaining the English machine value

