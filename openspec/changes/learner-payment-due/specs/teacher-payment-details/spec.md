## Purpose

Lets a teacher store the bank transfer details that the teacher's learners see on the payments screen.

## ADDED Requirements

### Requirement: Teacher stores transfer details

A signed-in teacher SHALL read and replace the teacher's own transfer details: account holder, IBAN, and note. The server SHALL normalize the IBAN by removing spaces and converting letters to upper case. A 26-digit account number SHALL gain the `PL` prefix.

#### Scenario: Teacher saves a valid IBAN

- **WHEN** the teacher saves holder `Dominika Nowak` and IBAN `61 1090 1014 0000 0712 1981 2874`
- **THEN** the server stores `PL61109010140000071219812874` and returns the saved details

#### Scenario: Teacher clears the details

- **WHEN** the teacher saves an empty holder, IBAN, and note
- **THEN** the server stores empty values and learners receive `instructions` equal to null

### Requirement: Transfer details are validated

The server SHALL accept only a Polish IBAN of 28 characters with a valid ISO 13616 checksum. The holder SHALL be required when the IBAN is present and SHALL hold at most 140 characters. The note SHALL hold at most 300 characters. A violation SHALL return `400 invalid_payment_details`.

#### Scenario: Wrong checksum

- **WHEN** the teacher saves `PL61109010140000071219812875`
- **THEN** the API returns `400 invalid_payment_details` and keeps the previous details

### Requirement: Transfer details stay private

The transfer details SHALL NOT appear in auth responses or native record routes. A learner SHALL receive them only through the payment-due read of the learner's own assignment.

#### Scenario: Teacher session read

- **WHEN** the teacher reads `/api/teachers/auth/me`
- **THEN** the record contains no transfer detail field
