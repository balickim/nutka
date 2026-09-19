## Purpose

Defines the teacher route spaces, the question each screen answers, and the data each screen loads, so the teacher panel stays readable and cheap to load as the number of learners grows.

## ADDED Requirements

### Requirement: Each teacher screen answers one question

The teacher panel SHALL provide six route spaces. `/teachers` answers "what must I do now". `/teachers/calendar` answers "when are my lessons". `/teachers/students` answers "who are my learners". `/teachers/students/{assignmentId}` answers "what is this learner's situation". `/teachers/billing` answers "what money decisions wait for me". `/teachers/availability` answers "when am I free". Each route SHALL require the teacher persona session.

#### Scenario: Teacher opens the panel root

- **WHEN** an authenticated teacher opens `/teachers`
- **THEN** the screen shows the teacher's lessons for the current day and a queue of items that need a decision
- **AND** the screen shows no plan, package, contract, or availability form

#### Scenario: Unauthenticated teacher opens a teacher route

- **WHEN** a visitor without a teacher session opens any teacher route
- **THEN** the panel redirects to `/teachers/login` and preserves the requested route as the post-login target

### Requirement: The Today screen contains only dated work

The Today screen SHALL list the teacher's lessons for the current local day with the learner name, start time, and duration. It SHALL list decision items derived from unresolved work, each with a learner name, a cause, and one primary action. The Today screen SHALL NOT contain configuration controls.

#### Scenario: Teacher has lessons and unresolved work

- **WHEN** the teacher has two lessons today and three unresolved financial items
- **THEN** the Today screen shows the two lessons in start order and the three items in the decision queue

#### Scenario: Teacher has no work today

- **WHEN** the teacher has no lessons today and no unresolved work
- **THEN** each block shows an empty state that names the next useful destination

### Requirement: The roster shows learner state, not learner forms

The `/teachers/students` screen SHALL list each assignment with the learner name, the active plan, the recurring weekly slot when a contract exists, the settlement state, and the assignment active flag. The roster SHALL NOT render package, contract, booking, or correction controls. Each row SHALL link to the learner detail route.

#### Scenario: Teacher reviews the roster

- **WHEN** the teacher opens `/teachers/students` with ten assignments
- **THEN** the screen shows ten rows of state and loads no per-assignment package, contract, or history request

#### Scenario: Teacher opens a learner

- **WHEN** the teacher selects a roster row
- **THEN** the panel navigates to `/teachers/students/{assignmentId}` for that assignment

### Requirement: Learner detail owns the commercial workspace

The `/teachers/students/{assignmentId}` screen SHALL present the commercial summary, the package state, the contract state, teacher booking, and the business history for one assignment. It SHALL load commercial data for that assignment only. It SHALL reject an assignment that does not belong to the signed-in teacher.

#### Scenario: Teacher opens an owned assignment

- **WHEN** the teacher opens the detail route for an assignment they own
- **THEN** the screen loads the commercial summary, packages, contracts, and history for that assignment only

#### Scenario: Teacher opens a foreign assignment

- **WHEN** the teacher opens the detail route for an assignment they do not own
- **THEN** the screen shows an authorization error and exposes no learner data

### Requirement: Billing is a work queue

The `/teachers/billing` screen SHALL present the unresolved financial work: ad hoc settlements, unpaid charges, overdue charges, refunds, and corrections. Each entry SHALL name the learner, the amount, and the cause. The Today decision queue SHALL derive from the same source and SHALL link to this screen.

#### Scenario: Teacher settles a charge from the queue

- **WHEN** the teacher records a payment for a queue entry
- **THEN** the entry leaves the queue and the Today decision count decreases by one

### Requirement: The panel exposes its screens in persistent navigation

The teacher panel SHALL show navigation to all six screens on every teacher route, and SHALL mark the current screen. On viewports narrower than 768 pixels, the navigation SHALL remain reachable without scrolling.

#### Scenario: Teacher moves between screens

- **WHEN** the teacher is on `/teachers/billing`
- **THEN** the navigation marks Billing as current and offers direct links to the other five screens
