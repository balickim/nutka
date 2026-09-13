## Purpose

Gives teachers and learners focused calendar panels for availability and lesson management while keeping authorization and timezone conversion at API and UI boundaries.

## ADDED Requirements

### Requirement: Teacher panel manages availability and lessons

The teacher calendar panel SHALL display the teacher's local weekly plan, dated exceptions, future lessons, lesson status, duration, and cancellation counter. It SHALL offer authorized controls for future rescheduling and cancellation.

#### Scenario: Teacher views calendar

- **WHEN** an authenticated teacher opens `/teachers`
- **THEN** the panel loads only that teacher's assignments, availability, lessons, and initiator-attributed cancellation counter

#### Scenario: Teacher edits conflicting block

- **WHEN** the teacher submits an unavailable interval that conflicts with a scheduled lesson
- **THEN** the panel displays the API conflict and keeps the prior interval and lesson visible

### Requirement: Learner panel books assigned teachers

The learner calendar panel SHALL display assigned teachers, their bookable slots for the rolling fourteen-day horizon, future lessons, and the learner's cancellation counter. It SHALL offer authorized controls for booking, future rescheduling, and cancellation.

#### Scenario: Learner views assigned calendars

- **WHEN** an authenticated learner opens `/learners/calendar`
- **THEN** the panel displays only assigned teachers and their eligible slots

#### Scenario: Learner books from panel

- **WHEN** the learner confirms an eligible slot
- **THEN** the panel submits one booking request and displays the resulting lesson or a precise conflict or validation error

### Requirement: Panels localize UTC instants

The panels SHALL parse concrete API datetimes as UTC instants and display them in the viewer's local timezone. Recurring teacher rules SHALL display in the teacher's configured IANA timezone.

#### Scenario: Learner views teacher lesson

- **WHEN** the API returns a UTC lesson interval to a learner in another timezone
- **THEN** the panel displays the equivalent local start and end without changing the stored instant

### Requirement: Panels preserve independent auth state

Teacher and learner panels SHALL bootstrap and clear only their matching auth state. Signing out one persona SHALL not sign out the other persona in the same browser.

#### Scenario: Teacher signs out with learner tab open

- **WHEN** the teacher signs out while a learner panel is open in another tab
- **THEN** teacher routes require teacher login and the learner session and learner panel remain authenticated

### Requirement: Panels expose no public account workflow

The panels SHALL provide login and existing-account session flows only. They SHALL not render registration, invitation, or account-import controls.

#### Scenario: Guest opens panel

- **WHEN** a guest opens a teacher or learner panel route
- **THEN** the application directs the guest to the matching login route without offering public registration

### Requirement: Application titles identify persona spaces

The application SHALL use `nutka — przestrzeń ucznia` for `/learners` and `/learners/*`. It SHALL use `nutka — przestrzeń nauczyciela` for `/teachers` and `/teachers/*`. It SHALL use `nutka` for other application routes.

#### Scenario: Learner route sets learner title

- **WHEN** the browser resolves `/learners` or `/learners/calendar`
- **THEN** the HTML document title is `nutka — przestrzeń ucznia`

#### Scenario: Teacher route sets teacher title

- **WHEN** the browser resolves `/teachers` or `/teachers/availability`
- **THEN** the HTML document title is `nutka — przestrzeń nauczyciela`

#### Scenario: Root route sets neutral title

- **WHEN** the browser resolves `/`
- **THEN** the HTML document title is `nutka`
