## Purpose

Models the many-to-many teacher–learner relationship that controls who can book whom and supplies configurable lesson defaults for each assignment.

## ADDED Requirements

### Requirement: Assignments support many-to-many relationships

The system SHALL allow one teacher to be assigned to many learners and one learner to be assigned to many teachers. Each assignment SHALL be addressable independently.

#### Scenario: Learner has multiple teachers

- **WHEN** a learner has assignments to two teachers
- **THEN** the learner can query each teacher's authorized availability independently

#### Scenario: Teacher has multiple learners

- **WHEN** a teacher has assignments to two learners
- **THEN** the teacher can manage each learner's lessons without mixing assignment data

### Requirement: Assignment access is private

Only an authorized teacher or learner involved in an assignment SHALL read that assignment. Only the teacher involved SHALL change its assignment configuration. A learner SHALL not access another learner's assignment, and a teacher SHALL not access another teacher's assignment.

#### Scenario: Unrelated learner reads assignment

- **WHEN** a learner requests an assignment that does not include the learner
- **THEN** the system returns an authorization error and no assignment data

#### Scenario: Learner changes assignment default

- **WHEN** a learner attempts to change the assignment default duration
- **THEN** the system rejects the request and retains the teacher-controlled value

### Requirement: Assignment defines a default lesson duration

Each assignment SHALL define a teacher-controlled default lesson duration. A new assignment SHALL default to 45 minutes. The default SHALL be a positive multiple of fifteen minutes, and the system SHALL reject invalid values.

#### Scenario: Teacher changes valid default

- **WHEN** a teacher sets an assignment default to 45 minutes
- **THEN** future slot and learner booking operations for that assignment use the new default, while teacher-controlled duration overrides remain on individual lessons

#### Scenario: New assignment receives default

- **WHEN** a teacher creates an assignment without a duration override
- **THEN** the system stores and uses a 45-minute default

#### Scenario: Invalid default is rejected

- **WHEN** a teacher sets a zero, negative, or non-multiple-of-fifteen duration
- **THEN** the system rejects the change and retains the prior default

### Requirement: Removing an assignment protects existing history

Removing an assignment SHALL prevent new bookings through that assignment while retaining existing lesson and event records.

#### Scenario: Removed assignment blocks new booking

- **WHEN** a learner attempts to book a teacher after their assignment is removed
- **THEN** the system rejects the booking and retains prior lesson history
