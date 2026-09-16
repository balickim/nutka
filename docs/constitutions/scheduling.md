# Scheduling constitution

This constitution defines scheduling data, commercial rules, time rules, authorization, and API boundaries.

## Identity and access

- The system keeps teacher and learner identities in separate authentication realms.
- Scheduling handlers resolve identity from the matching HttpOnly session cookie.
- Request identity fields cannot replace the resolved account.
- Teacher resources require a verified teacher session.
- Learner resources require a verified learner session.
- A teacher session cannot access learner resources.
- A learner session cannot access teacher resources.
- A teacher reads and changes only that teacher's assignments and availability.
- A learner reads only assignments that include that learner.
- Only the assigned teacher or learner mutates a lesson.
- Only the assigned teacher manages packages, contracts, corrections, and assignment activity.
- All commercial records belong to one teacher–learner assignment.
- See the [authentication contract](../auth.md) for session and route rules.

## Backend policy

- The backend owns one global business policy for scheduling and commercial values.
- Authenticated teacher and learner sessions read policy through the business policy endpoint.
- The public landing page does not consume the authenticated policy endpoint.
- The policy uses `PLN` and integer minor-unit prices.
- The policy sets ad hoc price to 8000 minor units.
- The policy sets package price to 26000 minor units.
- The policy sets regular lesson price to 5000 minor units.
- The policy sets package size to four tokens.
- The policy sets package validity to 60 teacher-local calendar days.
- The policy sets teacher cancellation extension to seven teacher-local calendar days.
- The policy sets commercial lesson duration to 45 minutes.
- The policy sets the lesson start grid to 15 minutes.
- The policy sets each participant buffer to five minutes.
- The policy sets learner booking minimum to 24 hours.
- The policy sets learner change cutoff to 24 hours.
- The policy sets flexible booking horizon to 14 days.
- The policy sets one contract reschedule allowance per original month.
- The policy sets two free learner cancellations per contract.
- The policy sets contract replacement deadline to 30 teacher-local calendar days.
- The policy sets monthly payment due day to five.
- The policy sets ordinary contract end to 30 June.
- Each package, contract, amendment, and ad hoc lesson stores an immutable policy snapshot.
- A policy change affects new obligations and does not rewrite stored snapshots.

## Assignments and plans

- Each teacher and learner pair has one assignment record.
- New assignments are active.
- Assignments do not configure lesson duration.
- Every commercial lesson uses `regular_contract`, `package`, or `ad_hoc`.
- The system excludes free trials from assignments and commercial records.
- Plan precedence is `regular_contract`, then `package`, then `ad_hoc`.
- An active contract rejects flexible package and ad hoc booking.
- An available valid package token funds flexible booking before ad hoc fallback.
- A caller cannot force a lower-priority plan.
- Package purchase and contract activation resolve prohibited earlier obligations atomically.
- Deactivation blocks new bookings and retains all history.
- Deactivation requires no active contract, open token, package-backed future lesson, or future scheduled lesson.
- Historical unpaid obligations do not alone block deactivation.
- See the [assignments API](../api/assignments.md) for the canonical contract.

## Availability and slots

- A teacher's weekly plan is unavailable unless an enabled rule opens an interval.
- Weekly rules store weekday and local wall-clock values.
- Weekly rules have no end date.
- Teacher timezones use valid IANA identifiers.
- New and migrated teachers use `Europe/Warsaw` when no timezone exists.
- Available exceptions add concrete UTC intervals.
- Unavailable exceptions subtract concrete UTC intervals.
- Unavailable exceptions take precedence over recurring and available intervals.
- A lesson interval must fit inside effective availability.
- Candidate starts use the policy start grid.
- Flexible learner starts are at least 24 hours and at most 14 days ahead.
- Flexible teacher starts are at most 14 days ahead.
- A teacher may confirm a flexible start inside the learner minimum.
- The horizon constrains lesson start, not lesson end.
- Each active lesson protects five minutes before and after its lesson interval.
- Protected intervals conflict for either participant.
- Contract occurrences reserve participant intervals beyond the flexible booking horizon.
- Availability mutations use preview and resolved commit.
- Near-term lesson conflicts require one teacher resolution each.
- Distant contract omissions and restorations use planned availability effects.
- Planned distant omissions do not consume allowances or extend packages.
- Restored near-term availability does not recreate a cancelled lesson.
- The system does not infer public holidays or statutory days off.
- See the [availability API](../api/availability.md) and [booking API](../api/booking.md).

