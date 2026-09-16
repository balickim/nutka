# Teacher unresolved work API

The unresolved work API lists teacher actions that require an explicit decision.

## Read unresolved work

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/unresolved-work` | Teacher | `UnresolvedWork` |

```json
{
  "awaiting_outcome": [
    { "lesson": "lesson-id", "assignment": "assignment-id", "ended_at": "2030-01-07T09:45:00Z" }
  ],
  "pending_settlements": [
    { "lesson": "lesson-id", "amount_minor": 8000, "currency": "PLN" }
  ],
  "unpaid_charges": [],
  "overdue_charges": []
}
```

The response includes only work for the authenticated teacher's assignments. Reading unresolved work does not change any state.
Outcome and settlement forms default visually to `completed` and `paid` respectively, but state changes require explicit mutation submission.
The endpoint does not include unbounded event or ledger history. Use the [history API](history.md) and [payments API](payments.md) for those records.
