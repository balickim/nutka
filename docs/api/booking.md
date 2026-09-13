# Booking API

The booking API exposes learner slot queries and atomic learner booking.

## Common rules

- Every booking request requires a learner session.
- The server derives the learner identifier from the session cookie.
- The assignment must be active and include the learner.
- The assignment supplies the booking duration.
- The learner cannot submit a duration override.
- Durations are positive multiples of 15 minutes.
- Starts use the 15-minute UTC grid.
- A lesson must fit inside effective teacher availability.
- A lesson protects five minutes before and after its interval.
- Protected intervals conflict for either participant.
- Booking rechecks assignment, availability, horizon, and conflicts in one transaction.
- Booking writes the lesson and creation event together.
- See [scheduling errors](errors.md) for the error contract.

## Query learner slots

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/learners/assignments/{id}/slots` | Assigned learner | `SlotResponse` |

The route uses only the assignment identifier in its path. The handler does not read query parameters.
The response has this shape:

```json
{
  "teacher": "teacher-id",
  "assignment": "assignment-id",
  "slots": [
    {
      "start_at": "2030-01-07T09:00:00Z",
      "end_at": "2030-01-07T09:45:00Z",
      "duration_minutes": 45,
      "protected_interval": {
        "start_at": "2030-01-07T08:55:00Z",
        "end_at": "2030-01-07T09:50:00Z"
      }
    }
  ]
}
```

Slots cover the current instant through 14 times 24 hours. A slot start must be after the current instant. A slot end must not exceed the horizon. Cancelled lessons do not remove slots through participant conflicts.

## Book a lesson

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/learners/assignments/{id}/book` | Assigned learner | `Lesson` with status `201` |

The request body accepts only `start_at`:

```json
{
  "start_at": "2030-01-07T09:00:00Z"
}
```

The server rejects a `duration_minutes` field, even when its value is valid. Offset inputs normalize to UTC in the response.
The decoder rejects unknown JSON fields.

The response has this shape:

```json
{
  "id": "lesson-id",
  "teacher": "teacher-id",
  "learner": "learner-id",
  "assignment": "assignment-id",
  "start_at": "2030-01-07T09:00:00Z",
  "end_at": "2030-01-07T09:45:00Z",
  "duration_minutes": 45,
  "status": "scheduled",
  "protected_interval": {
    "start_at": "2030-01-07T08:55:00Z",
    "end_at": "2030-01-07T09:50:00Z"
  }
}
```
