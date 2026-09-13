## MODIFIED Requirements

### Requirement: Assignment access is private

Only an authorized teacher or learner involved in an assignment SHALL read that assignment and its authorized commercial summary. Only the assigned teacher SHALL change assignment activity or create, close, amend, or correct a package or contract. Learners SHALL retain only their explicitly defined booking, lifecycle, payment-read, history-read, and contract-notice actions.

#### Scenario: Unrelated learner reads assignment
- **WHEN** a learner requests an assignment that does not include the learner
- **THEN** the system returns an authorization error and no assignment or commercial data

#### Scenario: Learner changes assignment activity
- **WHEN** a learner attempts to activate or deactivate an assignment
- **THEN** the system rejects the request and retains the teacher-controlled state

#### Scenario: Learner records package purchase
- **WHEN** a learner attempts to create package tokens
- **THEN** the system rejects the request and creates no package or payment

### Requirement: Removing an assignment protects existing history

Deactivating an assignment SHALL prevent new bookings while retaining lessons, plans, charges, payments, and events. The system SHALL reject deactivation while the assignment has an active contract, an open package with available tokens, a package-backed future lesson, or any future scheduled lesson. Unpaid historical obligations SHALL remain visible after deactivation and SHALL NOT by themselves block deactivation.

#### Scenario: Active contract blocks deactivation
- **WHEN** the teacher attempts to deactivate an assignment with an active regular contract
- **THEN** the system rejects the change and identifies the unresolved contract

#### Scenario: Package token blocks deactivation
- **WHEN** the assignment has an open package with an available token
- **THEN** the system rejects deactivation until the package is used, expires, or closes

#### Scenario: Resolved assignment deactivates
- **WHEN** no active commercial obligation or future lesson remains
- **THEN** the teacher may deactivate the assignment while all history and unpaid information remain authorized and queryable

## REMOVED Requirements

### Requirement: Assignment defines a default lesson duration

**Reason**: The commercial offer defines one global 45-minute lesson. Assignment-specific duration and teacher lesson overrides would violate the authoritative policy.

**Migration**: Remove `default_duration_minutes` from assignment mutation and response contracts after migrating supported records and lessons to 45 minutes. Remove the teacher duration control.
