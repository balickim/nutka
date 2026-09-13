## Purpose

Separates teacher and learner identity realms so both personas can use Nutka concurrently with independent authorization and browser sessions.

## ADDED Requirements

### Requirement: Teacher and learner realms remain separate

The system SHALL authenticate teachers from the `teachers` auth collection and learners from the `learners` auth collection. The system SHALL rename an existing custom `users` auth collection to `teachers` without changing record identifiers, and SHALL create `teachers` on a fresh database.

#### Scenario: Existing users collection migrates

- **WHEN** the application starts with a custom `users` collection and no `teachers` collection
- **THEN** the system exposes the same records through `teachers` with their identifiers and credentials preserved

#### Scenario: Fresh database creates teacher realm

- **WHEN** the application starts without `users` or `teachers`
- **THEN** the system creates the `teachers` auth collection and keeps `learners` as a separate auth collection

### Requirement: Persona sessions use independent cookies

The system SHALL issue teacher sessions through `__Host-nutka_teacher_session` and learner sessions through `__Host-nutka_learner_session`. Each cookie SHALL be HttpOnly, Secure, SameSite=Lax, Path=/, host-only, and valid for 12 hours.

#### Scenario: Teacher login does not replace learner session

- **WHEN** a browser has a valid learner session and successfully authenticates as a teacher
- **THEN** the browser retains both cookies and each protected API resolves only its own persona

#### Scenario: Session token stays server-managed

- **WHEN** either persona authenticates or calls a session endpoint
- **THEN** the response contains no bearer token and browser JavaScript cannot read the session cookie

### Requirement: Persona route spaces are isolated

The system SHALL provide teacher routes under `/teachers/*` and learner routes under `/learners/*`. A valid session for one persona SHALL NOT authorize the other persona's protected routes.

#### Scenario: Learner requests teacher route

- **WHEN** a learner session requests a protected `/teachers/*` route
- **THEN** the system returns an authorization error or redirects to teacher login without exposing teacher data

#### Scenario: Teacher requests learner route

- **WHEN** a teacher session requests a protected `/learners/*` route
- **THEN** the system returns an authorization error or redirects to learner login without exposing learner data

### Requirement: Authentication remains closed

The system SHALL NOT provide public registration or invitation workflows in this change. Only an existing verified account or development provisioning path SHALL create a session.

#### Scenario: Guest attempts registration

- **WHEN** a guest submits a registration or invitation request
- **THEN** the system does not create an account and returns a not-found or unsupported-operation response

### Requirement: Scheduling APIs enforce resolved persona identity

Custom scheduling APIs SHALL derive the caller identity from the matching HttpOnly session cookie. A request-supplied teacher or learner identifier SHALL NOT override the resolved identity.

#### Scenario: Request spoofs another identity

- **WHEN** an authenticated caller submits a different persona identifier in a scheduling request
- **THEN** the system ignores the supplied identity or rejects the request and performs no unauthorized write
