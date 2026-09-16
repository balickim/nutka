// This file contains calendar projection helpers and assignment-scoped data filters.
package commercialread

import (
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
)

func readPolicy(policy businesspolicy.Policy) (businesspolicy.Policy, error) {
	if policy.Version == "" || policy.Validate() != nil {
		return businesspolicy.Policy{}, ErrPolicy
	}
	return policy, nil
}

func authorize(assignment Assignment, viewer Viewer) error {
	if !viewer.valid() || assignment.ID == "" || viewer.Role == TeacherRole && assignment.TeacherID != viewer.ID || viewer.Role == LearnerRole && assignment.LearnerID != viewer.ID {
		return ErrUnauthorized
	}
	return nil
}

func tokenBalance(value *commercial.Package) TokenBalance {
	result := TokenBalance{Total: len(value.Tokens)}
	for _, token := range value.Tokens {
		switch token.State {
		case commercial.TokenAvailable:
			result.Available++
		case commercial.TokenReserved:
			result.Reserved++
		case commercial.TokenUsed:
			result.Used++
		case commercial.TokenExpired:
			result.Expired++
		case commercial.TokenInvalidated:
			result.Invalidated++
		}
	}
	return result
}

func displayPackage(values []*commercial.Package, assignment Assignment) *commercial.Package {
	var selected *commercial.Package
	for _, value := range values {
		if value == nil || value.Assignment.ID != assignment.ID || value.Assignment.TeacherID != assignment.TeacherID || value.Assignment.LearnerID != assignment.LearnerID {
			continue
		}
		if selected == nil || value.PurchasedOn.After(selected.PurchasedOn) || value.PurchasedOn.Equal(selected.PurchasedOn) && value.ID > selected.ID {
			selected = value
		}
	}
	return selected
}

func eligiblePlan(input SummaryInput) (domain.PlanType, error) {
	if input.Contract != nil && input.Contract.ActiveAt(input.Now) {
		return domain.RegularContractPlan, nil
	}
	packages := make([]*commercial.Package, 0, len(input.Packages))
	for _, value := range input.Packages {
		if value != nil && value.Assignment.ID == input.Assignment.ID && value.Assignment.TeacherID == input.Assignment.TeacherID && value.Assignment.LearnerID == input.Assignment.LearnerID {
			packages = append(packages, value)
		}
	}
	decision, err := commercial.EvaluateEligibility(commercial.EligibilityInput{Assignment: commercial.Assignment{ID: input.Assignment.ID, TeacherID: input.Assignment.TeacherID, LearnerID: input.Assignment.LearnerID}, Now: input.Now, LessonStart: input.Now, Packages: packages})
	if err != nil {
		return "", err
	}
	return decision.Plan, nil
}

func validateZones(input SummaryInput) error {
	if input.Contract != nil {
		if _, err := time.LoadLocation(input.Contract.TeacherTimezone); err != nil || input.Contract.TeacherTimezone == "" {
			return ErrTimezone
		}
	}
	for _, value := range input.Packages {
		if value == nil {
			continue
		}
		if value.TeacherZone == "" {
			return ErrTimezone
		}
		if _, err := time.LoadLocation(value.TeacherZone); err != nil {
			return ErrTimezone
		}
	}
	return nil
}

func paymentSummary(lessons []ledger.AdHocLesson, charges []ledger.Charge, credits []ledger.Credit, now time.Time, location *time.Location) PaymentSummary {
	value := ledger.BuildFinancialSummary(lessons, charges, credits, now, location)
	return PaymentSummary{Pending: int64(value.PendingSettlement), IntentionallyUnpaid: int64(value.UnpaidAdHoc + value.UnpaidCharges), Overdue: int64(value.OverdueCharges), CreditMinor: value.OpenCreditMinor, Currency: value.Currency}
}

func contractLocation(contract *regularcontract.RegularContract) (*time.Location, error) {
	if contract == nil {
		return nil, ErrTimezone
	}
	location, err := time.LoadLocation(contract.TeacherTimezone)
	if err != nil || contract.TeacherTimezone == "" {
		return nil, ErrTimezone
	}
	return location, nil
}

func locationFor(data AssignmentData) (*time.Location, error) {
	if data.Contract != nil {
		return contractLocation(data.Contract)
	}
	for _, value := range data.Packages {
		if value != nil {
			location, err := time.LoadLocation(value.TeacherZone)
			if err == nil {
				return location, nil
			}
			return nil, ErrTimezone
		}
	}
	return time.UTC, nil
}

func sortedAssignments(values []AssignmentData) []AssignmentData {
	result := append([]AssignmentData(nil), values...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Assignment.ID < result[j].Assignment.ID })
	return result
}

func horizon(start, now time.Time, policy businesspolicy.Policy) bool {
	return start.After(now.UTC()) && !start.After(now.UTC().Add(policy.BookingHorizon))
}

func appendNearLessons(target *[]LessonView, assignmentID string, lessons []LessonInput, now time.Time, policy businesspolicy.Policy, viewer Viewer) error {
	for _, lesson := range lessons {
		if lesson.AssignmentID != assignmentID || !horizon(lesson.StartAt, now, policy) || lesson.ScheduleState == domain.OmittedState {
			continue
		}
		if viewer.Role == TeacherRole && lesson.TeacherID != viewer.ID || viewer.Role == LearnerRole && lesson.LearnerID != viewer.ID {
			continue
		}
		view, err := lessonView(lesson, policy)
		if err != nil {
			return err
		}
		*target = append(*target, view)
	}
	return nil
}

