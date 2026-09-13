# Availability API

The availability API exposes teacher-owned recurring rules and concrete exceptions.

## Common rules

- Every availability mutation requires a teacher session.
- Every availability mutation requires `X-Requested-With: fetch`.
- The server derives the teacher identifier from the session cookie.
- Recurring times use the teacher's local wall clock.
- Recurring times use `HH:MM` on 15-minute boundaries.
- A rule end time may be `24:00`.
- A rule weekday uses `0` for Sunday through `6` for Saturday.
- Exceptions use RFC3339 input and persist as UTC instants.
- Exception kind is `available` or `unavailable`.
- An unavailable exception cannot overlap a scheduled lesson interval.
- A five-minute protected buffer does not block an availability exception.
- See [scheduling errors](errors.md) for the error contract.

## Recurring rules

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/availability/rules` | Teacher | `{ "availability_rules": AvailabilityRule[] }` |
| `POST` | `/api/teachers/availability/rules` | Teacher | `AvailabilityRule` with status `201` |
| `PATCH` | `/api/teachers/availability/rules/{id}` | Owning teacher | `AvailabilityRule` |
| `DELETE` | `/api/teachers/availability/rules/{id}` | Owning teacher | Empty response with status `204` |

`POST` requires all rule fields except `enabled`. `enabled` defaults to `true`.
`PATCH` accepts any non-empty subset of `weekday`, `start_time`, `end_time`, and `enabled`.

```json
{
  "weekday": 1,
  "start_time": "09:00",
  "end_time": "17:00",
  "enabled": true
}
```

The response has this shape:

```json
{
  "id": "rule-id",
  "teacher": "teacher-id",
  "weekday": 1,
  "start_time": "09:00",
  "end_time": "17:00",
  "enabled": true
}
```

## Concrete exceptions

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/availability/exceptions` | Teacher | `{ "availability_exceptions": AvailabilityException[] }` |
| `POST` | `/api/teachers/availability/exceptions` | Teacher | `AvailabilityException` with status `201` |
| `PATCH` | `/api/teachers/availability/exceptions/{id}` | Owning teacher | `AvailabilityException` |
| `DELETE` | `/api/teachers/availability/exceptions/{id}` | Owning teacher | Empty response with status `204` |

`POST` requires `start_at`, `end_at`, and `kind`. `note` is optional.
`PATCH` accepts optional `start_at`, `end_at`, `kind`, and `note` fields. A changed interval requires both `start_at` and `end_at`.

```json
{
  "start_at": "2030-01-07T09:00:00Z",
  "end_at": "2030-01-07T10:00:00Z",
  "kind": "unavailable",
  "note": "Holiday"
}
```

The response has this shape:

```json
{
  "id": "exception-id",
  "teacher": "teacher-id",
  "start_at": "2030-01-07T09:00:00Z",
  "end_at": "2030-01-07T10:00:00Z",
  "kind": "unavailable",
  "note": "Holiday"
}
```

Unavailable intervals take precedence over recurring and available intervals.
