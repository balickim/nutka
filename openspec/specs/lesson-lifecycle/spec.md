## Purpose

Preserves lesson history while allowing both personas to manage future appointments and tracks cancellation responsibility for dashboard counters.

## Requirements

### Requirement: Both personas manage future lessons

An authenticated teacher or the assigned learner SHALL reschedule or cancel a lesson whose start instant is in the future. The system SHALL allow the teacher to change the lesson duration during rescheduling, including without changing its start time. The learner SHALL retain the current lesson duration. The system SHALL apply the same assignment, availability, duration, buffer, and conflict rules to rescheduling as to booking.

#### Scenario: Learner reschedules future lesson

- **WHEN** the assigned learner submits a valid future time without a duration override
- **THEN** the system atomically updates the lesson and records a reschedule event

#### Scenario: Teacher changes future lesson duration

- **WHEN** the assigned teacher submits a valid future time and a positive multiple-of-fifteen duration, or changes only the duration
- **THEN** the system atomically updates the lesson duration and records a reschedule event

#### Scenario: Learner attempts to change lesson duration

- **WHEN** the assigned learner submits a duration override while rescheduling
- **THEN** the system rejects the request and leaves the lesson unchanged

#### Scenario: Teacher cancels future lesson

- **WHEN** the assigned teacher cancels a future scheduled lesson
- **THEN** the system marks the lesson cancelled, records the teacher as initiator, and releases its calendar interval

### Requirement: Started and past lessons are immutable

The system SHALL reject reschedule and cancellation requests after a lesson has started or ended. The system SHALL determine this from the current UTC instant and SHALL not alter the lesson or event history on rejection.

#### Scenario: Started lesson changes

- **WHEN** either persona attempts to reschedule or cancel a lesson at or after its start instant
- **THEN** the system rejects the request and leaves the lesson unchanged

### Requirement: Cancellation is retained with initiator attribution

Cancelling a lesson SHALL retain the lesson record and SHALL record the initiating persona, initiating account identifier, and cancellation UTC instant. A cancelled lesson SHALL not block future availability.

#### Scenario: Learner cancels lesson

- **WHEN** an assigned learner cancels a future lesson
- **THEN** the lesson remains queryable with cancelled status and learner initiator fields, and its buffers no longer conflict with new bookings

### Requirement: Reschedule history is separate from cancellation

Every successful reschedule SHALL append an event with prior and new UTC intervals, duration, initiator, and event time. A reschedule SHALL not increment a cancellation counter.

#### Scenario: Reschedule does not count as cancellation

- **WHEN** either persona successfully reschedules a future lesson
- **THEN** the system records the reschedule event and leaves both cancellation counters unchanged

### Requirement: Cancellation counters belong to initiating persona

Each teacher and learner dashboard SHALL report the sum of retained lessons cancelled by that account. A cancellation initiated by one persona SHALL increment only that persona's counter.

#### Scenario: Teacher and learner counters diverge

- **WHEN** a teacher cancels one lesson and a learner cancels another lesson
- **THEN** the teacher counter increases by one, the learner counter increases by one, and neither counter includes the other account's cancellation

### Requirement: Lifecycle writes are authorized and atomic

Only the assigned teacher or learner SHALL mutate a lesson. Reschedule and cancellation SHALL atomically update the lesson and append the matching event, or update neither.

#### Scenario: Unrelated account changes lesson

- **WHEN** an account outside the lesson participants submits a lifecycle mutation
- **THEN** the system rejects the request without changing the lesson, counters, or history
