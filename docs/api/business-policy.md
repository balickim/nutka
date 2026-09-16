# Business policy API

The business policy API exposes the current scheduling and commercial constants to authenticated application sessions.

## Read policy

| Method | Path | Auth | Response |
| --- | --- | --- | --- |
| `GET` | `/api/teachers/business-policy` | Teacher | `BusinessPolicy` |
| `GET` | `/api/learners/business-policy` | Learner | `BusinessPolicy` |

Guests receive `401` and no policy data. The public landing page does not call this endpoint.

```json
{
  "version": "v1",
  "currency": "PLN",
  "ad_hoc_price_minor": 8000,
  "package_price_minor": 26000,
  "regular_lesson_price_minor": 5000,
  "lesson_duration_minutes": 45,
  "start_grid_minutes": 15,
  "participant_buffer_minutes": 5,
  "learner_booking_minimum_hours": 24,
  "learner_change_cutoff_hours": 24,
  "booking_horizon_days": 14,
  "package_token_count": 4,
  "package_validity_days": 60,
  "teacher_cancellation_extension_days": 7,
  "contract_monthly_reschedules": 1,
  "contract_free_cancellations": 2,
  "contract_replacement_deadline_days": 30,
  "monthly_payment_due_day": 5,
  "contract_end_month": 6,
  "contract_end_day": 30
}
```

Field names and values use English machine terms. Clients translate display copy into Polish.
The version identifies the full policy source. Commercial records retain the values that governed their creation.
