## Purpose

Defines the exclusive commercial source for each lesson and the precedence that prevents ambiguous entitlements, prices, and settlements.

## ADDED Requirements

### Requirement: The system supports exactly three commercial lesson plans

Every in-system lesson SHALL use exactly one of `regular_contract`, `package`, or `ad_hoc`. The system SHALL NOT model the free trial, guardian relationships, instruments, or lesson-program content in this capability.

#### Scenario: Invited learner books without a product
- **WHEN** an active assignment has no active contract and no available package token
- **THEN** an eligible flexible booking is classified as `ad_hoc`

#### Scenario: New person takes a trial
- **WHEN** a prospective learner takes the free trial before invitation
- **THEN** the application creates no learner, lesson, payment, or entitlement record for the trial

### Requirement: Plans follow strict precedence

The system SHALL apply the precedence `regular_contract` before `package` before `ad_hoc` for each teacher–learner assignment. It SHALL reject an ad hoc or package booking while a regular contract is active. It SHALL reject an ad hoc booking while a valid package has an available token.

#### Scenario: Contract learner attempts flexible booking
- **WHEN** a learner with an active regular contract attempts an ad hoc or package booking
- **THEN** the system rejects the booking without creating a lesson or changing an entitlement

#### Scenario: Package learner attempts ad hoc booking
- **WHEN** a learner without a contract has an available token in a valid package and requests a flexible lesson
- **THEN** the system reserves a package token and does not create an ad hoc obligation

#### Scenario: Learner has neither higher plan
- **WHEN** a learner has no active contract and no available valid package token
- **THEN** the system may create an ad hoc lesson

### Requirement: Active contracts and usable packages do not overlap

The system SHALL reject contract activation while the assignment has a usable package or unresolved package-backed future lesson. The system SHALL reject package purchase while a regular contract is active. A teacher SHALL resolve the earlier arrangement before activating the next one.

#### Scenario: Teacher activates contract over package
- **WHEN** a package still has an available token or a future reserved lesson
- **THEN** contract activation fails and identifies the obligations that require resolution

#### Scenario: Teacher buys package during contract
- **WHEN** a teacher records a package purchase for an assignment with an active contract
- **THEN** the system rejects the purchase and records no payment

### Requirement: Existing ad hoc lessons require conversion or cancellation

Package purchase or contract activation SHALL NOT leave future ad hoc lessons beside the higher-priority plan. The teacher SHALL convert eligible lessons to the new plan or cancel them before the new plan becomes active. Conversion and activation SHALL commit atomically.

#### Scenario: Package covers existing ad hoc lessons
- **WHEN** the teacher selects up to four eligible future ad hoc lessons during package purchase
- **THEN** each selected lesson reserves one new package token and ceases to be ad hoc

#### Scenario: Too many ad hoc lessons remain
- **WHEN** future ad hoc lessons remain unconverted and uncancelled during higher-plan activation
- **THEN** the system rejects activation and leaves every lesson and plan unchanged

### Requirement: Plans belong to one assignment

Every package, contract, lesson, and commercial obligation SHALL belong to one authorized teacher–learner assignment. The system SHALL NOT transfer a plan or entitlement to another teacher or learner through an ordinary mutation.

#### Scenario: Learner has two teachers
- **WHEN** a learner owns tokens for one teacher and books another assigned teacher
- **THEN** the first teacher's tokens do not affect or fund the second teacher's booking

#### Scenario: Teacher requests exceptional transfer
- **WHEN** a teacher corrects plan ownership outside ordinary operations
- **THEN** the correction requires an administrative reason and preserves the original history

### Requirement: Every lesson retains its commercial source

A lesson SHALL retain its plan type, applied unit price, policy version, and package or contract identifier when applicable. A later policy or plan change SHALL NOT silently reclassify the lesson.

#### Scenario: Package expires after lesson booking
- **WHEN** a valid package token backed a lesson before the package later expires
- **THEN** the lesson remains linked to that package and follows its recorded lifecycle

