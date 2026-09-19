## Purpose

Defines the states every data block must express and the feedback every write must produce, so that loading, emptiness, failure, and success look and behave the same across every screen of the application.

## ADDED Requirements

### Requirement: Every data block expresses four states

A block that reads server data SHALL express loading, empty, error, and ready. The loading state SHALL reserve the layout of the ready state. The empty state SHALL contain one sentence and, when an action can change it, one control. The error state SHALL contain the domain cause and a retry control.

#### Scenario: A block loads

- **WHEN** a data block has no data yet
- **THEN** the block shows placeholder content in the shape of the ready state and does not shift the layout when data arrives

#### Scenario: A block is empty and actionable

- **WHEN** the teacher has no assignments
- **THEN** the roster shows one explanatory sentence and a control that leads to the next useful step

#### Scenario: A block fails

- **WHEN** a read fails
- **THEN** the block shows the domain cause from the error contract and a retry control
- **AND** the block does not show a raw status code or a transport message

### Requirement: Every mutation reports its result

A write SHALL report both outcomes. While the write runs, the control SHALL be busy and SHALL keep its label and size. On success, the panel SHALL show a transient confirmation. On failure, the panel SHALL show the domain cause next to the control that failed and SHALL keep the submitted values.

#### Scenario: A write succeeds

- **WHEN** a write completes
- **THEN** the panel shows a transient confirmation naming what changed
- **AND** the affected data block shows the new state

#### Scenario: A write fails

- **WHEN** a write fails
- **THEN** the panel shows the domain cause near the failing control and keeps the entered values for correction

#### Scenario: A write is in flight

- **WHEN** a write is in flight
- **THEN** the control reports a busy state to assistive technology and keeps its label

### Requirement: Policy limits are enforced by controls

A control governed by a business policy SHALL prevent an invalid value rather than describe the rule as body text. The panel SHALL make the rule available on demand next to the control.

#### Scenario: Teacher picks a lesson start

- **WHEN** the teacher picks a start time for a booking
- **THEN** the control offers only starts on the policy grid and inside the booking horizon
- **AND** the full rule is available from a control next to the field

### Requirement: Dialogs are accessible and dismissible

A dialog SHALL trap focus while open, SHALL close on the escape key, and SHALL return focus to the control that opened it.

#### Scenario: Teacher dismisses a dialog

- **WHEN** the teacher presses escape in an open dialog
- **THEN** the dialog closes without a write and focus returns to the control that opened it

### Requirement: Interface terms match the teacher's vocabulary

The interface SHALL name a teacher-learner relationship as the learner, not as the assignment. The interface SHALL use one term for each concept across every screen.

#### Scenario: Roster names a relationship

- **WHEN** the roster shows a teacher-learner relationship
- **THEN** the row names the learner and does not use the word for the underlying assignment record
