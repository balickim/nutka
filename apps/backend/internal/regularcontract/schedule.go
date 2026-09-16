// This file reconciles permanent schedules, omissions, notice, early end, and prospective prices.
package regularcontract

import (
	"sort"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

// OriginalLocalMonth returns the teacher-local month that owns an allowance.
func (o Occurrence) OriginalLocalMonth() string {
	if len(o.OriginalLocalDate) >= 7 {
		return o.OriginalLocalDate[:7]
	}
	return ""
}

// ChangeSchedule updates only future occurrences that have not been individually moved.
func (c *RegularContract) ChangeSchedule(request ScheduleChangeRequest) error {
	location, err := c.location()
	if err != nil {
		return err
	}
	policy := c.lifecyclePolicy()
	if err := c.validateScheduleChange(request, policy, location); err != nil {
		return err
	}
	effectiveDate := localDate(request.EffectiveOn, location)
	updated := append([]Occurrence(nil), c.Occurrences...)
	for index := range updated {
		if err := c.changeOccurrenceSchedule(&updated[index], request, policy, location, effectiveDate); err != nil {
			return err
		}
	}
	c.Occurrences = updated
	c.Weekday = request.Weekday
	c.StartMinute = request.StartMinute
	c.addEvent(EventScheduleChanged, "", "", request.Actor, request.Now, "")
	return nil
}

func (c RegularContract) validateScheduleChange(request ScheduleChangeRequest, policy Policy, location *time.Location) error {
	if request.Actor.Role != Teacher || !authorized(c, request.Actor) || request.EffectiveOn.IsZero() || request.Now.IsZero() {
		return ErrUnauthorized
	}
	if !validWeeklyTime(request.Weekday, request.StartMinute, policy.StartGrid) {
		return ErrInvalidContract
	}
	if !localDate(request.EffectiveOn, location).After(localDate(request.Now, location)) {
		return ErrInvalidContract
	}
	return nil
}

func (c RegularContract) changeOccurrenceSchedule(occurrence *Occurrence, request ScheduleChangeRequest, policy Policy, location *time.Location, effectiveDate time.Time) error {
	date, err := parseDateKey(occurrence.OriginalLocalDate)
	if err != nil || occurrence.OriginalLocalDate < dateKey(effectiveDate, location) || occurrence.IndividuallyRescheduled || occurrence.ScheduleState != Scheduled {
		return nil
	}
	for date.Weekday() != request.Weekday {
		date = date.AddDate(0, 0, 1)
	}
	interval, err := resolveOccurrence(date, request.StartMinute, c.TeacherTimezone, policy.LessonDuration)
	if err != nil {
		return err
	}
	if intersectsAny(interval, request.Unavailable) || !containsInterval(request.Availability, interval) {
		return ErrUnavailable
	}
	if scheduling.HasParticipantConflict(interval, c.TeacherID, c.LearnerID, request.Lessons, scheduling.IntervalPolicyFromBusinessPolicy(policy)) {
		return ErrParticipantConflict
	}
	occurrence.Interval = interval
	return nil
}

// RestoreOmitted makes a distant planned occurrence billable when availability is reopened.
func (c *RegularContract) RestoreOmitted(id string, availability []scheduling.Interval, actor Actor, now time.Time) error {
	if actor.Role != Teacher || !authorized(*c, actor) || now.IsZero() {
		return ErrUnauthorized
	}
	occurrence, err := c.occurrence(id)
	if err != nil {
		return err
	}
	policy := c.lifecyclePolicy()
	if occurrence.ScheduleState != Omitted || !occurrence.Interval.Start.After(now.UTC().Add(policy.BookingHorizon)) || !containsInterval(availability, occurrence.Interval) {
		return ErrUnavailable
	}
	occurrence.ScheduleState = Scheduled
	occurrence.OmissionReason = ""
	occurrence.BillingOutcome = BillableOrdinary
	c.addEvent("contract_occurrence_restored", id, occurrence.OriginalLocalMonth(), actor, now, "")
	return nil
}

// SubmitNotice computes the ordinary end at the final day of the next local month.
func (c *RegularContract) SubmitNotice(actor Actor, now time.Time) error {
	if !validNotice(*c, actor, now) {
		return ErrUnauthorized
	}
	if c.Status == NoticeGiven {
		return nil
	}
	location, err := c.location()
	if err != nil {
		return err
	}
	c.NoticeAt = now.UTC()
	c.EffectiveEndOn = lastOfMonth(nextMonth(now, location), location)
	if dateKey(c.EffectiveEndOn, location) > dateKey(c.EndOn, location) {
		c.EffectiveEndOn = c.EndOn
	}
	for index := range c.Occurrences {
		occurrence := &c.Occurrences[index]
		if occurrenceAfterLocalDate(occurrence.OriginalLocalDate, c.EffectiveEndOn, location) && occurrence.ScheduleState == Scheduled {
			occurrence.ScheduleState = Cancelled
			occurrence.BillingOutcome = ContractTermination
		}
	}
	c.Status = NoticeGiven
	c.addEvent(EventNoticeSubmitted, "", "", actor, now, "")
	return nil
}

func validNotice(contract RegularContract, actor Actor, now time.Time) bool {
	return authorized(contract, actor) && (actor.Role == Teacher || actor.Role == Learner) && contract.Status != Ended && !now.IsZero()
}

// EndEarly records a mutually agreed end and removes later scheduled occurrences.
func (c *RegularContract) EndEarly(endOn time.Time, reason string, actor Actor, now time.Time) error {
	if !validEarlyEndActor(*c, actor, now, reason) || c.Status == Ended {
		return ErrInvalidEarlyEnd
	}
	location, err := c.location()
	if err != nil {
		return err
	}
	endDate := localDate(endOn, location)
	if !c.validEarlyEndDate(endDate, location) {
		return ErrInvalidEarlyEnd
	}
	c.EffectiveEndOn = endDate
	c.Status = Ended
	for index := range c.Occurrences {
		occurrence := &c.Occurrences[index]
		if occurrenceAfterLocalDate(occurrence.OriginalLocalDate, endDate, location) && occurrence.ScheduleState == Scheduled {
			occurrence.ScheduleState = Cancelled
			occurrence.BillingOutcome = ContractTermination
		}
	}
	c.addEvent(EventEarlyEnded, "", "", actor, now, reason)
	return nil
}

func validEarlyEndActor(contract RegularContract, actor Actor, now time.Time, reason string) bool {
	return actor.Role == Teacher && authorized(contract, actor) && !now.IsZero() && strings.TrimSpace(reason) != ""
}

func (c RegularContract) validEarlyEndDate(endDate time.Time, location *time.Location) bool {
	key := dateKey(endDate, location)
	return key >= dateKey(c.StartOn, location) && key <= dateKey(c.EndOn, location) && key <= dateKey(c.EffectiveEndOn, location)
}

// AmendPrice records a future-month price without rewriting prior occurrence snapshots.
func (c *RegularContract) AmendPrice(request PriceAmendmentRequest) error {
	if !c.validPriceAmendmentActor(request) {
		return ErrInvalidAmendment
	}
	location, err := c.location()
	if err != nil {
		return err
	}
	effective, parseErr := time.ParseInLocation("2006-01", request.EffectiveMonth, location)
	if parseErr != nil || !firstOfMonth(effective, location).After(firstOfMonth(request.Now, location)) || c.hasAmendment(request.EffectiveMonth) {
		return ErrInvalidAmendment
	}
	c.Amendments = append(c.Amendments, Amendment{EffectiveMonth: request.EffectiveMonth, PriceMinor: request.PriceMinor, Currency: c.Currency, CreatedAt: request.Now.UTC(), Actor: request.Actor})
	sort.Slice(c.Amendments, func(left, right int) bool {
		return c.Amendments[left].EffectiveMonth < c.Amendments[right].EffectiveMonth
	})
	for index := range c.Occurrences {
		occurrence := &c.Occurrences[index]
		if occurrence.OriginalLocalMonth() >= request.EffectiveMonth && occurrence.ScheduleState != Cancelled {
			occurrence.UnitPriceMinor, occurrence.Currency = c.priceForMonth(occurrence.OriginalLocalMonth())
		}
	}
	c.addEvent(EventPriceAmended, "", request.EffectiveMonth, request.Actor, request.Now, "")
	return nil
}

func (c RegularContract) validPriceAmendmentActor(request PriceAmendmentRequest) bool {
	return request.Actor.Role == Teacher && authorized(c, request.Actor) && !request.Now.IsZero() && request.PriceMinor > 0 && request.Currency == c.Currency && request.Currency == c.lifecyclePolicy().Currency
}

func (c RegularContract) hasAmendment(month string) bool {
	for _, amendment := range c.Amendments {
		if amendment.EffectiveMonth == month {
			return true
		}
	}
	return false
}

func (c RegularContract) priceForMonth(month string) (int64, string) {
	price, currency := c.PriceMinor, c.Currency
	for _, amendment := range c.Amendments {
		if amendment.EffectiveMonth <= month {
			price, currency = amendment.PriceMinor, amendment.Currency
		}
	}
	return price, currency
}
