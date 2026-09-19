## Purpose

Defines how the teacher panel exposes write operations by frequency and risk, and which input forms are allowed, so that daily work stays fast and rare corrective work stays deliberate and attributable.

## ADDED Requirements

### Requirement: Teacher write operations belong to one of three tiers

Every teacher write operation SHALL belong to exactly one tier. Tier A covers daily operations: record a lesson outcome, book a lesson, record a settlement. Tier B covers periodic operations: activate a contract, purchase or renew a package, change a recurring slot, submit a notice, amend a future price. Tier C covers corrective operations: correct an event, activate a contract for a past date, end a contract early, close a package with a refund.

#### Scenario: Tier assignment is complete

- **WHEN** the panel offers a teacher write operation
- **THEN** that operation appears in exactly one tier and uses that tier's exposure rule

### Requirement: Tier A actions complete in one step

A tier A action SHALL be reachable as a control on the row or card it affects, SHALL require no more than one confirmation, and SHALL report its result without a page change.

#### Scenario: Teacher records a lesson outcome

- **WHEN** the teacher marks today's lesson as completed from the Today screen
- **THEN** the panel records the outcome and reports success on the same screen

### Requirement: Tier B actions use a guided flow with a summary

A tier B action SHALL open a dedicated flow, SHALL collect only the fields the operation requires, and SHALL show a plain-language summary of the effect before the write. The summary SHALL state the dates, amounts, and recurrence the operation creates or changes.

#### Scenario: Teacher activates a weekly contract

- **WHEN** the teacher submits a start date, weekday, and time
- **THEN** the flow shows the first lesson date, the recurring slot, the price per lesson, and the contract end date before the write
- **AND** the write happens only after the teacher confirms that summary

#### Scenario: Teacher abandons a flow

- **WHEN** the teacher closes a tier B flow before confirming
- **THEN** no write occurs and the underlying screen is unchanged

### Requirement: Tier C actions are disclosed, justified, and labelled

A tier C action SHALL sit behind an explicit advanced-operations disclosure that is collapsed by default. It SHALL require a non-empty reason before the control becomes enabled. The panel SHALL record the reason with the write and SHALL label the resulting history entry as a correction.

#### Scenario: Teacher corrects a package token

- **WHEN** the teacher opens advanced operations and selects a token correction
- **THEN** the control stays disabled until a reason is supplied
- **AND** the resulting history entry names the corrected event and shows the reason

#### Scenario: Corrective controls are hidden by default

- **WHEN** the teacher opens a learner detail screen
- **THEN** no corrective control is visible until the teacher opens the advanced-operations disclosure

### Requirement: Default inputs use domain values, not storage values

Outside advanced operations, the panel SHALL NOT ask the teacher to type an identifier, a minor-unit amount, or a delimiter-encoded list. Lesson selection SHALL use a list of the teacher's lessons. Event selection SHALL use a list of history entries. Money SHALL use a major-unit field with the currency shown. Past outcomes SHALL use a per-date control.

#### Scenario: Teacher converts ad hoc lessons into a package

- **WHEN** the teacher purchases a package and wants to convert existing ad hoc lessons
- **THEN** the flow shows the assignment's eligible ad hoc lessons with their dates and lets the teacher select them
- **AND** the flow provides no free-text identifier field

#### Scenario: Teacher changes a contract price

- **WHEN** the teacher sets a future price
- **THEN** the field accepts a major-unit amount and displays the currency
- **AND** the panel converts to minor units before the request

### Requirement: Irreversible actions confirm with a named consequence

An action that cannot be undone from the panel SHALL confirm through an in-app dialog that names the specific consequence with concrete dates or amounts. The confirming control SHALL carry the action name. The panel SHALL NOT use browser `prompt` or `confirm` dialogs.

#### Scenario: Teacher submits a contract notice

- **WHEN** the teacher submits a notice on a contract
- **THEN** the dialog states the resolved end date of the contract
- **AND** the confirming control reads as the action, not as a generic acknowledgement
