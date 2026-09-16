# Booking API

The booking API exposes flexible slots and plan-aware atomic booking for learners and assigned teachers.

## Common rules

- Learner requests use the learner session cookie.
- Teacher-on-behalf requests use the teacher session cookie.
- The server derives persona identity from the matching session cookie.
- The assignment must be active and include the learner.
- Every commercial lesson uses the policy duration of 45 minutes.
- Booking rejects a `duration_minutes` field from either persona.
- Starts use the policy 15-minute grid.
- A lesson must fit inside effective teacher availability.
- A lesson protects five minutes before and after its interval.
- Protected intervals conflict for either participant.
- Learner starts are at least 24 hours and at most 14 days ahead.
- Teacher flexible starts are at most 14 days ahead.
- A teacher may confirm a flexible start inside the learner minimum.
- The horizon constrains the lesson start, not the lesson end.
- Booking rechecks all rules inside one transaction.
- Booking writes lesson, entitlement, obligation, and events atomically.
- Unknown JSON fields and caller-supplied identity fields are rejected.
- See [scheduling errors](errors.md) for the error contract.

## Query learner slots

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/learners/assignments/{id}/slots` | Assigned learner | `SlotResponse` |

The route uses only the assignment identifier in its path. The handler does not read query parameters.
The response contains only starts inside the current policy window:

```json
{
  "teacher": "teacher-id",
  "assignment": "assignment-id",
  "policy_version": "v1",
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

Cancelled lessons do not remove slots through participant conflicts. Contract occurrences continue to block slots even when outside learner reads.

## Book a lesson

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/learners/assignments/{id}/book` | Assigned learner | `Lesson` with status `201` |
| `POST` | `/api/teachers/assignments/{id}/book` | Assigned teacher | `Lesson` with status `201` |

The learner request accepts only `start_at`. The teacher request accepts `start_at` and optional `confirm_short_notice`:

```json
{
  "start_at": "2030-01-07T09:00:00Z"
}
```

Teacher-on-behalf booking uses the same plan selection and 14-day horizon. A teacher request inside the 24-hour learner minimum requires explicit short-notice confirmation:

```json
{
  "start_at": "2030-01-06T12:00:00Z",
  "confirm_short_notice": true
}
```

The learner request does not accept `confirm_short_notice`. The caller cannot select `regular_contract`, `package`, or `ad_hoc`.
The backend applies `regular_contract`, then `package`, then `ad_hoc` precedence. An active contract rejects flexible booking. An available valid package token is reserved before ad hoc fallback.

The response has this shape:

```json
{
  "id": "lesson-id",
  "teacher": "teacher-id",
  "learner": "learner-id",
  "assignment": "assignment-id",
  "plan_type": "package",
  "package_token": "token-id",
  "contract": null,
  "start_at": "2030-01-07T09:00:00Z",
  "end_at": "2030-01-07T09:45:00Z",
  "duration_minutes": 45,
  "unit_price_minor": 0,
  "currency": "PLN",
  "policy_version": "v1",
  "schedule_state": "scheduled",
  "outcome": null,
  "protected_interval": {
    "start_at": "2030-01-07T08:55:00Z",
    "end_at": "2030-01-07T09:50:00Z"
  }
}
```

The response uses English machine values and RFC3339 UTC instants. A failed validation creates no lesson, token reservation, obligation, payment, or event.
