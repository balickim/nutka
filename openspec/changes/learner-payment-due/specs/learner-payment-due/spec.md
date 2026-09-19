## Purpose

Lets a learner see for one assignment how much to pay, by which date, to which account, and which payments the teacher already recorded.

## ADDED Requirements

### Requirement: Payment-due read lists open items

The payment-due read SHALL list each open item of the assignment. A contract charge is open when its current amount is above zero and its state is `pending` or `intentionally_unpaid`. An ad hoc charge is open when its state is `intentionally_unpaid`, or when its state is `pending_settlement` and its lesson has ended. The total SHALL equal the sum of the current amounts of the open items.

#### Scenario: Pending monthly charge

- **WHEN** a contract charge for `2026-10` has current amount 20000 and state `pending`
- **THEN** the read lists one `contract_month` item with period `2026-10`, amount 20000, and due date `2026-10-05`, and the total is 20000

#### Scenario: Future ad hoc lesson

- **WHEN** an ad hoc charge has state `pending_settlement` and its lesson has not ended
- **THEN** the read does not list the charge

#### Scenario: Paid charge

- **WHEN** every charge of the assignment has state `paid` or `not_applicable`
- **THEN** the read lists no open item and the total is 0

### Requirement: Payment-due read marks overdue items

An open contract item SHALL be overdue after its due date in the teacher timezone. The read SHALL list overdue items first and then order items by due date or lesson start.

#### Scenario: Charge after the due date

- **WHEN** the teacher-local date is `2026-10-06` and a pending charge is due on `2026-10-05`
- **THEN** the item has `overdue` equal to true and appears first

### Requirement: Payment-due read shows the next forecast

For an assignment with a regular contract, the read SHALL return the earliest future contract month without a charge as the next forecast. The forecast SHALL contain the month, the lesson count, the amount, and the due date.

#### Scenario: Contract with a forecast month

- **WHEN** the contract has a forecast for `2026-11` with 4 billable lessons and amount 20000
- **THEN** the read returns `next_forecast` with month `2026-11`, lesson count 4, amount 20000, and due date `2026-11-05`

### Requirement: Payment-due read shows recent payments and credit

The read SHALL return at most five paid charges of the assignment, newest payment first. The read SHALL return the open credit of the assignment separately and SHALL NOT subtract it from the total.

#### Scenario: Credit present

- **WHEN** the assignment holds an open credit of 5000
- **THEN** the read returns `open_credit_minor` equal to 5000 and the total stays the sum of open items

### Requirement: Payment-due read is assignment scoped

Only the assigned learner SHALL read the payment-due of an assignment. The read SHALL include the transfer details of the assignment teacher, or null when the teacher has no IBAN.

#### Scenario: Unrelated learner

- **WHEN** a learner requests the payment-due of another learner's assignment
- **THEN** the API returns `403 unauthorized` without payment data

#### Scenario: Teacher without transfer details

- **WHEN** the teacher of the assignment has no IBAN
- **THEN** the read returns `instructions` equal to null

### Requirement: Payments screen explains how to pay

`/learners/payments` SHALL show the total due, each open item with its month or lesson date, its amount, its due date, and an overdue badge. When transfer details exist, the screen SHALL show the account holder, the IBAN, a transfer title, copy controls for the IBAN and the title, and a ZBP transfer QR code. The screen SHALL NOT block any other learner action.

#### Scenario: Nothing to pay

- **WHEN** the total is 0
- **THEN** the screen shows one sentence that nothing is due and shows no QR code

#### Scenario: No transfer details

- **WHEN** the total is above 0 and `instructions` is null
- **THEN** the screen tells the learner to ask the teacher about the payment method

### Requirement: Start screen shows the amount due

The Start screen SHALL show a payment card with the total and the earliest due date only when the total is above zero. The card SHALL link to `/learners/payments`.

#### Scenario: Amount due

- **WHEN** the total is 20000 PLN
- **THEN** the Start screen shows `Do zapłaty: 200,00 zł` and a link to the payments screen
