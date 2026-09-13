## Purpose

Defines one authenticated backend contract for the commercial and scheduling constants that govern every Nutka application decision.

## ADDED Requirements

### Requirement: Backend policy is the application authority

The backend SHALL expose one global business-policy document to authenticated teacher and learner sessions. The React application SHALL use that document instead of duplicating policy constants. The public landing page SHALL remain outside this contract.

#### Scenario: Authenticated application loads policy
- **WHEN** an authenticated teacher or learner starts the React application
- **THEN** the application receives the current global policy from the backend and uses it for labels, controls, and validation hints

#### Scenario: Guest requests policy
- **WHEN** an unauthenticated caller requests the application policy
- **THEN** the system rejects the request without exposing authenticated application data

#### Scenario: Landing builds independently
- **WHEN** the public landing page builds or renders
- **THEN** it does not require the authenticated business-policy endpoint

### Requirement: Policy declares all current commercial constants

The policy SHALL declare PLN as the currency, ad hoc price 80 PLN, package price 260 PLN, regular lesson price 50 PLN, four package tokens, 60 package-validity days, and seven extension days per teacher cancellation.

The policy SHALL declare a 45-minute lesson duration, 15-minute start grid, five-minute participant buffer, 24-hour learner booking minimum, 24-hour learner change cutoff, and 14-day booking horizon.

The policy SHALL declare one regular-plan reschedule per original lesson month, two free cancellations per contract, a 30-day replacement deadline, monthly payment day five, and contract end month and day 30 June.

#### Scenario: Client renders current policy
- **WHEN** the application renders booking, package, contract, or payment controls
- **THEN** it uses the values supplied by the policy response

#### Scenario: Server validates a mutation
- **WHEN** any scheduling or commercial mutation reaches the backend
- **THEN** the backend applies the same authoritative policy values regardless of client-supplied values

### Requirement: Policy uses English API terms

Policy field names, enum values, stable error codes, and persisted machine values SHALL use English terms. The React application SHALL translate user-facing content into Polish.

#### Scenario: Polish application displays a plan
- **WHEN** the backend returns the plan value `regular_contract`
- **THEN** the application displays its Polish label without changing the backend value

### Requirement: Commercial records preserve policy snapshots

Every package purchase, regular contract, contract amendment, and ad hoc booking SHALL retain the price and material policy values applied at creation. A global policy change SHALL affect only later obligations unless an authorized amendment changes an active contract prospectively.

#### Scenario: Price changes after package purchase
- **WHEN** the global package price changes after a learner bought a package
- **THEN** the existing package retains its recorded price and token terms

#### Scenario: Horizon changes after booking
- **WHEN** the global booking horizon changes after a lesson was validly booked
- **THEN** the existing lesson remains valid and retains its creation policy snapshot

### Requirement: Teacher timezone controls calendar rules

The system SHALL evaluate local calendar dates, package validity, contract dates, payment months, and day-five deadlines in the assigned teacher's IANA timezone. The system SHALL persist concrete event and lesson instants in UTC.

#### Scenario: Calendar rule crosses a timezone offset change
- **WHEN** a contract or package spans a daylight-saving transition
- **THEN** local dates retain their teacher-timezone meaning while concrete instants remain UTC

