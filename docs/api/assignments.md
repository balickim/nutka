# Assignments API

The assignments API exposes private teacher–learner assignments.

## Common rules

- Teacher requests use the teacher session cookie.
- Learner requests use the learner session cookie.
- A response contains only assignments owned by the resolved account.
- Assignment identifiers are opaque strings.
- New assignments use `active: true` and `default_duration_minutes: 45`.
- Durations are positive multiples of 15 minutes.
- Mutations require `X-Requested-With: fetch`.
- See [scheduling errors](errors.md) for the error contract.

## Read assignments

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments` | Teacher | `{ "assignments": Assignment[] }` |
| `GET` | `/api/learners/assignments` | Learner | `{ "assignments": Assignment[] }` |

`Assignment` has this shape:

```json
{
  "id": "assignment-id",
  "teacher": "teacher-id",
  "teacher_name": "Test Teacher",
  "learner": "learner-id",
  "learner_name": "Test Learner",
  "active": true,
  "default_duration_minutes": 45
}
```

Teacher responses include active and inactive assignments. Learner responses include active and inactive assignments.
Names appear only for the authorized assignment. Assignment responses never include email addresses.

## Change assignment

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `PATCH` | `/api/teachers/assignments/{id}` | Assigned teacher | `Assignment` |

The request body accepts one or both fields:

```json
{
  "active": false,
  "default_duration_minutes": 60
}
```

The request must contain at least one supported field. The server rejects caller-supplied identity fields. A learner cannot change an assignment.

The body cannot contain `teacher`, `learner`, `actor_id`, or `actor_role`.

Deactivating an assignment blocks new bookings and retains existing lesson history.
