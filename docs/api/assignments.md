# Assignments API

The assignments API exposes private teacher–learner assignments and concise commercial summaries.

## Common rules

- Teacher requests use the teacher session cookie.
- Learner requests use the learner session cookie.
- A response contains only assignments owned by the resolved account.
- Assignment identifiers are opaque strings.
- New assignments use `active: true`.
- Assignments do not expose or accept a lesson duration.
- Mutations require `X-Requested-With: fetch`.
- Request identity fields are rejected.
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
  "active": true
}
```

Teacher responses include active and inactive assignments. Learner responses include active and inactive assignments.
Names appear only for the authorized assignment. Assignment responses never include email addresses.

## Read a commercial summary

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments/{id}/commercial-summary` | Assigned teacher | `CommercialSummary` |
| `GET` | `/api/learners/assignments/{id}/commercial-summary` | Assigned learner | `CommercialSummary` |

The summary contains only assignment-owned data:

```json
{
  "assignment": "assignment-id",
  "active_plan": "package",
  "package": {
    "id": "package-id",
    "valid_through": "2030-11-11",
    "token_balance": {
      "available": 2,
      "reserved": 1,
      "used": 1,
      "expired": 0,
      "invalidated": 0
    }
  },
  "contract": null,
  "payments": {
    "pending": 0,
    "intentionally_unpaid": 0,
    "overdue": 0,
    "credit_minor": 0,
    "currency": "PLN"
  }
}
```

`active_plan` is `regular_contract`, `package`, or `ad_hoc`, or `null` when no current obligation exists.
Contract summaries expose status, start date, effective end date, unit price, currency, remaining monthly reschedules, and remaining free cancellations.
Learner summaries omit teacher-only notes and unrelated financial records.

## Change assignment activity

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `PATCH` | `/api/teachers/assignments/{id}` | Assigned teacher | `Assignment` |

The request body accepts only `active`:

```json
{
  "active": false
}
```

The body must contain one supported field. A learner cannot change assignment activity.

Deactivation blocks new bookings and retains lessons, plans, charges, payments, and history.
The server rejects deactivation while an active contract, open package token, package-backed future lesson, or future scheduled lesson remains.
Historical unpaid obligations do not alone block deactivation.
