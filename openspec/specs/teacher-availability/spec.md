## Purpose

Defines how teachers publish recurring availability and dated exceptions, and how Nutka derives safe booking slots across timezones and lesson buffers.

## Requirements

### Requirement: Weekly availability defaults to unavailable

Each teacher SHALL have a weekly availability plan that is unavailable unless an explicit recurring window allows a weekday and local wall-clock interval. The plan SHALL have no end date and SHALL be interpreted in the teacher's valid IANA timezone. New and migrated teachers SHALL default to `Europe/Warsaw`, and every stored timezone SHALL be a valid IANA identifier.

#### Scenario: No recurring window exists

- **WHEN** a learner requests a future date with no matching weekly availability rule
- **THEN** the system returns no bookable slots

#### Scenario: Timezone interprets recurring window

- **WHEN** a teacher sets a Monday window from 16:00 to 20:00 in `Europe/Warsaw`
- **THEN** the system applies that local wall-clock window on every matching Monday, including after a daylight-saving transition

#### Scenario: New teacher receives timezone default

- **WHEN** the system creates or migrates a teacher without a timezone value
- **THEN** the system stores `Europe/Warsaw` and uses it to interpret recurring rules

#### Scenario: Invalid timezone is rejected

- **WHEN** a caller attempts to store a timezone that is not a valid IANA identifier
- **THEN** the system rejects the change and retains the prior timezone

#### Scenario: Existing teacher sets valid timezone

- **WHEN** an existing teacher sets `America/New_York` as the teacher timezone
- **THEN** the system stores the valid IANA identifier and interprets future recurring rules in that timezone

### Requirement: Dated exceptions override recurring availability

Teachers SHALL be able to add dated available or unavailable exceptions. An unavailable exception SHALL take precedence over every available exception and recurring window that overlaps it.

#### Scenario: Available exception opens closed date

- **WHEN** a dated available exception covers a time with no recurring availability
- **THEN** the covered time becomes eligible for slot generation

#### Scenario: Unavailable exception closes available date

- **WHEN** an unavailable exception overlaps a recurring or dated available interval
- **THEN** the overlapping time produces no bookable slots

### Requirement: Availability mutations preserve scheduled lessons

The system SHALL reject a new or changed unavailable interval when it conflicts with a scheduled lesson interval. Protected buffers SHALL affect participant conflict checks only. The system SHALL leave the existing availability and lesson unchanged after rejection.

#### Scenario: Block overlaps lesson

- **WHEN** a teacher attempts to add an unavailable exception across a scheduled lesson interval
- **THEN** the system rejects the mutation with a conflict error and retains the prior calendar state

### Requirement: Slot grid uses fifteen-minute boundaries

The system SHALL generate candidate start times on fifteen-minute boundaries. A candidate SHALL be bookable only when the complete lesson duration fits an available interval and the protected buffers do not conflict with another scheduled lesson.

#### Scenario: Lesson crosses availability end

- **WHEN** a candidate starts inside an available interval but the lesson interval exceeds that interval's end
- **THEN** the candidate is not returned as bookable

#### Scenario: Candidate starts off grid

- **WHEN** a caller requests a start time that is not aligned to a fifteen-minute boundary
- **THEN** the system rejects the booking request

### Requirement: Availability queries use a rolling fourteen-day horizon

The system SHALL return bookable slots only from the current instant through the next fourteen calendar days. The system SHALL reject slot or booking requests outside this rolling horizon.

#### Scenario: Slot is inside horizon

- **WHEN** an eligible slot starts after the current instant and no later than fourteen days from the current instant
- **THEN** the system may return the slot

#### Scenario: Slot is outside horizon

- **WHEN** a caller requests a slot after the rolling fourteen-day horizon
- **THEN** the system returns a horizon validation error and does not create a lesson

### Requirement: Lesson buffers protect both participants

Every scheduled lesson SHALL reserve five minutes immediately before and after its lesson interval for both the teacher and learner. Buffers SHALL participate in conflict checks even when they extend beyond the weekly availability interval.

#### Scenario: Buffer overlaps another lesson

- **WHEN** a proposed lesson starts or ends within five minutes of an existing lesson for either participant
- **THEN** the system reports a conflict and does not offer or create the proposed lesson

### Requirement: Concrete schedule times use UTC instants

The system SHALL persist concrete lesson and dated-exception datetimes as UTC instants. The system SHALL preserve the teacher timezone and local wall-clock fields required to interpret recurring rules.

#### Scenario: Local time crosses daylight saving

- **WHEN** a recurring wall-clock rule maps to an ambiguous or nonexistent local instant
- **THEN** the system applies one documented deterministic resolution and exposes the resulting UTC instant consistently in API responses and UI
