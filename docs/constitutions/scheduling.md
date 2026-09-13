# Scheduling constitution

This constitution defines scheduling data, time rules, authorization, and API boundaries.

## Identity and access

- The system keeps teacher and learner identities in separate authentication realms.
- Scheduling handlers resolve identity from the matching HttpOnly session cookie.
- Request identity fields cannot replace the resolved account.
- Teacher resources require a verified teacher session.
- Learner resources require a verified learner session.
- A teacher session cannot access learner resources.
- A learner session cannot access teacher resources.
- A teacher can read and change only that teacher's assignments and availability.
- A learner can read only assignments that include that learner.
- Only the assigned teacher or learner can mutate a lesson.
- See the [authentication contract](../auth.md) for session and route rules.

## Assignments

- Assignments support many teachers and many learners.
- Each teacher and learner pair has one assignment record.
- New assignments are active and use a 45-minute default duration.
- Assignment durations are positive multiples of 15 minutes.
- Only teachers can change assignment activity or default duration.
- Deactivating an assignment prevents new learner bookings.
- Deactivating an assignment retains existing lessons and events.
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
- The lesson interval must fit inside effective availability.
- Candidate starts use the 15-minute UTC grid.
- A learner can query slots only through an active assignment.
- Learner slots use a rolling 14-day horizon.
- Slot starts must be after the current instant.
- Slot ends must not exceed the horizon end.
- Each active lesson protects five minutes before and after its lesson interval.
- Protected intervals conflict for either participant.
- Availability blocks conflict with lesson intervals, not their protected buffers.
- See the [availability API](../api/availability.md) and [booking API](../api/booking.md).

## Lessons and history

- Learner booking always uses the active assignment default duration.
- Learner booking accepts no duration override.
- Only teachers can change a lesson duration during rescheduling.
- Learners retain the current duration during rescheduling.
- Duration changes remain positive multiples of 15 minutes.
- Rescheduling and booking revalidate availability, horizon, duration, and participant conflicts atomically.
- Started and past lessons cannot be rescheduled or cancelled.
- Cancellation retains the lesson with `cancelled` status.
- Cancellation records the initiating role, account identifier, and UTC instant.
- Cancelled lessons do not block future participant conflicts.
- Each successful reschedule appends prior and new UTC interval snapshots.
- Rescheduling does not increment cancellation counters.
- Each cancellation counter counts cancellations initiated by its account.
- See the [lifecycle API](../api/lifecycle.md) for the canonical contract.

## Time persistence

- Concrete lesson and exception datetimes persist as UTC instants.
- Concrete scheduling responses use RFC3339 UTC values with a `Z` offset.
- Recurring rules preserve local wall-clock values and the teacher IANA timezone.
- The default recurring timezone is `Europe/Warsaw`.
- The resolver chooses the earliest UTC instant for an ambiguous wall-clock value.
- The resolver shifts a nonexistent wall-clock value forward by the DST gap.
- The resolver applies one policy to every recurring expansion.
- Clients localize concrete UTC values only at display boundaries.
- Clients display recurring rules in the teacher's configured IANA timezone.
- See the [availability API](../api/availability.md), [booking API](../api/booking.md), and [calendar API](../api/calendar.md).

## API contract

- Browser mutations send `X-Requested-With: fetch`.
- Scheduling responses expose DTO fields and not PocketBase collection records.
- Scheduling errors use the shared JSON contract.
- See [scheduling errors](../api/errors.md) for status codes and stable error codes.
