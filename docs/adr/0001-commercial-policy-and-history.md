# ADR 0001: Backend commercial policy and immutable history

## Status

Accepted

## Context

Nutka has one scheduling application and an independent public landing page. Commercial lessons need stable prices, entitlements, timing rules, payment records, and corrections.
Those rules cross booking, availability, lifecycle, calendar, and assignment boundaries. PocketBase records alone cannot provide one authoritative rule set or reconstruct corrections.

## Decision

The authenticated backend owns one global business policy. The policy exposes prices, fixed lesson duration, timing limits, package terms, contract limits, and payment dates.
Each package, contract, amendment, and ad hoc lesson stores the policy values used at creation.

The backend models exactly three commercial plans with strict precedence: `regular_contract`, `package`, then `ad_hoc`.
It stores package tokens, contract occurrences, charges, and financial entries as explicit records.

Every business transition appends an immutable event in the same transaction. Administrative corrections append compensating events with a required reason.
Teacher and learner history projections enforce role visibility. Learner projections omit teacher-only notes.

The landing page remains independent from the authenticated policy endpoint. Release checks compare its displayed prices with backend policy.

## Consequences

- Client controls read policy instead of duplicating commercial constants.
- Existing obligations retain their creation snapshots after policy changes.
- Booking and lifecycle operations must persist state and events atomically.
- Queries derive current balances from uncompensated events and current projections.
- New business endpoint contracts use English machine values and localized React copy.
- Policy changes require a backend deployment in this version.
