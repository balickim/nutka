# Nutka documentation

This directory contains the contracts for Nutka authentication, scheduling, commercial lesson plans, learner materials, lesson notes, repertoire, and practice.

## Contract order

- Read the relevant constitution before changing behavior.
- Treat each page in `docs/api/` as the canonical contract for its endpoints.
- Link to the canonical API page instead of copying endpoint definitions.
- Record rationale in `docs/adr/` only when a rule needs a decision record.
- Keep concrete datetimes in UTC unless a constitution defines a local wall-clock field.

## Contracts

- [Authentication contract](auth.md) defines the teacher and learner realms.
- [Scheduling constitution](constitutions/scheduling.md) defines scheduling rules.
- [Frontend query state constitution](constitutions/frontend-query-state.md) defines frontend server-state ownership.
- [Frontend view state constitution](constitutions/frontend-view-states.md) defines data states, write feedback, dialogs, teacher action tiers, and learner navigation.
- [Learner content constitution](constitutions/learner-content.md) defines scope, storage, and routes of assignment-scoped learner content.
- [Commercial policy decision](adr/0001-commercial-policy-and-history.md) records backend policy authority and immutable history.
- [Assignments API](api/assignments.md) defines assignment reads and teacher updates.
- [Availability API](api/availability.md) defines recurring rules and exceptions.
- [Booking API](api/booking.md) defines learner slots and booking.
- [Lifecycle API](api/lifecycle.md) defines lesson changes, outcomes, and corrections.
- [Calendar API](api/calendar.md) defines teacher and learner calendar reads.
- [Business policy API](api/business-policy.md) defines authenticated policy reads.
- [Packages API](api/packages.md) defines package purchases and token transitions.
- [Regular contracts API](api/contracts.md) defines weekly contracts and occurrence series.
- [Payments API](api/payments.md) defines settlement, charges, credits, and refunds.
- [Business history API](api/history.md) defines immutable role-scoped events.
- [Teacher unresolved work API](api/unresolved-work.md) defines actionable teacher work.
- [Learner materials API](api/materials.md) defines teacher-authored text, image, and PDF materials.
- [Lesson notes API](api/lesson-notes.md) defines one teacher note for each lesson.
- [Repertoire API](api/repertoire.md) defines the pieces of a learner, learner wishes, and arrangement versions.
- [Practice API](api/practice.md) defines practice tasks, learner practice sessions, and practice summaries.
- [Scheduling errors](api/errors.md) defines shared status and error codes.

## Change workflow

- Update the constitution when a shared scheduling rule changes.
- Update one canonical API page when an endpoint contract changes.
- Update consumers after the contract change.
- Validate links and OpenSpec artifacts before review.
