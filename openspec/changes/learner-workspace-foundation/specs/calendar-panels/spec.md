## MODIFIED Requirements

### Requirement: Learner panel books assigned teachers

The learner lessons screen at `/learners/lessons` SHALL display assigned teachers, their bookable slots for the rolling fourteen-day horizon, future lessons, and the learner's cancellation counter. It SHALL offer authorized controls for booking, future rescheduling, and cancellation. The route `/learners/calendar` SHALL redirect to `/learners/lessons`.

#### Scenario: Learner views assigned calendars

- **WHEN** an authenticated learner opens `/learners/lessons`
- **THEN** the screen displays only assigned teachers and their eligible slots

#### Scenario: Learner books from panel

- **WHEN** the learner confirms an eligible slot
- **THEN** the screen submits one booking request and displays the resulting lesson or a precise conflict or validation error

#### Scenario: Learner opens the former calendar route

- **WHEN** an authenticated learner opens `/learners/calendar`
- **THEN** the application replaces the location with `/learners/lessons` and renders the lessons screen
