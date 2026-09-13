## Purpose

Defines prepaid four-lesson packages as auditable token lots whose booking, validity, cancellation, renewal, and correction rules remain deterministic.

## ADDED Requirements

### Requirement: Teacher records a paid package purchase

Only the assigned teacher SHALL record a package purchase. Saving the purchase SHALL confirm full payment, create exactly four available tokens, record 260 PLN and its policy snapshot, and use the teacher-selected purchase date.

The purchase date SHALL default to the current teacher-local date. A normal purchase date SHALL be today or earlier and SHALL NOT be in the future.

#### Scenario: Teacher records today's purchase
- **WHEN** the teacher accepts the default purchase date
- **THEN** the system creates a paid package with four available tokens

#### Scenario: Teacher records a late entry
- **WHEN** the teacher chooses an earlier purchase date
- **THEN** the system calculates validity from that earlier date and records the actual creation event separately

#### Scenario: Teacher chooses a future date
- **WHEN** the teacher submits a purchase date after the current teacher-local date
- **THEN** the system rejects the purchase and creates no tokens or payment

### Requirement: Package validity uses teacher-local calendar days

The purchase date SHALL count as validity day one. A package SHALL remain valid through the end of its sixtieth teacher-local calendar day. A package bought on 13 September SHALL therefore be valid through 11 November in that teacher's timezone. Available tokens SHALL expire after that boundary without an automatic refund.

#### Scenario: Booking starts on the final valid date
- **WHEN** an eligible lesson starts before the end of the package's final local validity date
- **THEN** the system may reserve a token for that lesson

#### Scenario: Booking was made before expiry for a later lesson
- **WHEN** a requested lesson starts after the package validity period
- **THEN** the system rejects package funding even if the booking request occurred before expiry

#### Scenario: Daylight-saving offset changes
- **WHEN** the validity period crosses a daylight-saving transition
- **THEN** the package still receives 60 local calendar dates rather than a fixed display-hour count

#### Scenario: Tokens remain unused at expiry
- **WHEN** the package reaches the end of its final valid local date with available tokens
- **THEN** those tokens expire and create no automatic cash value or refund

### Requirement: Booking reserves one package token

An atomic package booking SHALL change one available token to reserved and link it to the lesson. A completed lesson, learner no-show, or learner cancellation within the cutoff SHALL settle that reserved token as used. The system SHALL never reserve more tokens than the package provides.

#### Scenario: Learner books the fourth lesson
- **WHEN** a package has one available token and an eligible booking succeeds
- **THEN** that token becomes reserved and the available balance becomes zero

#### Scenario: Concurrent fifth booking races
- **WHEN** concurrent requests contend for the final available token
- **THEN** at most one request reserves it and no package balance becomes negative

#### Scenario: Learner cancels too late
- **WHEN** the learner cancels a package lesson less than 24 hours before its start
- **THEN** the lesson releases its calendar interval and its reserved token becomes used

#### Scenario: Learner does not attend
- **WHEN** the teacher closes a package lesson as learner no-show
- **THEN** its reserved token becomes used

### Requirement: Timely learner changes preserve package value

A learner cancellation at least 24 hours before the lesson start SHALL return the reserved token if the package remains open. A timely reschedule SHALL move the same reserved token to the replacement lesson. Neither action SHALL extend package validity.

A learner SHALL NOT reschedule inside the cutoff. The learner MAY instead cancel, which settles the original token as used, and later create a separate booking with another eligible token.

#### Scenario: Learner cancels in time
- **WHEN** the learner cancels at or before the 24-hour cutoff
- **THEN** the token becomes available subject to the unchanged package expiry

#### Scenario: Learner reschedules repeatedly
- **WHEN** each requested replacement satisfies the cutoff, availability, horizon, and package-expiry rules
- **THEN** the learner may reschedule repeatedly while the same token remains reserved

#### Scenario: Returned token immediately expires
- **WHEN** a timely cancellation returns a token after the package's final valid lesson start has passed
- **THEN** the token expires and cannot fund another lesson

#### Scenario: Learner attempts late package reschedule
- **WHEN** the source package lesson starts less than 24 hours after the request
- **THEN** the system rejects rescheduling without changing the lesson or token

### Requirement: Teacher cancellation restores and extends a package

A teacher cancellation SHALL return the reserved token and extend the package's validity by seven local calendar days. The teacher cutoff SHALL NOT apply. Each teacher cancellation SHALL add a separate seven-day extension and history event.

#### Scenario: Teacher cancels one package lesson
- **WHEN** the teacher cancels a future package lesson at any time
- **THEN** the token becomes available and the package validity end advances by seven local calendar days

#### Scenario: Teacher cancels twice
- **WHEN** the teacher cancels two distinct package lessons
- **THEN** the package validity end advances by a total of 14 calendar days

#### Scenario: Teacher reschedules instead
- **WHEN** the teacher atomically reschedules a package lesson
- **THEN** the same token remains reserved and no cancellation extension applies

### Requirement: Package renewal requires no available prior tokens

The teacher SHALL record a new package only after every token in the previous package is reserved, used, expired, or invalidated. Future lessons SHALL retain their original package links. The system SHALL NOT allow unused token pools to accumulate in advance.

#### Scenario: Prior package has reserved future lessons only
- **WHEN** all prior tokens are used or reserved and none is available
- **THEN** the teacher may record the next package purchase

#### Scenario: Prior package has one available token
- **WHEN** the teacher attempts another purchase while one prior token remains available
- **THEN** the system rejects the purchase without creating a second pool

### Requirement: Teacher can close a package without deleting it

Only the assigned teacher SHALL close an open package early. The teacher SHALL first resolve every package-backed future lesson. Closure SHALL require a reason, invalidate remaining available tokens, optionally record refund information, and preserve the purchase and token history.

#### Scenario: Teacher closes resolved package
- **WHEN** no future lesson holds a package token and the teacher supplies a reason
- **THEN** the package closes, remaining tokens become invalidated, and the event records their count

#### Scenario: Future package lesson remains
- **WHEN** the teacher attempts closure while a future lesson still reserves a token
- **THEN** the system rejects closure and identifies the unresolved lesson

### Requirement: Package corrections use compensating events

An authorized teacher correction SHALL restore or remove token effects without deleting prior events. The correction SHALL require a reason and SHALL leave a deterministic current token balance.

#### Scenario: Teacher reverses erroneous cancellation
- **WHEN** a teacher corrects an erroneous event that used a token
- **THEN** a compensating event restores the token state and the original event remains visible
