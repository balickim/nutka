# Packages API

The packages API lets an assigned teacher record prepaid four-token packages and manage their token lifecycle.

## Read package data

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments/{id}/packages` | Assigned teacher | `{ "packages": Package[] }` |
| `GET` | `/api/learners/assignments/{id}/packages` | Assigned learner | `{ "packages": Package[] }` |

Learner responses contain owned package fields and token states. Teacher responses contain all package history for the assignment.

## Record a purchase

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/assignments/{id}/packages` | Assigned teacher | `Package` with status `201` |

```json
{
  "purchased_on": "2030-09-13",
  "convert_lesson_ids": ["lesson-id-1", "lesson-id-2"]
}
```

`purchased_on` defaults to the current teacher-local date and cannot be future dated. The purchase is paid, costs 26000 minor units, creates four tokens, and stores a policy snapshot.
The purchase date counts as validity day one. `valid_through` is the sixtieth teacher-local calendar day.
Selected future ad hoc lessons can convert atomically, up to four tokens. Unconverted prohibited obligations reject the transaction.

## Package and token shape

```json
{
  "id": "package-id",
  "assignment": "assignment-id",
  "status": "open",
  "purchased_on": "2030-09-13",
  "valid_through": "2030-11-11",
  "price_minor": 26000,
  "currency": "PLN",
  "policy_version": "v1",
  "tokens": [
    { "id": "token-1", "ordinal": 1, "state": "available", "lesson": null }
  ]
}
```

Token states are `available`, `reserved`, `used`, `expired`, and `invalidated`.
Flexible booking reserves the lowest available ordinal. Completion, no-show, and late learner cancellation settle a reserved token as used.
Timely learner cancellation returns a token without extending validity. Teacher cancellation returns a token and extends validity by seven local days.

## Close or correct a package

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/packages/{id}/close` | Assigned teacher | `Package` |
| `POST` | `/api/teachers/packages/{id}/correction` | Assigned teacher | `Package` |

Closure requires a reason and no future lesson may reserve a token. Remaining available tokens become invalidated. Optional refund information is recorded without processing money.
Corrections require a reason and append compensating events. They do not delete purchase, token, or lesson history.

The teacher cannot purchase a new package while a prior package has an available token. An active regular contract also rejects package purchase.
