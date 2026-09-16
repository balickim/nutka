# Calendar API

The calendar API returns authorized assignments, availability, plan summaries, and horizon-scoped lesson views.

## Read a teacher calendar

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/calendar` | Teacher | `TeacherCalendarResponse` |

The teacher response includes that teacher's assignments, enabled recurring rules, availability exceptions, commercial summaries, near-term lessons, later contract reservations, and unresolved work counts.
It does not include unbounded event or ledger detail.

## Read a learner calendar

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/learners/calendar` | Learner | `LearnerCalendarResponse` |

The learner response includes that learner's assignments, owned commercial summaries, eligible near-term slots, owned payment information, owned history summary, and lessons whose starts are inside the current policy horizon.
It does not include later contract occurrences or teacher-only notes.

## Teacher response shape

```json
{
  "assignments": [],
  "availability_rules": [],
  "availability_exceptions": [],
  "commercial_summaries": [],
  "near_term_lessons": [],
  "later_contract_lessons": [],
  "unresolved_work": {
    "awaiting_outcome": 1,
    "pending_settlement": 2,
    "unpaid_charges": 1,
    "overdue_charges": 0
  }
}
```

`near_term_lessons` contains starts within the policy horizon. `later_contract_lessons` contains starts outside that horizon and remains visible only to the teacher.
Later occurrences preserve conflict protection even though they are not flexible slots.

## Learner response shape

```json
{
  "assignments": [],
  "commercial_summaries": [],
  "near_term_lessons": [],
  "payment_summary": [],
  "history_summary": []
}
```

The learner response contains no generic cancellation counters. Plan-specific balances appear in each commercial summary.

## Lesson shape

Each lesson contains `id`, `teacher`, `learner`, `assignment`, `plan_type`, `package_token`, `contract`, `start_at`, `end_at`, `duration_minutes`, `unit_price_minor`, `currency`, `policy_version`, `schedule_state`, `outcome`, and `protected_interval`.

`schedule_state` is `scheduled`, `cancelled`, or `omitted`. A non-cancelled ended lesson without an outcome derives `awaiting_outcome`.
Cancelled and omitted lessons remain available through focused history reads. They do not block new flexible bookings.

## Display time

- Clients parse concrete values as UTC instants.
- Clients localize concrete values for the viewer.
- Clients display recurring teacher rules in the teacher IANA timezone.
- Clients display package validity and contract dates in the teacher timezone.
- Clients do not rewrite persisted UTC values when formatting them.
