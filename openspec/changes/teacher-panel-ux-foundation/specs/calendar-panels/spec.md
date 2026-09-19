## MODIFIED Requirements

### Requirement: Teacher panel manages availability and lessons

The teacher panel SHALL spread its work across dedicated route spaces rather than one page. `/teachers/calendar` SHALL display the teacher's future lessons, lesson status, duration, and initiator-attributed cancellation counter, and SHALL offer authorized controls for future rescheduling and cancellation. `/teachers/availability` SHALL display the teacher's local weekly plan and dated exceptions and SHALL offer authorized edit controls. `/teachers` SHALL display the current day's lessons and the decision queue only. Every teacher route SHALL load data for the signed-in teacher only.

#### Scenario: Teacher views calendar

- **WHEN** an authenticated teacher opens `/teachers/calendar`
- **THEN** the panel loads only that teacher's assignments, lessons, and initiator-attributed cancellation counter

#### Scenario: Teacher views the panel root

- **WHEN** an authenticated teacher opens `/teachers`
- **THEN** the panel shows the current day's lessons and the decision queue and loads no availability rules

#### Scenario: Teacher edits conflicting block

- **WHEN** the teacher submits an unavailable interval that conflicts with a scheduled lesson on `/teachers/availability`
- **THEN** the panel displays the API conflict and keeps the prior interval and lesson visible
