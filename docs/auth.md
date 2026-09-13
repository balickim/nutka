# Nutka authentication contract

This document defines the closed teacher and learner authentication realms.
PocketBase stores both realms and the server manages browser sessions.

## Scope

- The `teachers` and `learners` auth collections are separate realms.
- Existing custom `users` records migrate to `teachers` with identifiers and credentials preserved.
- Verified accounts and development seed commands are the only session sources.
- Public registration, invitations, password reset, MFA, and account import are unavailable.

## HTTP API

The browser uses relative `/api` URLs with `credentials: "include"`.
Browser auth mutations send `X-Requested-With: fetch`.

| Method | Path | Contract |
| --- | --- | --- |
| `POST` | `/api/collections/teachers/auth-with-password` | Authenticates a verified teacher and sets the teacher session cookie. |
| `POST` | `/api/collections/learners/auth-with-password` | Authenticates a verified learner and sets the learner session cookie. |
| `GET` | `/api/teachers/auth/me` | Reads only the teacher session and returns `{ "record": <teacher>, "session_expires_at": <timestamp>? }`. |
| `GET` | `/api/learners/auth/me` | Reads only the learner session and returns `{ "record": <learner>, "session_expires_at": <timestamp>? }`. |
| `POST` | `/api/teachers/auth/logout` | Clears only the teacher session. The request requires the intent header. |
| `POST` | `/api/learners/auth/logout` | Clears only the learner session. The request requires the intent header. |

Missing, expired, or wrong-realm sessions return `401` and clear only the matching cookie.
Logout is idempotent. Legacy `/api/auth/*` routes do not exist.

Native PocketBase auth refresh is disabled for both realms.
Native record and self-service routes remain unavailable to guests and persona sessions.
Only PocketBase superusers may use those native routes.

Authentication responses contain no bearer token.
The server derives scheduling identity from the matching session cookie.
Request identity fields cannot replace the resolved persona.

## Persona route spaces

- Teacher application routes use `/teachers/*`.
- Learner application routes use `/learners/*`.
- A teacher session cannot authorize learner routes.
- A learner session cannot authorize teacher routes.
- Each realm resolves only its own cookie.

## Session cookies

Successful login sets one cookie for the matching realm.

| Realm | Name |
| --- | --- |
| Teacher | `__Host-nutka_teacher_session` |
| Learner | `__Host-nutka_learner_session` |

Each session cookie has these attributes:

- `HttpOnly` is enabled.
- `Secure` is enabled.
- `SameSite` is `Lax`.
- `Path` is `/`.
- `Domain` is omitted.
- Lifetime is 12 hours.

The `__Host-` prefix requires `Secure`, `Path=/`, and no `Domain`.
Tokens are unavailable to JavaScript, `localStorage`, and application logs.

## Development seeds

Run these commands from `apps/backend` in a development environment:

```sh
NUTKA_ENV=development go run . seed-teacher \
  --email teacher@example.test \
  --password 'local-password' \
  --name 'Test Teacher'

NUTKA_ENV=development go run . seed-learner \
  --email learner@example.test \
  --password 'local-password' \
  --name 'Test Learner'
```

Each command creates or updates one verified local account.
Each command rejects missing fields and never logs the supplied password.
Neither command creates a public registration mechanism.

## Frontend lifecycle

Each protected route bootstraps its own `/api/{realm}/auth/me` request.
The frontend stores each record in memory only.

A `401` clears the matching realm state and redirects to that realm login.
Network and `5xx` bootstrap failures remain retryable.

After login, redirects accept only same-origin paths beginning with exactly one `/`.
Full URLs and protocol-relative paths are rejected.

Logout clears matching local state even when the server request fails.
Cross-tab messages contain event types only.

The frontend signs out at the server-reported expiry.
Without an expiry, it uses 12 hours from successful login.
