# Scheduling errors

Scheduling errors use this JSON shape:

```json
{
  "code": "conflict",
  "message": "The requested interval conflicts with a scheduled lesson."
}
```

The `code` value is stable. Clients can display `message` and branch on `code`.

| Status | Code | Meaning |
| --- | --- | --- |
| `401` | `unauthenticated` | The matching session is missing, invalid, or expired. |
| `403` | `unauthorized` | The account cannot access the resource. |
| `403` | `missing_intent` | The mutation lacks `X-Requested-With: fetch`. |
| `400` | `invalid_duration` | The duration is not a positive multiple of 15 minutes. |
| `400` | `invalid_grid` | The start is not aligned to a 15-minute boundary. |
| `400` | `invalid_request` | The JSON body or field values are invalid. |
| `400` | `horizon` | The requested time is outside the rolling horizon. |
| `409` | `conflict` | The requested interval conflicts with a scheduled lesson. |
| `500` | `internal_error` | The scheduling operation failed unexpectedly. |

Unauthenticated and wrong-realm requests do not expose scheduling data.
Mutation endpoints require the intent header even when the body is empty.
