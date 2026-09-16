// This file applies contract lifecycle commands and derives allowance effects from immutable events.
package regularcontract

import (
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

// Reschedule moves one occurrence while preserving its identity and billing value.
func (c *RegularContract) Reschedule(request RescheduleRequest) error {
	if !authorized(*c, request.Actor) || request.AssignmentID != c.AssignmentID {
		return ErrUnauthorized
	}
	occurrence, err := c.occurrence(request.OccurrenceID)
	if err != nil {
		return err
	}
	now := request.Now.UTC()
	if !changeableOccurrence(*occurrence, now) {
		return ErrStartedOccurrence
	}
	policy := c.lifecyclePolicy()
	if err := c.validateLearnerReschedule(request, *occurrence, now, policy); err != nil {
		return err
	}
	location, locationErr := c.location()
	if locationErr != nil {
		return locationErr
	}
	if err := validateReplacementDeadline(request.ReplacementStart, *occurrence, location, policy.ContractReplacementDays); err != nil {
		return err
	}
	priorInterval := occurrence.Interval
	replacement, replacementErr := availableReplacement(request, policy.LessonDuration, c.TeacherID, c.LearnerID, scheduling.IntervalPolicyFromBusinessPolicy(policy))
	if replacementErr != nil {
		return replacementErr
	}
	occurrence.Interval = replacement
	occurrence.IndividuallyRescheduled = true
	eventType := EventTeacherRescheduled
	if request.Actor.Role == Learner {
		eventType = EventLearnerRescheduled
	}
	c.addEvent(eventType, occurrence.ID, occurrence.OriginalLocalMonth(), request.Actor, now, "")
	c.Events[len(c.Events)-1].PriorInterval = priorInterval
	c.Events[len(c.Events)-1].NewInterval = replacement
	return nil
}

func changeableOccurrence(occurrence Occurrence, now time.Time) bool {
	return !now.IsZero() && occurrence.ScheduleState == Scheduled && occurrence.Interval.Start.After(now)
}

func (c RegularContract) validateLearnerReschedule(request RescheduleRequest, occurrence Occurrence, now time.Time, policy Policy) error {
	if request.Actor.Role != Learner {
		return nil
	}
	if now.Add(policy.LearnerChangeCutoff).After(occurrence.Interval.Start) || request.ReplacementStart.UTC().Before(now.Add(policy.LearnerChangeCutoff)) {
		return ErrCutoff
	}
	if c.occurrenceWasLearnerRescheduled(occurrence.ID) {
		return ErrAlreadyRescheduled
	}
	if c.Allowances(occurrence.OriginalLocalMonth()).MonthlyReschedulesRemaining <= 0 {
		return ErrAllowanceExhausted
	}
	if request.ReplacementStart.UTC().After(now.Add(policy.BookingHorizon)) {
		return ErrReplacementHorizon
	}
	return nil
}

func validateReplacementDeadline(replacement time.Time, occurrence Occurrence, location *time.Location, days int) error {
	originalDate, err := parseDateKey(occurrence.OriginalLocalDate)
	if err != nil {
		return err
	}
	deadline := time.Date(originalDate.Year(), originalDate.Month(), originalDate.Day(), 0, 0, 0, 0, location).AddDate(0, 0, days)
	if localDate(replacement, location).After(deadline) {
		return ErrReplacementDeadline
	}
	return nil
}

// Cancel releases an occurrence and records whether its value remains billable.
func (c *RegularContract) Cancel(request CancellationRequest) error {
	if !authorized(*c, request.Actor) {
		return ErrUnauthorized
	}
	occurrence, err := c.occurrence(request.OccurrenceID)
	if err != nil {
		return err
	}
	now := request.Now.UTC()
	if now.IsZero() || occurrence.ScheduleState != Scheduled || !occurrence.Interval.Start.After(now) {
		return ErrStartedOccurrence
	}
	priorInterval := occurrence.Interval
	occurrence.ScheduleState = Cancelled
	occurrence.BillingOutcome = TeacherCancel
	eventType := EventTeacherCancel
	if request.Actor.Role == Learner {
		timely := !now.Add(c.lifecyclePolicy().LearnerChangeCutoff).After(occurrence.Interval.Start)
		if timely && c.Allowances(occurrence.OriginalLocalMonth()).FreeCancellationsRemaining > 0 {
			occurrence.BillingOutcome = FreeLearnerCancel
			eventType = EventLearnerFreeCancel
		} else if timely {
			occurrence.BillingOutcome = BillableExhausted
			eventType = EventLearnerBillableCancel
		} else {
			occurrence.BillingOutcome = BillableLateCancel
			eventType = EventLearnerLateCancel
		}
	}
	c.addEvent(eventType, occurrence.ID, occurrence.OriginalLocalMonth(), request.Actor, now, "")
	c.Events[len(c.Events)-1].PriorInterval = priorInterval
	return nil
}

// RecordOutcome closes an ended occurrence without altering its scheduled interval.
func (c *RegularContract) RecordOutcome(id, outcome string, actor Actor, now time.Time) error {
	if actor.Role != Teacher || !authorized(*c, actor) {
		return ErrUnauthorized
	}
	occurrence, err := c.occurrence(id)
	if err != nil {
		return err
	}
	if occurrence.ScheduleState != Scheduled || occurrence.Outcome != "" || now.UTC().Before(occurrence.Interval.End) || (outcome != "completed" && outcome != "learner_no_show") {
		return ErrInvalidContract
	}
	occurrence.Outcome = outcome
	if outcome == "learner_no_show" {
		occurrence.BillingOutcome = BillableNoShow
		c.addEvent(EventOutcomeNoShow, id, occurrence.OriginalLocalMonth(), actor, now, "")
	} else {
		c.addEvent(EventOutcomeCompleted, id, occurrence.OriginalLocalMonth(), actor, now, "")
	}
	return nil
}

// CorrectEvent appends a compensating event and restores the directly affected occurrence effect.
func (c *RegularContract) CorrectEvent(eventID, reason string, actor Actor, now time.Time) error {
	if actor.Role != Teacher || !authorized(*c, actor) || now.IsZero() || strings.TrimSpace(reason) == "" {
		return ErrUnauthorized
	}
	var corrected *Event
	for index := range c.Events {
		if c.Events[index].ID == eventID {
			corrected = &c.Events[index]
			break
		}
	}
	if corrected == nil {
		return ErrInvalidContract
	}
	if !c.uncompensatedEvents()[eventID] {
		return ErrInvalidContract
	}
	c.reverseEventEffect(*corrected)
	c.addEvent("correction", corrected.OccurrenceID, corrected.OriginalMonth, actor, now, reason)
	c.Events[len(c.Events)-1].CorrectsEvent = eventID
	return nil
}
