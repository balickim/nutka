# Nutka documentation

This directory contains the contracts for Nutka authentication and scheduling.

## Contract order

- Read the relevant constitution before changing behavior.
- Treat each page in `docs/api/` as the canonical contract for its endpoints.
- Link to the canonical API page instead of copying endpoint definitions.
- Record rationale in `docs/adr/` only when a rule needs a decision record.
- Keep concrete datetimes in UTC unless a constitution defines a local wall-clock field.

## Contracts

- [Authentication contract](auth.md) defines the teacher and learner realms.
- [Scheduling constitution](constitutions/scheduling.md) defines scheduling rules.
- [Assignments API](api/assignments.md) defines assignment reads and teacher updates.
- [Availability API](api/availability.md) defines recurring rules and exceptions.
- [Booking API](api/booking.md) defines learner slots and booking.
- [Lifecycle API](api/lifecycle.md) defines lesson changes and counters.
- [Calendar API](api/calendar.md) defines teacher and learner calendar reads.
- [Scheduling errors](api/errors.md) defines shared status and error codes.

## Change workflow

- Update the constitution when a shared scheduling rule changes.
- Update one canonical API page when an endpoint contract changes.
- Update consumers after the contract change.
- Validate links and OpenSpec artifacts before review.
