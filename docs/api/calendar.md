# Calendar API

The calendar API returns authorized assignments, availability, retained lessons, and persona counters.

## Read a teacher calendar

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/calendar` | Teacher | `CalendarResponse` |

The teacher response includes that teacher's assignments, enabled recurring rules, all availability exceptions, all lessons, and that teacher's cancellation count.

## Read a learner calendar

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/learners/calendar` | Learner | `CalendarResponse` |

The learner response includes that learner's assignments, enabled rules and exceptions for active assigned teachers, all lessons for that learner, and that learner's cancellation count.

## Response shape

Both resources return:

```json
{
  "assignments": [],
  "availability_rules": [],
  "availability_exceptions": [],
  "lessons": [],
  "cancellation_counters": {
    "teacher": 0,
    "learner": 0
  }
}
```

Assignments contain `id`, `teacher`, `teacher_name`, `learner`, `learner_name`, `active`, and `default_duration_minutes`.
Assignment names appear only for the authorized assignment. Assignment responses never include email addresses.
Rules contain `id`, `teacher`, `weekday`, `start_time`, `end_time`, and `enabled`.
Exceptions contain `id`, `teacher`, `start_at`, `end_at`, `kind`, and optional `note`.

Lessons contain `id`, `teacher`, `learner`, `assignment`, `start_at`, `end_at`, `duration_minutes`, `status`, and `protected_interval`.
Cancelled lessons also contain cancellation initiator fields and `cancelled_at`.

Concrete `start_at`, `end_at`, `cancelled_at`, and protected interval values are RFC3339 UTC instants.
The API retains cancelled lessons in calendar responses. Cancelled lessons do not block new bookings.

## Display time

- Clients parse concrete values as UTC instants.
- Clients localize concrete values for the viewer.
- Clients display recurring teacher rules in the teacher's IANA timezone.
- Clients do not rewrite persisted UTC values when formatting them.
