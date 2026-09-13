## Purpose

Defines teacher-selected weekly contracts, their concrete lesson series, learner allowances, availability interaction, termination, renewal, and prospective amendments.

## ADDED Requirements

### Requirement: Teacher creates the fixed weekly contract

Only the assigned teacher SHALL activate a regular-plan contract. The teacher SHALL select a weekly weekday and local start time agreed outside the application. The interval SHALL fit the teacher's effective availability, use the policy grid and duration, and avoid participant conflicts.

The contract SHALL store explicit start and end dates in the teacher timezone. The ordinary end date SHALL be the next applicable 30 June.

#### Scenario: Teacher selects an eligible fixed term
- **WHEN** the teacher activates a contract at an available weekly time
- **THEN** the system records the contract and reserves its weekly occurrences through 30 June

#### Scenario: Fixed time is unavailable
- **WHEN** the selected weekly interval does not fit effective teacher availability
- **THEN** the system rejects activation without creating a contract or lesson

#### Scenario: Contract starts during a month
- **WHEN** the selected start date is after the first day of a month
- **THEN** the contract begins on that date and includes only later matching occurrences

### Requirement: Backdated contracts are administrative corrections

A standard contract start date SHALL be the current or a future teacher-local date. A teacher MAY backdate a contract only through an administrative correction that includes a reason and explicit outcomes for every generated past occurrence.

#### Scenario: Teacher backdates without outcomes
- **WHEN** a teacher submits a past start date through the ordinary activation flow
- **THEN** the system rejects activation and creates no retrospective obligations

#### Scenario: Teacher completes a controlled backdate
- **WHEN** the teacher supplies a reason and resolves every past occurrence
- **THEN** the system creates a complete auditable contract history without unresolved fictitious lessons

### Requirement: Contract activation materializes the weekly series

Contract activation SHALL create concrete weekly lesson occurrences from the start date through the end date. Each occurrence SHALL retain its contract link, original local schedule identity, current UTC interval, and billing state. Contract occurrences SHALL reserve participant intervals even beyond the flexible booking horizon.

#### Scenario: Contract spans several months
- **WHEN** activation succeeds for a weekly term through 30 June
- **THEN** the teacher calendar contains the full concrete series and those intervals block conflicting bookings

#### Scenario: Local time crosses daylight saving
- **WHEN** a weekly contract crosses a timezone offset transition
- **THEN** each occurrence keeps the selected local weekday and time while storing its resolved UTC interval

### Requirement: Learner contract visibility and actions use the booking horizon

The learner SHALL see detailed contract lessons only when their starts fall within the current booking horizon. The learner SHALL act only on visible near-term lessons. The teacher SHALL see the full contract series, clearly separated into starts inside and outside the horizon.

#### Scenario: Learner views a six-month contract
- **WHEN** the learner opens the calendar
- **THEN** the learner sees only contract lessons starting within the configured 14-day horizon

#### Scenario: Teacher views the same contract
- **WHEN** the teacher opens the calendar
- **THEN** the teacher sees both near-term lessons and the separate later-contract series

### Requirement: Learner receives one reschedule allowance per original month

The learner SHALL receive one timely contract reschedule for each teacher-local calendar month containing an original contract occurrence. The allowance SHALL belong to the original lesson month, not the request or replacement month. An unused monthly allowance SHALL expire rather than accumulate. Each occurrence SHALL be learner-rescheduled at most once.

#### Scenario: September lesson moves into October
- **WHEN** the learner reschedules a 30 September occurrence to 3 October
- **THEN** the operation uses September's allowance and leaves October's allowance unchanged

#### Scenario: Monthly allowance is already used
- **WHEN** the learner requests another reschedule for an occurrence with the same original month
- **THEN** the system rejects rescheduling and offers cancellation under the applicable cancellation rules

#### Scenario: Learner tries to move replacement again
- **WHEN** the learner requests a second reschedule of the same contract occurrence
- **THEN** the system rejects the request even if another calendar month has begun

#### Scenario: Prior month allowance was unused
- **WHEN** a new teacher-local month begins after no reschedule in the prior month
- **THEN** the learner has one current-month allowance rather than two accumulated allowances

### Requirement: Contract replacement has a 30-day deadline

A learner contract reschedule SHALL select a replacement atomically and SHALL require the replacement start no later than 30 teacher-local calendar days after the original start. The replacement SHALL satisfy availability and conflict rules. A replacement MAY occur after contract end when it remains within that deadline.

#### Scenario: Replacement occurs after 30 June
- **WHEN** a late-June occurrence moves to an eligible date within 30 days after its original start
- **THEN** the replacement remains linked to the ended contract and retains its settlement

#### Scenario: Replacement exceeds deadline
- **WHEN** the proposed start is later than the 30-day replacement deadline
- **THEN** the system rejects rescheduling and leaves the original occurrence unchanged

#### Scenario: Learner cannot see a needed later slot
- **WHEN** an eligible replacement lies beyond the learner's current horizon but within the 30-day deadline
- **THEN** the learner cannot select it directly and the teacher may perform the contract reschedule on the learner's behalf

