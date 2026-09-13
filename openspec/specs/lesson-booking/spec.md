## Purpose

Provides authorized, atomic lesson booking for assigned teacher–learner pairs within the published availability and rolling scheduling horizon.

## Requirements

### Requirement: Learner books only an assigned teacher

A learner SHALL create a lesson only for an active teacher–learner assignment. The system SHALL reject requests for unassigned teachers without revealing whether the teacher has availability.

#### Scenario: Assigned learner books slot

- **WHEN** an authenticated learner submits an eligible slot for an active assignment
- **THEN** the system creates one scheduled lesson for that learner and teacher

#### Scenario: Unassigned learner books slot

- **WHEN** an authenticated learner submits a slot for a teacher without an active assignment
- **THEN** the system rejects the request and creates no lesson

### Requirement: Learner booking uses the assignment duration

A learner booking SHALL always use the active assignment's teacher-controlled default duration. The learner booking API SHALL reject a duration override from the learner. The default SHALL be a positive multiple of fifteen minutes and SHALL be recorded on the lesson.

#### Scenario: Learner books with assignment default

- **WHEN** a learner books an available slot for an assignment with a 45-minute default
- **THEN** the system creates a 45-minute lesson if the lesson interval and participant conflicts pass

#### Scenario: Learner attempts duration override

- **WHEN** a learner submits any duration override, including a valid 60-minute value
- **THEN** the system rejects the request and creates no lesson

### Requirement: Booking is atomic across both calendars

The system SHALL validate assignment, horizon, availability, duration, and teacher and learner conflicts within one atomic operation. A failed validation SHALL create no partial lesson or booking event.

#### Scenario: Concurrent requests target same slot

- **WHEN** two eligible booking requests race for the same teacher and interval
- **THEN** at most one request succeeds and every failed request receives a conflict response without partial records

#### Scenario: Learner conflict differs from teacher conflict

- **WHEN** a slot is free for the teacher but overlaps another scheduled lesson for the learner or either buffer
- **THEN** the system rejects the booking and identifies the calendar conflict without creating a lesson

### Requirement: Booking response exposes UTC data

Successful booking responses SHALL identify the lesson, participants, status, duration, UTC start and end instants, and protected buffer interval. Responses SHALL not expose session tokens.

#### Scenario: UI receives booked lesson

- **WHEN** booking succeeds
- **THEN** the API returns RFC3339 UTC instants that the client can localize for display
