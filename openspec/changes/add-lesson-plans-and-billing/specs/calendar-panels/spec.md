## MODIFIED Requirements

### Requirement: Teacher panel manages availability and lessons

The teacher application SHALL display assignments, current plans, package balances, contract allowances, payment work, immutable history, availability, and lessons. It SHALL separate lesson starts inside the booking horizon from later contract reservations with clear labels and visual boundaries.

The teacher SHALL book flexible lessons on behalf of learners, create and manage packages and contracts, close ended lessons, settle payments, submit corrections, and resolve availability conflicts through authorized controls.

#### Scenario: Teacher views calendar
- **WHEN** an authenticated teacher opens `/teachers`
- **THEN** the panel loads only that teacher's commercial and scheduling resources and clearly separates near-term lessons from later contract occurrences

#### Scenario: Teacher views unresolved work
- **WHEN** lessons await outcome or ad hoc settlement, or charges are unpaid or overdue
- **THEN** the panel presents actionable items without automatically changing their state

#### Scenario: Teacher edits conflicting block
- **WHEN** a proposed unavailable interval affects near-term lessons
- **THEN** the panel previews each conflict and requires reschedule or cancellation before submitting the atomic change

#### Scenario: Teacher manages an assigned learner
- **WHEN** the teacher opens an assignment
- **THEN** the panel shows its active plan, eligible transitions, balances, future lessons, payments, and authorized history

### Requirement: Learner panel books assigned teachers

The learner application SHALL display current plan status, eligible flexible slots, own near-term lessons, package balance and expiry, contract allowances, payment information, and authorized history. Detailed lessons and lifecycle controls SHALL be limited to starts inside the backend-configured horizon.

The panel SHALL offer booking only when plan precedence permits it and SHALL explain disabled operations using Polish copy derived from stable English backend codes.

#### Scenario: Learner views assigned calendars
- **WHEN** an authenticated learner opens `/learners/calendar`
- **THEN** the panel displays only owned assignments, current plan data, and lesson starts within the 14-day horizon

#### Scenario: Learner books from panel
- **WHEN** the learner confirms an eligible slot at least 24 hours away
- **THEN** the panel submits one booking and displays the resulting package or ad hoc lesson

#### Scenario: Contract learner opens booking
- **WHEN** an active contract has precedence
- **THEN** the panel does not offer ad hoc or package booking and explains the fixed contract schedule

#### Scenario: Package learner views balance
- **WHEN** a valid package has available or reserved tokens
- **THEN** the panel shows token states and the teacher-local final validity date

### Requirement: Panels localize UTC instants

The panels SHALL parse concrete API datetimes as UTC instants and display them for the viewer. They SHALL present teacher-local policy dates, contract wall-clock schedules, package validity dates, payment months, and deadlines according to the teacher timezone returned by the backend.

#### Scenario: Learner views teacher lesson
- **WHEN** the API returns a UTC lesson interval to a learner in another timezone
- **THEN** the panel displays the equivalent viewer-local interval without changing the stored instant or teacher-local commercial date

## ADDED Requirements

### Requirement: Application renders backend policy in Polish

The React application SHALL load authenticated backend policy through shared server-state queries. It SHALL use backend values for prices, duration, grid, buffer, cutoff, horizon, package terms, and contract limits. It SHALL render Polish labels without sending translated enum values back to the backend.

#### Scenario: Policy horizon changes
- **WHEN** a deployed backend returns a different global horizon
- **THEN** the application updates its range labels and controls without a duplicated frontend constant change

### Requirement: Outcome and settlement forms require submission

The teacher lesson-close form SHALL default to `completed`. An ad hoc settlement form SHALL default to `paid`. Neither default SHALL change backend state until the teacher submits the form. The panel SHALL distinguish awaiting outcome, pending settlement, intentionally unpaid, and paid records.

#### Scenario: Teacher leaves close form untouched
- **WHEN** an ended lesson appears and the teacher does not submit its form
- **THEN** the lesson remains awaiting outcome and no payment assertion is recorded

### Requirement: Learner notices show financial consequences

Before a learner cancellation or reschedule, the application SHALL display whether the action is timely and whether it returns or uses a package token, consumes a contract allowance, creates a credit, or remains billable. Ad hoc late changes SHALL state that they are logged without penalty.

#### Scenario: Contract allowances are exhausted
- **WHEN** the learner starts a timely cancellation after both free cancellations were used
- **THEN** the panel warns that the lesson remains billable before requesting confirmation

#### Scenario: Package cancellation is late
- **WHEN** the learner starts cancellation less than 24 hours before a package lesson
- **THEN** the panel warns that the reserved token will become used

### Requirement: History respects role-specific presentation

The teacher SHALL see full assignment history. The learner SHALL see only owned commercial and scheduling events without internal teacher notes. Both panels SHALL translate English machine event types into Polish display copy.

#### Scenario: Learner opens corrected event
- **WHEN** a teacher correction affects the learner's token or allowance
- **THEN** the learner sees both the original and correction in Polish without seeing the internal note

