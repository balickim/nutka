# Business history API

The business history API returns immutable, role-scoped events for scheduling and commercial transitions.

## Read history

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments/{id}/history` | Assigned teacher | Paginated `BusinessEvent[]` |
| `GET` | `/api/learners/assignments/{id}/history` | Assigned learner | Paginated owned `BusinessEvent[]` |

The teacher response contains all events for the assignment. The learner response contains only events affecting the learner's lessons, packages, contracts, charges, payments, credits, or notices.
Learner DTOs omit `internal_note` and teacher-only correction detail. Unrelated accounts receive `unauthorized` without event existence.

## Event shape

```json
{
  "id": "event-id",
  "assignment": "assignment-id",
  "aggregate_type": "lesson",
  "aggregate_id": "lesson-id",
  "event_type": "lesson_rescheduled",
  "actor_role": "learner",
  "actor_id": "learner-id",
  "event_at": "2030-01-06T12:00:00Z",
  "prior_state": { "start_at": "2030-01-07T09:00:00Z" },
  "new_state": { "start_at": "2030-01-08T09:00:00Z" },
  "reason": null,
  "corrects_event": null
}
```

Event types, actor roles, states, and field names use English machine values. Clients localize display copy.
Every business transition appends an event in the same transaction as its state change.
System expiry and reconciliation events use actor role `system` without an account identifier.

## Corrections and immutability

- Ordinary clients cannot update or delete business events.
- Administrative corrections require a non-empty reason.
- A correction appends a compensating event with `corrects_event`.
- Original and compensating events remain queryable.
- Current balances exclude compensated effects.
