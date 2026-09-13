## MODIFIED Requirements

### Requirement: Learner books only an assigned teacher

A learner SHALL create a lesson only for an active teacher–learner assignment. The assigned teacher SHALL also create an eligible flexible lesson on the learner's behalf. The system SHALL reject requests for unassigned teachers without revealing whether the teacher has availability.

#### Scenario: Assigned learner books slot
- **WHEN** an authenticated learner submits an eligible slot for an active assignment
- **THEN** the system creates one scheduled lesson for that learner and teacher

#### Scenario: Assigned teacher books for learner
- **WHEN** the assigned teacher submits an eligible flexible slot for the learner
- **THEN** the system creates the same plan-aware lesson that an eligible learner booking would create

#### Scenario: Unassigned learner books slot
- **WHEN** an authenticated learner submits a slot for a teacher without an active assignment
- **THEN** the system rejects the request and creates no lesson

### Requirement: Booking is atomic across both calendars

The system SHALL validate assignment, plan precedence, entitlement, policy snapshot, horizon, minimum notice, availability, fixed duration, and teacher and learner conflicts within one atomic operation. It SHALL atomically create the lesson, reserve any token, create any obligation, and append required events. A failed validation SHALL create no partial state.

#### Scenario: Concurrent requests target same slot
- **WHEN** two eligible booking requests race for the same teacher and interval
- **THEN** at most one request succeeds and every failed request receives a conflict response without partial records

#### Scenario: Concurrent requests target final token
- **WHEN** two eligible booking requests race for the last package token at different free times
- **THEN** at most one request succeeds and the token balance remains valid

#### Scenario: Learner conflict differs from teacher conflict
- **WHEN** a slot is free for the teacher but overlaps another scheduled lesson for the learner or either buffer
- **THEN** the system rejects the booking and identifies the calendar conflict without creating a lesson

### Requirement: Booking response exposes UTC data

Successful booking responses SHALL identify the lesson, participants, assignment, plan type, applicable package or contract, status, fixed duration, price snapshot, policy snapshot, UTC start and end instants, and protected buffer interval. Responses SHALL not expose session tokens or unauthorized commercial data.

#### Scenario: UI receives booked lesson
- **WHEN** booking succeeds
- **THEN** the API returns English machine values and RFC3339 UTC instants that the client can localize for display

## ADDED Requirements

### Requirement: Every booking uses the fixed policy duration

Every ad hoc, package, and contract lesson SHALL use the current policy duration of 45 minutes when created. Booking and rescheduling APIs SHALL reject a duration override from either persona. Existing commercial lessons SHALL retain their 45-minute snapshot.

#### Scenario: Learner omits duration
- **WHEN** an eligible learner submits only a start instant
- **THEN** the system creates a 45-minute lesson

#### Scenario: Teacher submits duration override
- **WHEN** the teacher attempts to create or reschedule a commercial lesson with a different duration
- **THEN** the system rejects the override and leaves the lesson unchanged

### Requirement: Flexible booking selects the authoritative plan

The booking operation SHALL classify a flexible request using `regular_contract`, `package`, then `ad_hoc` precedence. It SHALL reject flexible booking for an active contract, reserve a token when one is available, and create ad hoc only when no higher plan applies. The caller SHALL NOT select a lower-priority plan.

#### Scenario: Caller requests ad hoc despite token
- **WHEN** an eligible package token exists and the request attempts to force ad hoc
- **THEN** the system rejects the override or ignores it and creates only a package-backed lesson

#### Scenario: No contract or token exists
- **WHEN** an active assignment has no active contract and no available valid token
- **THEN** the system creates an ad hoc booking with its current price snapshot

### Requirement: Flexible starts use a bounded rolling window

A learner flexible booking start SHALL be at or after exactly 24 hours from the current instant and at or before exactly 14 days from the current instant. The limits SHALL apply to lesson start, not lesson end. Both values SHALL come from backend policy.

The teacher SHALL use the same 14-day upper limit for ad hoc and package booking. The teacher MAY create such a booking inside the 24-hour learner minimum only after explicit warning confirmation.

#### Scenario: Learner chooses exactly 24 hours ahead
- **WHEN** all other rules pass and the requested start equals the current instant plus 24 hours
- **THEN** the booking is eligible

#### Scenario: Learner chooses less than 24 hours ahead
- **WHEN** the requested start is earlier than the learner minimum
- **THEN** the system rejects the booking and creates no entitlement or obligation

#### Scenario: Start equals horizon end
- **WHEN** an eligible start equals the current instant plus 14 days
- **THEN** the booking may succeed even though the 45-minute lesson ends after the horizon boundary

#### Scenario: Teacher books short notice
- **WHEN** the teacher confirms an eligible ad hoc or package booking less than 24 hours ahead
- **THEN** the booking may succeed and the history records the teacher action

#### Scenario: Teacher books beyond horizon
- **WHEN** the teacher requests an ad hoc or package start after 14 days
- **THEN** the system rejects the booking

## REMOVED Requirements

### Requirement: Learner booking uses the assignment duration

**Reason**: All three commercial plans have one authoritative 45-minute duration. Assignment-specific and teacher-specific overrides conflict with the pricing contract.

**Migration**: Migrate valid existing assignments and lessons to 45 minutes, remove duration mutation fields and UI controls, and reject all future duration overrides.

