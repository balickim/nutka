# Scheduling errors

Scheduling and commercial errors use this JSON shape:

```json
{
  "code": "lesson_conflict",
  "message": "The requested interval conflicts with a scheduled lesson."
}
```

The `code` value is stable and uses English terms. Clients may display `message` and branch on `code`.

| Status | Code | Meaning |
| --- | --- | --- |
| `401` | `unauthenticated` | The matching session is missing, invalid, or expired. |
| `403` | `unauthorized` | The account cannot access the resource. |
| `403` | `missing_intent` | The mutation lacks `X-Requested-With: fetch`. |
| `400` | `invalid_request` | The JSON body or field values are invalid. |
| `400` | `invalid_grid` | The start is not aligned to the policy grid. |
| `400` | `duration_override` | The request contains a lesson duration override. |
| `400` | `booking_minimum` | The learner start is less than 24 hours ahead. |
| `400` | `learner_change_cutoff` | The learner submitted a reschedule inside the 24-hour cutoff. |
| `400` | `horizon` | The lesson start is outside the rolling 14-day horizon. |
| `400` | `plan_precedence` | The requested lower-priority plan is not eligible. |
| `400` | `package_expired` | The package is not valid for the requested lesson start. |
| `409` | `package_token_exhausted` | No available package token can fund the lesson. |
| `409` | `package_overlap` | A package cannot overlap an active contract or unresolved obligation. |
| `409` | `contract_overlap` | A contract conflicts with a package, lesson, or contract obligation. |
| `409` | `contract_allowance_exhausted` | The contract allowance is unavailable for this learner action. |
| `400` | `contract_replacement_deadline` | The replacement exceeds 30 teacher-local calendar days. |
| `409` | `lesson_rescheduled` | The lesson already has its allowed learner reschedule. |
| `409` | `lesson_conflict` | The requested interval conflicts with a participant or protected buffer. |
| `409` | `unresolved_obligation` | A contract, token, package-backed lesson, or future lesson blocks the operation. |
| `409` | `incomplete_resolution` | An availability preview has an unresolved near-term conflict. |
| `409` | `stale_preview` | Availability state changed after the preview was created. |
| `400` | `invalid_resolution` | A selected availability resolution is not eligible. |
| `400` | `correction_reason_required` | An administrative correction lacks a non-empty reason. |
| `403` | `correction_not_allowed` | The teacher cannot correct this transition or record. |
| `409` | `event_immutable` | Business history cannot be edited or deleted. |
| `409` | `timezone_locked` | Active obligations prevent an ordinary teacher timezone change. |
| `500` | `internal_error` | The operation failed unexpectedly. |

Unauthenticated and wrong-realm requests do not expose scheduling or commercial data.
Mutation endpoints require the intent header even when the body is empty.