## Lessons and entitlements

- Every lesson retains its plan, price, policy version, and applicable entitlement link.
- Every commercial lesson uses the fixed 45-minute duration.
- Booking and rescheduling reject duration overrides from either persona.
- Learner flexible booking requires an active assignment.
- The assigned teacher can book an eligible flexible lesson for the learner.
- Booking validates assignment, plan, entitlement, policy, timing, availability, duration, and conflicts atomically.
- Booking creates the lesson, entitlement, obligation, and required events atomically.
- Started and past lessons cannot be rescheduled or cancelled.
- Cancellation releases the interval and retains the lesson record.
- Cancellation records initiator, account, UTC instant, cutoff class, and plan consequence.
- Rescheduling moves one lesson identity and records old and new UTC intervals.
- Rescheduling requires an eligible replacement and does not masquerade as cancellation.
- Learner rescheduling is unavailable inside the 24-hour cutoff.
- Exactly 24 hours before start is timely.
- Contract learner rescheduling uses the original lesson month allowance.
- A contract occurrence accepts at most one learner reschedule.
- Contract replacements must start within 30 teacher-local calendar days of the original start.
- Teacher rescheduling bypasses learner cutoff and learner allowances.
- A scheduled ended lesson derives `awaiting_outcome` until teacher submission.
- Only the assigned teacher records `completed` or `learner_no_show`.
- Outcomes do not rewrite scheduled intervals or actual delivery times.
- Package completion, no-show, and late cancellation settle the reserved token as used.
- Timely package cancellation returns the reserved token without extending validity.
- Teacher package cancellation returns the token and extends validity by seven local days.
- Contract free cancellation consumes an allowance and removes or credits billable value.
- Late contract cancellation, no-show, and exhausted allowances remain billable.
- Ad hoc cancellation and no-show create no debt and settle payment as not applicable.
- See the [lifecycle API](../api/lifecycle.md), [packages API](../api/packages.md), and [contracts API](../api/contracts.md).

## Payments and history

- Ad hoc settlement starts as `pending_settlement`.
- A teacher submission changes ad hoc settlement to `paid` or `intentionally_unpaid`.
- An unpaid ad hoc lesson does not block later booking.
- Contract charges derive from billable occurrence snapshots.
- Contract charges support `pending`, `paid`, and `intentionally_unpaid`.
- The system derives overdue after the fifth teacher-local calendar day.
- Unpaid records do not suspend lessons, actions, or contracts automatically.
- Charge reductions preserve original amounts and create immutable adjustments.
- Paid reductions create a credit or a teacher-recorded refund.
- Package purchase records confirmed payment atomically with package and token creation.
- Every business transition appends an immutable event in the same transaction.
- Administrative corrections require a reason and append compensating events.
- Ordinary APIs cannot edit or delete business events.
- Teachers see full assignment history including teacher-only notes.
- Learners see owned events without teacher-only notes.
- Machine values and event types use English terms.
- See the [payments API](../api/payments.md), [history API](../api/history.md), and [errors](../api/errors.md).

## Time persistence

- Concrete lesson, payment, cancellation, outcome, and event datetimes persist as UTC instants.
- Concrete scheduling responses use RFC3339 UTC values with a `Z` offset.
- Recurring rules preserve local wall-clock values and the teacher IANA timezone.
- Teacher-local dates persist as normalized date strings, not instants.
- Package validity uses the teacher-local purchase date as day one.
- Contract dates and weekly times use the teacher timezone.
- Contract monthly boundaries and due dates use the teacher timezone.
- The resolver chooses the earliest UTC instant for an ambiguous wall-clock value.
- The resolver shifts a nonexistent wall-clock value forward by the DST gap.
- The resolver applies one policy to every recurring expansion.
- Clients localize concrete UTC values only at display boundaries.
- Clients display recurring rules and business dates in the teacher timezone.
- See the [availability API](../api/availability.md), [booking API](../api/booking.md), and [calendar API](../api/calendar.md).

## API contract

- Browser mutations send `X-Requested-With: fetch`.
- Scheduling responses expose DTO fields and not PocketBase collection records.
- Mutation bodies reject unknown fields and caller-supplied identity.
- Compound mutations commit business state and required events atomically.
- Scheduling errors use the shared JSON contract.
- See [scheduling errors](../api/errors.md) for status codes and stable error codes.
