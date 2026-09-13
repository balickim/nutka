# Lesson lifecycle API

The lifecycle API reschedules and cancels future lessons and exposes cancellation counters.

## Common rules

- Teacher requests use the teacher session cookie.
- Learner requests use the learner session cookie.
- Only a lesson participant can mutate that lesson.
- Every mutation requires `X-Requested-With: fetch`.
- A lesson must be scheduled and its start must be in the future.
- Started and past lessons are immutable.
- Rescheduling rechecks assignment, availability, horizon, duration, and participant conflicts atomically.
- Rescheduling updates the lesson and appends one event together.
- Cancellation retains the lesson record with status `cancelled`.
- Cancellation stores the initiating role, account identifier, and UTC instant.
- Cancelled lessons do not participate in future conflict checks.
- See [scheduling errors](errors.md) for the error contract.

## Reschedule a lesson

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `PATCH` | `/api/teachers/lessons/{id}/reschedule` | Assigned teacher | `Lesson` |
| `PATCH` | `/api/learners/lessons/{id}/reschedule` | Assigned learner | `Lesson` |

Teacher requests may contain `start_at`, `duration_minutes`, or both. A teacher may change duration without changing start time.
Learner requests must contain `start_at` and must not contain `duration_minutes`.
The decoder rejects unknown JSON fields.

```json
{
  "start_at": "2030-01-07T10:00:00Z",
  "duration_minutes": 60
}
```

The response is a `Lesson`. A reschedule event stores the prior and new UTC intervals, prior and new duration snapshots, initiating role and account, and event UTC instant.

## Cancel a lesson

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/lessons/{id}/cancel` | Assigned teacher | `Lesson` |
| `POST` | `/api/learners/lessons/{id}/cancel` | Assigned learner | `Lesson` |

The request body may be `{}` or omitted. Identity fields and unknown JSON fields are not accepted.

The cancelled response contains the lesson interval and these fields:

```json
{
  "status": "cancelled",
  "cancellation_initiator_role": "learner",
  "cancellation_initiator_id": "learner-id",
  "cancelled_at": "2030-01-06T12:00:00Z"
}
```

Cancellation appends a cancellation event with the prior UTC interval, duration, initiating role and account, and event UTC instant.

## Read cancellation counters

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/cancellation-counters` | Teacher | `CancellationCounters` |
| `GET` | `/api/learners/cancellation-counters` | Learner | `CancellationCounters` |

The response has this shape:

```json
{
  "teacher": 1,
  "learner": 0
}
```

The teacher resource counts only cancellations initiated by the authenticated teacher. The learner resource counts only cancellations initiated by the authenticated learner. Reschedules never increment either count.