func scopedAdHoc(values []ledger.AdHocLesson, assignmentID string) []ledger.AdHocLesson {
	result := make([]ledger.AdHocLesson, 0, len(values))
	for _, value := range values {
		if value.AssignmentID == assignmentID {
			result = append(result, value)
		}
	}
	return result
}

func scopedCharges(values []ledger.Charge, assignmentID string) []ledger.Charge {
	result := make([]ledger.Charge, 0, len(values))
	for _, value := range values {
		if value.AssignmentID == assignmentID {
			result = append(result, value)
		}
	}
	return result
}

func scopedCredits(values []ledger.Credit, assignmentID string) []ledger.Credit {
	result := make([]ledger.Credit, 0, len(values))
	for _, value := range values {
		if value.AssignmentID == assignmentID {
			result = append(result, value)
		}
	}
	return result
}

func lessonView(value LessonInput, policy businesspolicy.Policy) (LessonView, error) {
	duration := value.DurationMinutes
	if duration == 0 {
		duration = int(policy.LessonDuration / time.Minute)
	}
	if duration != int(policy.LessonDuration/time.Minute) || !value.StartAt.IsZero() && !value.EndAt.IsZero() && !value.EndAt.Equal(value.StartAt.Add(policy.LessonDuration)) {
		return LessonView{}, ErrDuration
	}
	view := LessonView{ID: value.ID, Teacher: value.TeacherID, Learner: value.LearnerID, Assignment: value.AssignmentID, PlanType: string(value.Plan), PackageToken: value.PackageTokenID, Contract: value.ContractID, StartAt: instant(value.StartAt), EndAt: instant(value.EndAt), DurationMinutes: duration, UnitPriceMinor: value.UnitPriceMinor, Currency: value.Currency, PolicyVersion: value.PolicyVersion, ScheduleState: string(value.ScheduleState), Outcome: string(value.Outcome)}
	if !value.StartAt.IsZero() && !value.EndAt.IsZero() {
		view.ProtectedStartAt = instant(value.StartAt.Add(-policy.ParticipantBuffer))
		view.ProtectedEndAt = instant(value.EndAt.Add(policy.ParticipantBuffer))
	}
	return view, nil
}

func occurrenceView(value regularcontract.Occurrence) ContractOccurrenceView {
	return ContractOccurrenceView{ID: value.ID, Contract: value.ContractID, Assignment: value.AssignmentID, OriginalLocalDate: value.OriginalLocalDate, StartAt: instant(value.Interval.Start), EndAt: instant(value.Interval.End), ScheduleState: string(value.ScheduleState), Outcome: value.Outcome, UnitPriceMinor: value.UnitPriceMinor, Currency: value.Currency}
}

func occurrenceLessonView(value regularcontract.Occurrence, policy businesspolicy.Policy) (LessonView, error) {
	return lessonView(LessonInput{ID: value.ID, AssignmentID: value.AssignmentID, TeacherID: value.TeacherID, LearnerID: value.LearnerID, Plan: domain.RegularContractPlan, ContractID: value.ContractID, StartAt: value.Interval.Start, EndAt: value.Interval.End, DurationMinutes: int(policy.LessonDuration / time.Minute), UnitPriceMinor: value.UnitPriceMinor, Currency: value.Currency, ScheduleState: value.ScheduleState, Outcome: domain.Outcome(value.Outcome), PolicyVersion: policy.Version}, policy)
}

func partitionContract(contract regularcontract.RegularContract, now time.Time, bookingHorizon time.Duration) ([]regularcontract.Occurrence, []regularcontract.Occurrence) {
	near, later := []regularcontract.Occurrence{}, []regularcontract.Occurrence{}
	limit := now.UTC().Add(bookingHorizon)
	for _, value := range contract.Occurrences {
		start := value.Interval.Start.UTC()
		if !start.After(now.UTC()) {
			continue
		}
		if !start.After(limit) {
			near = append(near, value)
		} else {
			later = append(later, value)
		}
	}
	sort.Slice(near, func(i, j int) bool { return near[i].Interval.Start.Before(near[j].Interval.Start) })
	sort.Slice(later, func(i, j int) bool { return later[i].Interval.Start.Before(later[j].Interval.Start) })
	return near, later
}

func containsLesson(lessons []LessonInput, id string) bool {
	for _, lesson := range lessons {
		if lesson.ID == id {
			return true
		}
	}
	return false
}

func awaitingCount(lessons []LessonInput, now time.Time) int {
	count := 0
	for _, lesson := range lessons {
		if lesson.ScheduleState == domain.ScheduledState && !lesson.EndAt.IsZero() && !lesson.EndAt.After(now) && lesson.Outcome == "" {
			count++
		}
	}
	return count
}

func pendingCount(lessons []ledger.AdHocLesson) int {
	count := 0
	for _, lesson := range lessons {
		if lesson.SettlementState == ledger.PendingSettlement {
			count++
		}
	}
	return count
}
