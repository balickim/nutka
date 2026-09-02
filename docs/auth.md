# Nutka authentication contract

This document is the canonical contract for learner authentication. It covers the
first milestone only: the Nutka learner realm backed by PocketBase, with a
server-managed cookie session and an in-memory frontend auth state.

## Scope

- The auth realm is the closed `learners` PocketBase auth collection.
- Learner accounts are invite-only. Until invitation delivery exists, the
  development seed command is the only supported provisioning path.
- Learner auth is separate from any Sailormoon realm or PocketBase `users` and
  `owner_users` records. No records are imported between realms.
- Only verified learners may create a session.

## HTTP API

The browser uses relative `/api` URLs. Requests that participate in the auth
flow include `credentials: "include"` and the intent header
`X-Requested-With: fetch`.

| Method | Path | Contract |
| --- | --- | --- |
| `POST` | `/api/collections/learners/auth-with-password` | Accepts the learner identity and password. Requires `X-Requested-With: fetch`. A successful, verified authentication sets the session cookie and returns the learner record without an auth token. Invalid or unverified credentials do not issue a session cookie. |
| `GET` | `/api/auth/me` | Reads the Nutka session and returns `{ "record": <learner>, "session_expires_at": <timestamp>? }`. Missing, expired, or wrong-realm sessions return `401` and clear the session cookie. |
| `POST` | `/api/auth/logout` | Clears the Nutka session cookie and returns `{ "status": "ok" }`. Logout is idempotent and requires the auth intent header. |

Native PocketBase auth refresh is not part of this contract and is rejected.
The server strips auth tokens from authentication responses; the browser never
needs to handle or persist a bearer token.

PocketBase superusers retain native admin-panel access to the `learners`
collection. Deferred learner self-service routes remain hidden from guests and
learner sessions; the session bridge runs before PocketBase auth resolution,
and the deferred-route guard runs after it so only a resolved superuser can
reach those native collection routes.

## Session cookie

Successful learner login sets exactly one Nutka session cookie:

| Attribute | Required value |
| --- | --- |
| Name | `__Host-nutka_session` |
| `HttpOnly` | enabled |
| `Secure` | enabled |
| `SameSite` | `Lax` |
| `Path` | `/` |
| `Domain` | omitted |
| Lifetime | 12 hours |

The `__Host-` prefix requires `Secure`, `Path=/`, and no `Domain`. The token
value is available only to the browser's cookie jar, not to JavaScript,
`localStorage`, or application logs. In local development the app's Vite
server proxies `/api` requests to the dynamic backend target, so the browser
sees the session as same-origin.

## Frontend lifecycle

1. A protected route performs a single-flight `/api/auth/me` bootstrap. A valid
   response populates in-memory auth state; no token or auth record is persisted
   to `localStorage`.
2. A missing or expired session clears local state and sends the user to
   `/login`. A network or 5xx bootstrap failure is retryable rather than being
   treated as a confirmed unauthenticated session.
3. After login, the app navigates to a validated `?redirect=` path or `/`.
   Redirects accept only same-origin paths beginning with exactly one `/`; full
   URLs and protocol-relative paths are rejected.
4. Logout is best effort against the server but always clears local in-memory
   state. Login/logout coordination between tabs may use `BroadcastChannel`,
   but messages contain no token or learner data.
5. The app signs out at the server-reported expiry. If login does not provide an
   expiry, it falls back to 12 hours from successful login.

The auth intent header is required on mutation endpoints so browser auth actions
cannot be mistaken for ordinary cross-site requests. Invalid credentials use a
generic error and do not reveal whether an account exists.

## Development seed

Run this command from `apps/backend` while using a development environment:

```sh
NUTKA_ENV=development go run . seed-learner \
  --email learner@example.test \
  --password 'local-password' \
  --name 'Test Learner'
```

The command creates or updates a verified local learner, never logs the supplied
password, and is not registered outside development. It does not add a public
registration mechanism.

## Deferred and out of scope

This milestone deliberately does not include:

- public learner registration;
- invitation delivery or invitation management UI;
- password reset or account recovery;
- MFA;
- learner profile editing;
- refresh-token rotation;
- multi-language auth UI or localization infrastructure;
- importing users from another auth realm.

Adding any of these requires an update to this contract before implementation.
