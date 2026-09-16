# Lesson lifecycle API

The lifecycle API reschedules, cancels, closes, and corrects plan-aware lessons.

## Common rules

- Teacher requests use the teacher session cookie.
- Learner requests use the learner session cookie.
- Only a lesson participant can mutate that lesson.
- Only the assigned teacher records outcomes and corrections.
- Every mutation requires `X-Requested-With: fetch`.
- A lesson must be scheduled with a future start for reschedule or cancellation.
- Started and past lessons are immutable for scheduling changes.
- Every commercial lesson remains 45 minutes.
- Duration fields are rejected on lifecycle requests.
- Learner changes at least 24 hours before start are timely.
- Learner changes less than 24 hours before start are late.
- Learner rescheduling requires a timely source lesson.
- Learner cancellation remains available inside the cutoff with plan consequences.
- Teacher changes bypass learner cutoff and allowances.
- Lifecycle state, entitlement, settlement, and history commit atomically.
- See [scheduling errors](errors.md) for the error contract.

## Reschedule a lesson

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/lessons/{id}/reschedule` | Assigned teacher | `Lesson` |
| `POST` | `/api/learners/lessons/{id}/reschedule` | Assigned learner | `Lesson` |

The request accepts only `start_at`:

```json
{
  "start_at": "2030-01-08T10:00:00Z"
}
```

The backend validates assignment, availability, grid, buffers, participant conflicts, and plan rules in one transaction.
Learner flexible replacements must start at least 24 hours ahead and inside the 14-day horizon.
Package replacements must start within package validity. Contract replacements must start within 30 teacher-local calendar days of the original start.
Each contract occurrence accepts one learner reschedule tied to its original local month allowance.
The same lesson identity remains linked to its package token or contract occurrence.

The response includes `schedule_state: "scheduled"`, the new UTC interval, plan, entitlement references, and policy snapshot.
The event records the prior and new UTC intervals and does not record a cancellation.

## Cancel a lesson

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/lessons/{id}/cancel` | Assigned teacher | `Lesson` |
| `POST` | `/api/learners/lessons/{id}/cancel` | Assigned learner | `Lesson` |

The request body may be `{}`. Identity fields and unknown fields are rejected.
Cancellation has no replacement and releases the protected interval. The retained response includes:

```json
{
  "schedule_state": "cancelled",
  "cancellation_initiator_role": "learner",
  "cancellation_initiator_id": "learner-id",
  "cancelled_at": "2030-01-06T12:00:00Z",
  "cutoff": "timely",
  "plan_effect": "package_token_returned"
}
```

Exactly 24 hours before start is timely. A late learner package cancellation uses the reserved token. A timely learner package cancellation returns the token without extending validity.
A teacher package cancellation returns the token and extends validity by seven teacher-local calendar days.
A timely contract cancellation consumes one free allowance and removes or credits its billable value. Late, exhausted, and no-show contract outcomes remain billable.
Ad hoc cancellation and no-show settle payment as `not_applicable` without creating debt.

## Record a lesson outcome

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/lessons/{id}/outcome` | Assigned teacher | `Lesson` |

The request accepts one outcome:

```json
{
  "outcome": "completed"
}
```

Allowed values are `completed` and `learner_no_show`. A lesson that ended without submission derives `awaiting_outcome` and remains unresolved teacher work.
The backend does not infer attendance, arrival time, or delivery time. Package outcomes settle the token as used. Contract no-show remains billable. Ad hoc no-show creates no debt.

## Correct a lifecycle transition

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/lessons/{id}/correction` | Assigned teacher | `Lesson` |

The request requires a named correction and a non-empty reason:

```json
{
  "correction": "restore_entitlement",
  "reason": "The cancellation was recorded against the wrong lesson."
}
```

Corrections restore applicable token, allowance, outcome, or settlement effects through a compensating event.
The original event remains queryable. The API does not expose arbitrary event or balance editing.

## Atomicity and authorization

- An invalid replacement leaves the original lesson and entitlement unchanged.
- A failed entitlement, charge, or event write rolls back the lifecycle mutation.
- An unrelated account receives an authorization error without resource details.
- A started lesson cannot receive an ordinary cancellation or reschedule.
- Corrections never delete or overwrite prior history.
