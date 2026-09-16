## Purpose

Tracks manual commercial settlement, credits, refunds, and overdue information without collecting money or imposing automated access restrictions.

## ADDED Requirements

### Requirement: Ad hoc settlement requires teacher confirmation

An ad hoc lesson SHALL use payment state `pending_settlement` until the teacher submits a post-lesson settlement. The settlement form SHALL default its selected option to `paid`, but the backend SHALL record no payment conclusion until submission. The teacher MAY instead select `intentionally_unpaid`.

#### Scenario: Ad hoc lesson ends without teacher action
- **WHEN** an ad hoc lesson passes its end time
- **THEN** its payment remains `pending_settlement` and appears in the teacher's unresolved work

#### Scenario: Teacher accepts the default
- **WHEN** the teacher submits the settlement form with `paid` selected
- **THEN** the system records the paid state, actor, and payment time

#### Scenario: Teacher records nonpayment
- **WHEN** the teacher explicitly selects `intentionally_unpaid`
- **THEN** the lesson appears on the unpaid list without being confused with an unreviewed lesson

### Requirement: Ad hoc nonpayment can settle later

The teacher SHALL change `intentionally_unpaid` to `paid` after later payment. The system SHALL preserve the earlier state and both transition times. An unpaid ad hoc lesson SHALL NOT block later booking.

#### Scenario: Learner pays an old balance
- **WHEN** the teacher marks a previously unpaid ad hoc lesson as paid
- **THEN** it leaves the current unpaid list and retains the nonpayment and payment history

#### Scenario: Learner books with an unpaid item
- **WHEN** an otherwise eligible learner has an unpaid ad hoc lesson
- **THEN** the system does not reject the new booking because of that balance

### Requirement: Cancelled and missed ad hoc lessons have no charge penalty

A timely or late learner cancellation and a learner no-show for an ad hoc lesson SHALL be logged but SHALL NOT create a debt, cancellation fee, or additional penalty. A teacher cancellation SHALL also create no ad hoc debt. Each of these outcomes SHALL resolve any `pending_settlement` state as `not_applicable` so it does not remain in unresolved payment work.

#### Scenario: Learner cancels ad hoc inside cutoff
- **WHEN** the learner cancels an ad hoc lesson less than 24 hours before start
- **THEN** the system logs the late cancellation and creates no payable balance

#### Scenario: Learner misses ad hoc lesson
- **WHEN** the teacher records learner no-show for an ad hoc lesson
- **THEN** the system logs the outcome, marks settlement not applicable, and creates no payable balance

### Requirement: Contract charges begin as monthly obligations

The system SHALL create a contract monthly charge on the first teacher-local day of each month. A contract activated during a month SHALL create that month's charge immediately. Before creation, the system SHALL expose a forecast rather than a payable obligation.

#### Scenario: Future month is previewed
- **WHEN** a teacher or learner views a later contract month before its first day
- **THEN** the application labels its lesson count and amount as a forecast

#### Scenario: Month begins
- **WHEN** the first teacher-local day of the month arrives
- **THEN** the forecast becomes a monthly charge using the billable occurrence snapshots

#### Scenario: Mid-month contract starts
- **WHEN** a teacher activates a contract after the first day of a month
- **THEN** the system immediately creates a charge for matching occurrences from the contract start onward

### Requirement: Contract charges use billable occurrences

Each monthly charge SHALL equal the sum of that month's billable contract occurrence values. It SHALL exclude occurrences removed by planned teacher unavailability outside the horizon, eligible free learner cancellations, and teacher cancellations. It SHALL retain late cancellations, no-shows, and cancellations after learner allowances are exhausted.

#### Scenario: Month has five ordinary occurrences
- **WHEN** five 50 PLN contract occurrences remain billable
- **THEN** the monthly charge amount is 250 PLN

#### Scenario: Planned absence removes one occurrence
- **WHEN** the teacher records unavailability outside the horizon and one of five occurrences is removed
- **THEN** the forecast or charge contains four billable occurrences and totals 200 PLN

### Requirement: Contract charges track manual payment state

A monthly charge SHALL support `pending`, `paid`, and `intentionally_unpaid`. The system SHALL derive `overdue` when a charge remains unpaid after the fifth teacher-local calendar day. A later payment SHALL resolve unpaid or overdue presentation without erasing history.

#### Scenario: Fifth day passes
- **WHEN** a charge remains `pending` after the configured day-five deadline
- **THEN** the application presents it as overdue while retaining its stored settlement state

#### Scenario: Teacher confirms later payment
- **WHEN** the teacher records payment for an overdue charge
- **THEN** the charge becomes paid and retains its prior overdue history

### Requirement: Charge reductions create adjustments

If a qualifying cancellation or schedule change occurs before payment, the system SHALL reduce the open monthly amount. If it occurs after payment, the system SHALL create a credit for the next monthly charge. The teacher MAY record a refund instead of applying the credit. Every adjustment SHALL retain the original charge amount and reason.

#### Scenario: Unpaid charge loses one billable occurrence
- **WHEN** an eligible free cancellation removes a 50 PLN occurrence before payment
- **THEN** the current amount decreases by 50 PLN and the original amount remains in history

#### Scenario: Paid charge loses one billable occurrence
- **WHEN** a teacher cancellation removes a 50 PLN occurrence after payment
- **THEN** the system creates a 50 PLN credit for the next charge unless the teacher records a refund

#### Scenario: Teacher chooses refund
- **WHEN** the teacher records that an adjustment was refunded
- **THEN** the credit is not applied again and the refund event remains visible

### Requirement: Package purchase is a paid commercial record

Recording a package purchase SHALL represent confirmed full payment. The system SHALL NOT create pending package debt or permit an unpaid package to issue tokens.

#### Scenario: Package form is submitted
- **WHEN** the teacher successfully records a package purchase
- **THEN** its payment is recorded as paid in the same atomic operation that creates tokens

### Requirement: Ledger does not process money or enforce access

The application SHALL NOT initiate card, bank, cash, refund, or payout transactions. Unpaid and overdue records SHALL be informational and SHALL NOT automatically prevent lessons, booking, rescheduling, or contract continuation.

#### Scenario: Payment provider is unavailable
- **WHEN** any payment state changes
- **THEN** no external payment provider is called because settlement is teacher-recorded