### Requirement: Contract cancellation allowances affect billing

Each contract SHALL provide two learner cancellations at least 24 hours before lesson start without replacement. Each eligible cancellation SHALL consume one contract allowance and remove that occurrence's value from the monthly charge or create a credit after payment.

An unused allowance SHALL NOT transfer to another contract or become cash. Teacher cancellations SHALL NOT consume learner allowances.

#### Scenario: Learner uses first free cancellation
- **WHEN** the learner cancels at or before the cutoff with an allowance remaining
- **THEN** the occurrence is cancelled, one allowance is consumed, and its value is removed or credited

#### Scenario: Contract ends with unused allowances
- **WHEN** the contract ends while free cancellations remain
- **THEN** the remaining allowances expire without payment or transfer

### Requirement: Non-eligible contract cancellations remain billable

A learner cancellation inside 24 hours, a learner no-show, or a cancellation after both allowances are used SHALL release the calendar interval and remain billable. It SHALL NOT create an additional penalty. A late cancellation SHALL NOT consume a free cancellation allowance.

#### Scenario: Learner cancels late
- **WHEN** the learner cancels less than 24 hours before lesson start
- **THEN** the occurrence remains in the monthly charge, the allowance balance remains unchanged, and the event is logged

#### Scenario: Learner cancels after allowances are exhausted
- **WHEN** the learner cancels in time after using both contract allowances
- **THEN** the system warns that the occurrence remains billable and cancels it only after confirmation

### Requirement: Teacher changes have no learner cutoff or allowance cost

The teacher SHALL reschedule or cancel a contract occurrence at any time before it starts. A teacher reschedule SHALL preserve billing and learner allowances. A teacher cancellation SHALL remove or credit the occurrence value and SHALL NOT consume learner allowances.

#### Scenario: Teacher reschedules inside 24 hours
- **WHEN** the teacher selects an eligible replacement shortly before lesson start
- **THEN** the occurrence moves atomically without using learner allowances or changing its value

#### Scenario: Teacher cancels after monthly payment
- **WHEN** the teacher cancels an already paid occurrence
- **THEN** the system creates a credit or teacher-recorded refund and preserves the cancellation history

### Requirement: Teacher can change the permanent weekly time

The teacher SHALL change a contract's permanent weekday or time only to an eligible available interval. The change SHALL update unmodified future occurrences from the selected effective date. Individually rescheduled occurrences SHALL keep their replacement times. The change SHALL consume no learner allowance.

#### Scenario: Teacher changes next month's fixed time
- **WHEN** the teacher selects an eligible new weekly time and future effective date
- **THEN** the system updates applicable future occurrences atomically and logs the prior and new schedule

#### Scenario: One future occurrence was already moved
- **WHEN** a permanent schedule change spans an individually rescheduled occurrence
- **THEN** that occurrence retains its individual replacement interval

### Requirement: Either party can submit contract notice

The learner or teacher SHALL submit ordinary notice in the application. The notice period SHALL start on the first day of the following teacher-local month and the contract SHALL end on that month's final day. Lessons and charges SHALL continue through the effective end.

The teacher MAY record an earlier mutually agreed end as an administrative correction with a reason.

#### Scenario: Learner submits notice in October
- **WHEN** the learner submits notice on any October date
- **THEN** the notice period runs from 1 November through 30 November and the contract ends after that date

#### Scenario: Parties agree to end earlier
- **WHEN** the teacher records an earlier mutually agreed end and a reason
- **THEN** later occurrences are removed or corrected and the agreement history remains visible

### Requirement: Contract renewal is explicit

The system SHALL NOT renew a contract automatically after 30 June. The teacher MAY start a new contract through a renewal action that proposes the prior weekly time, revalidates availability, snapshots current policy, and creates new allowance balances.

#### Scenario: Contract reaches 30 June
- **WHEN** no renewal action occurs
- **THEN** the contract ends, its fixed interval is released, and no new weekly occurrences appear

#### Scenario: Teacher renews the learner
- **WHEN** the teacher accepts an eligible proposed time for the next term
- **THEN** the system creates a distinct contract with current price and fresh allowances

### Requirement: Contract price amendments are prospective

The teacher SHALL change an active contract price only through a recorded amendment effective on the first day of a future month. The amendment SHALL NOT alter earlier charges. The learner SHALL see the new price and amendment history. The application SHALL NOT require digital learner acceptance.

#### Scenario: Teacher schedules a November price change
- **WHEN** the teacher records an amendment during October with effect from 1 November
- **THEN** November and later obligations use the new price while prior obligations retain their snapshots

### Requirement: Nonpayment does not suspend a contract

An unpaid or overdue contract charge SHALL NOT automatically cancel lessons, block lifecycle actions, suspend the contract, or terminate it. A teacher decision to end the relationship SHALL use the explicit notice or early-end workflow.

#### Scenario: Charge becomes overdue
- **WHEN** the fifth day passes without recorded payment
- **THEN** scheduled contract lessons and learner actions remain available under ordinary rules
