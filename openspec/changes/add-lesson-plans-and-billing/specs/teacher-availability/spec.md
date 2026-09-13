## MODIFIED Requirements

### Requirement: Availability mutations preserve scheduled lessons

An availability mutation SHALL preview every affected scheduled lesson before commit. For lessons starting within the booking horizon, the teacher SHALL select `reschedule` with an eligible replacement or `cancel` for each conflict. The availability mutation, all lesson consequences, entitlements, settlements, and history SHALL commit atomically.

For contract occurrences starting beyond the horizon, the system SHALL automatically remove occurrences made unavailable without applying learner allowances or teacher cancellation consequences. Reopening availability beyond the horizon SHALL recreate eligible original contract occurrences and billing. The system SHALL never silently leave a scheduled lesson inside unavailable time.

#### Scenario: Near-term block overlaps lessons
- **WHEN** the teacher proposes an unavailable interval that overlaps lessons starting within 14 days
- **THEN** the application previews every conflict and the backend requires one resolution per lesson before commit

#### Scenario: One conflict remains unresolved
- **WHEN** the teacher submits an availability change without resolving every near-term conflict
- **THEN** the system rejects the whole mutation and retains the prior availability and lessons

#### Scenario: Future block removes contract occurrence
- **WHEN** an unavailable interval affects a contract occurrence starting after the current horizon
- **THEN** the system removes that occurrence from schedule and billing without using learner allowances and records the change

#### Scenario: Future block is removed again
- **WHEN** the teacher restores availability while the original contract occurrence remains beyond the horizon
- **THEN** the system recreates the eligible occurrence and restores its forecast or charge effect

#### Scenario: Near-term availability returns
- **WHEN** the teacher restores availability for a previously cancelled occurrence inside the horizon
- **THEN** the system does not recreate the lesson until the teacher explicitly creates or reschedules it

### Requirement: Slot grid uses fifteen-minute boundaries

The system SHALL source the global start grid from backend policy, currently fifteen minutes. A candidate SHALL be bookable only when its start aligns to that grid, the complete fixed lesson fits an available interval, and protected buffers do not conflict with another lesson.

#### Scenario: Lesson crosses availability end
- **WHEN** a candidate starts inside an available interval but its 45-minute interval exceeds the availability end
- **THEN** the candidate is not returned as bookable

#### Scenario: Candidate starts off grid
- **WHEN** a caller requests a start that is not aligned to the configured fifteen-minute grid
- **THEN** the system rejects booking

### Requirement: Availability queries use a rolling fourteen-day horizon

The system SHALL source the global booking horizon and learner minimum notice from backend policy. Flexible slots SHALL start at least 24 hours and no more than 14 days from the current instant. The upper boundary SHALL constrain lesson start rather than lesson end.

#### Scenario: Slot is inside both boundaries
- **WHEN** an eligible slot starts between 24 hours and 14 days from the current instant inclusive
- **THEN** the system may return the slot

#### Scenario: Slot is too soon
- **WHEN** an otherwise eligible learner slot starts less than 24 hours away
- **THEN** the system omits it from learner slot results

#### Scenario: Slot starts at horizon
- **WHEN** an eligible slot starts exactly 14 days away
- **THEN** the system may return it even though its end exceeds the horizon instant

### Requirement: Lesson buffers protect both participants

Every scheduled lesson SHALL use the global backend policy buffer, currently five minutes before and after its interval, for both participants. Buffers SHALL participate in participant conflict checks even when they extend beyond availability.

#### Scenario: Buffer overlaps another lesson
- **WHEN** a proposed lesson starts or ends within the configured buffer of another participant lesson
- **THEN** the system reports a conflict and does not offer or create the proposed lesson

## ADDED Requirements

### Requirement: Contract fixed times require effective availability

A teacher SHALL activate or permanently move a contract only when applicable generated occurrences fit effective availability or are explicitly removed by dated unavailability. A contract SHALL reserve its occurrence intervals before flexible slots are derived.

#### Scenario: Contract claims a flexible slot
- **WHEN** a contract occurrence exists inside otherwise available time
- **THEN** the system omits conflicting ad hoc and package slots

### Requirement: Teacher records every planned closure

The system SHALL NOT infer public holidays or statutory days off. The teacher SHALL express holidays, vacations, and planned breaks through availability changes. Occurrences removed outside the horizon SHALL not be billable.

#### Scenario: Public holiday has no teacher exception
- **WHEN** a contract occurrence falls on a statutory holiday that the teacher did not mark unavailable
- **THEN** the system keeps the occurrence scheduled and billable

### Requirement: Active obligations lock ordinary timezone changes

The system SHALL reject an ordinary teacher timezone change while the teacher has an active contract or future lesson. A timezone migration with schedule preview and history SHALL remain outside this change.

#### Scenario: Teacher changes timezone with future contract lessons
- **WHEN** the teacher submits a different timezone through ordinary profile editing
- **THEN** the system rejects the change and preserves all local schedule identities

