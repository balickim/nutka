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

## Read the learner payment due

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/learners/assignments/{id}/payment-due` | Assigned learner | `PaymentDue` |

```json
{
  "assignment": "assignment-id",
  "currency": "PLN",
  "total_minor": 28000,
  "items": [
    { "charge": "charge-id", "kind": "contract_month", "period": "2026-10", "amount_minor": 20000, "due_on": "2026-10-05", "overdue": true },
    { "charge": "charge-id-2", "kind": "lesson", "lesson_start_at": "2026-10-02T15:00:00Z", "amount_minor": 8000, "overdue": false }
  ],
  "open_credit_minor": 0,
  "next_forecast": { "month": "2026-11", "lesson_count": 4, "amount_minor": 20000, "due_on": "2026-11-05" },
  "recent_payments": [
    { "charge": "charge-id-3", "kind": "contract_month", "period": "2026-09", "amount_minor": 20000, "paid_at": "2026-09-03T08:00:00Z" }
  ],
  "instructions": { "account_holder": "Dominika Nowak", "iban": "PL61109010140000071219812874", "note": "" }
}
```

- A contract charge is open when its current amount is above zero and its state is `pending` or `intentionally_unpaid`.
- An ad hoc charge is open when its state is `intentionally_unpaid`.
- An ad hoc charge is also open when its state is `pending_settlement` and its lesson has ended.
- The total is the sum of the current amounts of the open items.
- The total does not subtract `open_credit_minor`, because the ledger already applies credits to charges.
- Overdue items come first. Other items follow by due date or lesson start.
- `next_forecast` is the earliest future contract month without a charge, or null.
- `recent_payments` holds at most five paid charges, newest payment first.
- `instructions` holds the transfer details of the assignment teacher, or null when the teacher has no IBAN.
- An unrelated account receives `403 unauthorized` without payment data.

## Teacher transfer details

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/payment-details` | Teacher | `PaymentDetails` |
| `PUT` | `/api/teachers/payment-details` | Teacher | `PaymentDetails` |

```json
{ "account_holder": "Dominika Nowak", "iban": "PL61109010140000071219812874", "note": "" }
```

- The `PUT` route requires the `X-Requested-With: fetch` header and replaces all three fields.
- The server trims text, removes IBAN spaces, and converts IBAN letters to upper case.
- The server adds the `PL` prefix to a 26-digit account number.
- The IBAN must be a Polish IBAN with 28 characters and a valid ISO 13616 checksum.
- The holder is required when the IBAN is present. The holder holds at most 140 characters.
- The note holds at most 300 characters.
- A rule violation returns `400 invalid_payment_details`.
- Empty values remove the transfer details from the learner payment-due read.
- The `teachers` collection stores the fields as hidden fields. Auth responses and native routes do not return them.

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
