# Regular contracts API

The regular contracts API lets an assigned teacher create and manage a fixed weekly contract and its materialized lesson series.

## Read contracts and series

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/assignments/{id}/contracts` | Assigned teacher | `{ "contracts": Contract[] }` |
| `GET` | `/api/learners/assignments/{id}/contracts` | Assigned learner | `{ "contracts": Contract[] }` |
| `GET` | `/api/teachers/contracts/{id}/series` | Assigned teacher | `{ "near_term": Occurrence[], "later": Occurrence[] }` |

Learners see only occurrence starts inside the current 14-day horizon. Teachers see the full series split into near-term and later starts.
All occurrences reserve participant intervals, including later starts.

## Activate a contract

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/assignments/{id}/contracts` | Assigned teacher | `Contract` with status `201` |

```json
{
  "start_on": "2030-09-16",
  "weekday": 1,
  "start_time": "17:00",
  "convert_lesson_ids": []
}
```

The weekly time uses the teacher timezone and policy grid. The ordinary end date is the next applicable 30 June.
Activation validates availability, participant conflicts, plan overlap, and future ad hoc obligations in one transaction.
Past starts require a correction reason and explicit outcomes for every generated past occurrence.

## Contract shape and allowances

```json
{
  "id": "contract-id",
  "assignment": "assignment-id",
  "status": "active",
  "start_on": "2030-09-16",
  "end_on": "2031-06-30",
  "weekday": 1,
  "start_time": "17:00",
  "price_minor": 5000,
  "currency": "PLN",
  "policy_version": "v1",
  "remaining_monthly_reschedules": 1,
  "remaining_free_cancellations": 2
}
```

Each occurrence retains its original local date, current UTC interval, contract link, schedule state, and billable value.
Learners receive one reschedule per original local month and two free cancellations per contract. Allowances do not accumulate or transfer.

## Contract actions

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/contracts/{id}/schedule` | Assigned teacher | Change future unmodified weekly occurrences. |
| `POST` | `/api/teachers/contracts/{id}/notice` | Assigned teacher | Submit ordinary notice. |
| `POST` | `/api/learners/contracts/{id}/notice` | Assigned learner | Submit ordinary notice. |
| `POST` | `/api/teachers/contracts/{id}/renew` | Assigned teacher | Create a distinct next-term contract. |
| `POST` | `/api/teachers/contracts/{id}/amendments` | Assigned teacher | Set a future-month price. |
| `POST` | `/api/teachers/contracts/{id}/correction` | Assigned teacher | Append a reasoned compensating correction. |

Ordinary notice ends the contract on the final day of the following teacher-local month. A teacher may record an earlier mutual end only with a reason.
Renewal is explicit and snapshots current policy. Price amendments take effect on the first day of a future month and do not alter prior charges.
The API never renews contracts automatically.
