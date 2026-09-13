## Purpose

Defines consistent frontend server-state ownership, persona-safe caching, request deduplication, and mutation invalidation without changing backend contracts.

## ADDED Requirements

### Requirement: One server-state authority
The frontend SHALL execute every authentication and scheduling read through the shared query cache. The frontend SHALL execute every authentication and scheduling write through a managed mutation.

#### Scenario: Two consumers request the same data
- **WHEN** two mounted consumers request the same query key concurrently
- **THEN** the frontend performs one network request and shares the result

#### Scenario: A view needs server data
- **WHEN** a route or component loads authentication, calendar, or slot data
- **THEN** it uses query state instead of a separate effect-driven request state

### Requirement: Central query-key registry
The frontend SHALL define every query key and cache rule in one authoritative registry. Each key SHALL include every variable that changes the returned data.

#### Scenario: Persona queries coexist
- **WHEN** teacher and learner sessions are active in one browser
- **THEN** their session and calendar queries use distinct keys

#### Scenario: Assignment slots coexist
- **WHEN** a learner loads slots for multiple assignments
- **THEN** each assignment and learner combination uses a distinct key

#### Scenario: A developer adds a query
- **WHEN** frontend code defines a new query
- **THEN** it obtains the key from the central registry without an inline query-key array

### Requirement: Account-safe cache isolation
Each account-owned calendar and slot key SHALL include the persona role and authenticated account identifier. A role change SHALL not expose cached data from the previous account.

#### Scenario: An account replaces another account in one role
- **WHEN** a browser authenticates a different account for an existing persona role
- **THEN** the frontend removes the previous account cache before exposing the new session

#### Scenario: One persona logs out
- **WHEN** the teacher or learner logs out
- **THEN** the frontend cancels and removes only that persona cache
- **AND** the other persona remains authenticated

### Requirement: Session query lifecycle
Route guards and rendered components SHALL consume the same persona session query. Session expiry, `401`, logout, and cross-tab logout SHALL clear the matching persona cache.

#### Scenario: A route and component bootstrap together
- **WHEN** a protected route guard and its component request the same session
- **THEN** they share one in-flight session request

#### Scenario: A session expires
- **WHEN** the server expiry passes or the session endpoint returns `401`
- **THEN** the frontend clears the matching persona data and redirects to that persona login

#### Scenario: Another tab logs out
- **WHEN** a tab receives a matching persona logout event
- **THEN** it removes that persona cache without receiving account or token data

### Requirement: Scheduling cache rules
Successful mutations SHALL apply the central cache rules before the affected view settles. The rules SHALL preserve unaffected persona data.

#### Scenario: Availability changes
- **WHEN** a teacher creates, updates, or deletes an availability rule or exception
- **THEN** the frontend invalidates teacher calendars, learner calendars, and learner slot queries

#### Scenario: An assignment changes
- **WHEN** a teacher changes assignment activity or duration
- **THEN** the frontend invalidates teacher and learner calendars and the changed assignment slot query

#### Scenario: A lesson changes
- **WHEN** a lesson is booked, rescheduled, or cancelled
- **THEN** the frontend invalidates teacher calendars, learner calendars, and all learner slot queries

### Requirement: Stable query presentation
The frontend SHALL use query state for initial loading, retry, error, and background refresh presentation. A background refresh SHALL retain the last successful data.

#### Scenario: Initial query is pending
- **WHEN** no cached result exists and a query is pending
- **THEN** the view renders its existing loading state

#### Scenario: Background refresh is pending
- **WHEN** cached data exists and the frontend refreshes it
- **THEN** the view retains the cached data until the refresh settles

#### Scenario: A scheduling query fails
- **WHEN** a scheduling query returns a stable API error code
- **THEN** the view renders the existing Polish message for that code

### Requirement: Shared request boundary
Only one frontend transport boundary SHALL call the Fetch API. It SHALL send cookies on every request and the intent header on every mutation.

#### Scenario: A query is cancelled
- **WHEN** the query cache aborts an obsolete read
- **THEN** the transport receives and applies the abort signal

#### Scenario: A mutation sends a request
- **WHEN** a login, logout, or scheduling mutation runs
- **THEN** the request includes credentials and `X-Requested-With: fetch`

### Requirement: Controlled retry behavior
The frontend SHALL not retry mutations or deterministic `4xx` responses. Read queries MAY retry transient network and `5xx` failures once.

#### Scenario: Booking conflicts
- **WHEN** booking returns a conflict response
- **THEN** the frontend reports the conflict without repeating the mutation

#### Scenario: A read has a transient failure
- **WHEN** a read fails because of the network or a `5xx` response
- **THEN** the frontend performs at most one automatic retry
