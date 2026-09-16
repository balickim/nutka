# Payments API

The payments API records manual settlement, contract charges, credits, refunds, and informational overdue state.

## Read financial work

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/financial-work` | Teacher | `FinancialWork` |
| `GET` | `/api/learners/financial-summary` | Learner | `FinancialSummary` |
| `GET` | `/api/teachers/assignments/{id}/financial-history` | Assigned teacher | Paginated financial entries |
| `GET` | `/api/learners/assignments/{id}/financial-history` | Assigned learner | Paginated owned entries |

Learner reads omit teacher-only notes and unrelated assignments. Teacher reads include records for that teacher's assignments.

## Settle an ad hoc lesson

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `POST` | `/api/teachers/lessons/{id}/settlement` | Assigned teacher | `LessonPayment` |

```json
{
  "settlement": "paid"
}
```

The settlement values are `paid` and `intentionally_unpaid`. An ad hoc lesson starts as `pending_settlement` and the form default does not write state.
Cancellation and no-show resolve settlement as `not_applicable` without debt. Later payment may change `intentionally_unpaid` to `paid`.

## Contract charges

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/contracts/{id}/months` | Assigned teacher | `{ "months": ContractMonth[] }` |
| `GET` | `/api/learners/contracts/{id}/months` | Assigned learner | `{ "months": ContractMonth[] }` |
| `POST` | `/api/teachers/charges/{id}/payment` | Assigned teacher | `Charge` |
| `POST` | `/api/teachers/charges/{id}/refund` | Assigned teacher | `Charge` |
| `POST` | `/api/teachers/charges/{id}/correction` | Assigned teacher | `Charge` |

Before a month begins, the API returns a forecast. On or after its first teacher-local day, reconciliation creates one charge from billable occurrence snapshots.
Mid-month activation creates the current charge immediately. A charge supports `pending`, `paid`, and `intentionally_unpaid`.
The API derives `overdue` after the fifth teacher-local calendar day. It does not store overdue as an irreversible settlement state.

Open charge reductions lower the current amount. Paid reductions create a next-charge credit or a teacher-recorded refund. Credits apply oldest-first.

## Ledger limits

- The API does not process cards, bank transfers, cash, or payouts.
- Unpaid and overdue records do not block booking or lifecycle actions.
- Every payment, adjustment, credit, refund, and correction appends immutable history.
