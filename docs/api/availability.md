# Availability API

The availability API exposes teacher-owned recurring rules and exceptions through an impact preview and resolved atomic commit.

## Common rules

- Every availability read and mutation requires a teacher session.
- Every availability mutation requires `X-Requested-With: fetch`.
- The server derives the teacher identifier from the session cookie.
- Recurring times use the teacher's local wall clock.
- Recurring times use `HH:MM` on policy grid boundaries.
- A rule end time may be `24:00`.
- A rule weekday uses `0` for Sunday through `6` for Saturday.
- Exceptions use RFC3339 input and persist as UTC instants.
- Exception kind is `available` or `unavailable`.
- A lesson interval must fit inside effective availability.
- Protected five-minute buffers participate in participant conflicts.
- See [scheduling errors](errors.md) for the error contract.

## Read rules and exceptions

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/availability/rules` | Teacher | `{ "availability_rules": AvailabilityRule[] }` |
| `GET` | `/api/teachers/availability/exceptions` | Teacher | `{ "availability_exceptions": AvailabilityException[] }` |

`AvailabilityRule` has this shape:

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

`AvailabilityException` has this shape:

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

## Preview an availability mutation

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/availability/preview` | Teacher | `AvailabilityPreview` |

The request describes one operation on a recurring rule or concrete exception. It accepts only the operation and normalized rule or exception fields.

```json
{
  "operation": "create",
  "exception": {
    "start_at": "2030-01-07T09:00:00Z",
    "end_at": "2030-01-07T10:00:00Z",
    "kind": "unavailable",
    "note": "Holiday"
  }
}
```

The operation values are `create`, `update`, `enable`, `disable`, and `delete` for a rule or exception.
The preview response contains the normalized proposal, an opaque `preview_version`, every near-term conflict, and distant contract effects.

```json
{
  "preview_version": "opaque-version",
  "proposal": { "operation": "create" },
  "near_term_conflicts": [
    {
      "lesson": "lesson-id",
      "plan": "package",
      "start_at": "2030-01-07T09:00:00Z",
      "allowed_resolutions": ["cancel", "reschedule"]
    }
  ],
  "distant_effects": [
    {
      "occurrence": "occurrence-id",
      "effect": "omit",
      "start_at": "2030-02-04T09:00:00Z"
    }
  ]
}
```

Near-term means a lesson start inside the current policy horizon. Distant effects concern materialized contract occurrences beyond that horizon.
The preview does not change availability, lessons, entitlements, charges, or history.

## Commit a resolved mutation

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/availability/commit` | Teacher | `AvailabilityCommitResponse` |

The request repeats the proposal, sends `preview_version`, and resolves every near-term conflict:

```json
{
  "preview_version": "opaque-version",
  "proposal": { "operation": "create_exception" },
  "resolutions": [
    { "lesson": "lesson-id", "action": "reschedule", "replacement_start_at": "2030-01-08T09:00:00Z" }
  ]
}
```

Each conflict requires `cancel` or `reschedule`. Rescheduling requires an eligible replacement start.
The backend recomputes the preview inside one transaction and rejects a stale version, changed conflict set, incomplete resolution set, or invalid replacement.

On success, the transaction commits the availability mutation, all near-term lifecycle effects, distant omissions or restorations, entitlement effects, charge effects, and business events together.
Planned distant omissions are non-billable and do not extend packages or consume learner allowances.
Restoring near-term availability does not recreate a cancelled lesson.
