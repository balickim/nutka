// This file validates contract activation and expands the fixed weekly term into concrete occurrences.
package regularcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

// Activate validates a fixed term and materializes every weekly occurrence through its local end date.
func Activate(request ActivationRequest) (RegularContract, error) {
	policy := request.Policy
	if policy.Version == "" {
		policy = DefaultPolicy()
	}
	location, err := validateActivation(request, policy)
	if err != nil {
		return RegularContract{}, err
	}
	now := request.Now.UTC()
	startDate := localDate(request.StartOn, location)
	nowDate := localDate(now, location)
	backdated := startDate.Before(nowDate)
	if backdated && strings.TrimSpace(request.BackdateReason) == "" {
		return RegularContract{}, ErrPastStart
	}
	if request.ExistingCommercialObligation {
		return RegularContract{}, ErrCommercialOverlap
	}
	endDate := nextContractEnd(startDate, location, policy)
	contract := RegularContract{ID: request.ID, AssignmentID: request.AssignmentID, TeacherID: request.TeacherID, LearnerID: request.LearnerID,
		TeacherTimezone: request.TeacherTimezone, StartOn: startDate, EndOn: endDate, EffectiveEndOn: endDate,
		Weekday: request.Weekday, StartMinute: request.StartMinute, PriceMinor: policy.RegularLessonPrice.Minor,
		Currency: policy.Currency, Status: Active, Policy: policy, PolicySnapshot: policy.Snapshot()}
	occurrences, err := materializeOccurrences(request, policy, location, startDate, endDate, nowDate, backdated)
	if err != nil {
		return RegularContract{}, err
	}
	contract.Occurrences = occurrences
	contract.addEvent(EventActivated, "", "", request.Actor, now, request.BackdateReason)
	return contract, nil
}

func validateActivation(request ActivationRequest, policy Policy) (*time.Location, error) {
	if !validActivationIdentity(request) || policy.Validate() != nil {
		return nil, ErrInvalidContract
	}
	if request.Actor.Role != Teacher || request.Actor.ID == "" || request.Actor.ID != request.TeacherID {
		return nil, ErrUnauthorized
	}
	if !validWeeklyTime(request.Weekday, request.StartMinute, policy.StartGrid) {
		return nil, ErrInvalidContract
	}
	return scheduling.LoadTimezone(request.TeacherTimezone)
}

func validActivationIdentity(request ActivationRequest) bool {
	return request.ID != "" && request.AssignmentID != "" && request.TeacherID != "" && request.LearnerID != "" && !request.StartOn.IsZero() && !request.Now.IsZero()
}

func validWeeklyTime(weekday time.Weekday, startMinute int, grid time.Duration) bool {
	return weekday >= time.Sunday && weekday <= time.Saturday && startMinute >= 0 && startMinute < 24*60 && startMinute%int(grid/time.Minute) == 0
}

func materializeOccurrences(request ActivationRequest, policy Policy, location *time.Location, startDate, endDate, nowDate time.Time, backdated bool) ([]Occurrence, error) {
	result := make([]Occurrence, 0)
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		if date.Weekday() != request.Weekday {
			continue
		}
		occurrence, err := activationOccurrence(request, policy, location, date)
		if err != nil {
			return nil, err
		}
		if err := applyPastOutcome(&occurrence, request, nowDate, backdated); err != nil {
			return nil, err
		}
		result = append(result, occurrence)
	}
	return result, nil
}

func activationOccurrence(request ActivationRequest, policy Policy, location *time.Location, date time.Time) (Occurrence, error) {
	interval, err := resolveOccurrence(date, request.StartMinute, request.TeacherTimezone, policy.LessonDuration)
	if err != nil {
		return Occurrence{}, err
	}
	state, reason, err := activationScheduleState(request, policy, location, date, interval)
	if err != nil {
		return Occurrence{}, err
	}
	return Occurrence{ID: occurrenceID(request.ID, dateKey(date, location)), ContractID: request.ID, AssignmentID: request.AssignmentID, TeacherID: request.TeacherID, LearnerID: request.LearnerID, OriginalLocalDate: dateKey(date, location), OriginalStartAt: interval.Start, Interval: interval, ScheduleState: state, BillingOutcome: BillableOrdinary, UnitPriceMinor: policy.RegularLessonPrice.Minor, Currency: policy.Currency, OmissionReason: reason}, nil
}

func activationScheduleState(request ActivationRequest, policy Policy, location *time.Location, date time.Time, interval scheduling.Interval) (ScheduleState, string, error) {
	if intersectsAny(interval, request.Unavailable) {
		return Omitted, "planned_unavailability", nil
	}
	if !containsInterval(request.Availability, interval) {
		return "", "", fmt.Errorf("%w: %s", ErrUnavailable, dateKey(date, location))
	}
	lessons := append([]scheduling.Lesson(nil), request.ExistingLessons...)
	if scheduling.HasParticipantConflict(interval, request.TeacherID, request.LearnerID, lessons, scheduling.IntervalPolicyFromBusinessPolicy(policy)) {
		return "", "", fmt.Errorf("%w: %s", ErrParticipantConflict, dateKey(date, location))
	}
	return Scheduled, "", nil
}

func applyPastOutcome(occurrence *Occurrence, request ActivationRequest, nowDate time.Time, backdated bool) error {
	if !backdated || occurrence.OriginalLocalDate > dateKey(nowDate, nowDate.Location()) || occurrence.Interval.End.After(request.Now.UTC()) {
		return nil
	}
	outcome := request.PastOutcomes[occurrence.OriginalLocalDate]
	if outcome == "" {
		return ErrMissingPastOutcome
	}
	if outcome != "completed" && outcome != "learner_no_show" {
		return ErrInvalidContract
	}
	occurrence.Outcome = outcome
	return nil
}

func occurrenceID(contractID, localDate string) string {
	digest := sha256.Sum256([]byte(contractID + ":" + localDate))
	return hex.EncodeToString(digest[:])[:15]
}
