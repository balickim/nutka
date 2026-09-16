// This file composes domain aggregates into bounded calendar and assignment views.
package commercialread

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
)

func BuildAssignmentSummary(input SummaryInput) (AssignmentCommercialSummary, error) {
	if err := validateSummaryInput(input); err != nil {
		return AssignmentCommercialSummary{}, err
	}
	result := AssignmentCommercialSummary{AssignmentID: input.Assignment.ID}
	if err := validateZones(input); err != nil {
		return AssignmentCommercialSummary{}, err
	}
	contract, err := contractSummary(input)
	if err != nil {
		return AssignmentCommercialSummary{}, err
	}
	result.Contract = contract
	if selected := displayPackage(input.Packages, input.Assignment); selected != nil {
		result.Package = &PackageSummary{ID: selected.ID, ValidThrough: selected.ValidThrough.Format("2006-01-02"), TokenBalance: tokenBalance(selected)}
	}
	if input.Assignment.Active {
		plan, err := eligiblePlan(input)
		if err != nil {
			return AssignmentCommercialSummary{}, err
		}
		result.ActivePlan = plan
	}
	result.Payments = paymentSummary(scopedAdHoc(input.AdHocLessons, input.Assignment.ID), scopedCharges(input.Charges, input.Assignment.ID), scopedCredits(input.Credits, input.Assignment.ID), input.Now, input.Location)
	return result, nil
}

func validateSummaryInput(input SummaryInput) error {
	if err := authorize(input.Assignment, input.Viewer); err != nil {
		return err
	}
	if input.Now.IsZero() {
		return ErrPolicy
	}
	if err := businesspolicy.ValidateCurrent(); err != nil {
		return err
	}
	if input.Contract != nil && (input.Contract.AssignmentID != input.Assignment.ID || input.Contract.TeacherID != input.Assignment.TeacherID || input.Contract.LearnerID != input.Assignment.LearnerID) {
		return ErrUnauthorized
	}
	return nil
}

func contractSummary(input SummaryInput) (*ContractSummary, error) {
	if input.Contract == nil || !input.Contract.ActiveAt(input.Now) {
		return nil, nil
	}
	location, err := contractLocation(input.Contract)
	if err != nil {
		return nil, err
	}
	allowance := input.Contract.Allowances(input.Now.In(location).Format("2006-01"))
	return &ContractSummary{ID: input.Contract.ID, Status: string(input.Contract.Status), StartOn: dateString(input.Contract.StartOn), EndOn: dateString(input.Contract.EffectiveEndOn), RemainingMonthlyReschedules: allowance.MonthlyReschedulesRemaining, RemainingFreeCancellations: allowance.FreeCancellationsRemaining, PriceMinor: input.Contract.PriceMinor, Currency: input.Contract.Currency}, nil
}

func BuildTeacherCalendar(request CalendarRequest) (TeacherCalendar, error) {
	if request.Viewer.Role != TeacherRole || !request.Viewer.valid() || request.Now.IsZero() {
		return TeacherCalendar{}, ErrUnauthorized
	}
	policy, err := readPolicy(request.Policy)
	if err != nil {
		return TeacherCalendar{}, err
	}
	result := TeacherCalendar{Assignments: []AssignmentView{}, CommercialSummaries: []AssignmentCommercialSummary{}, NearTermLessons: []LessonView{}, LaterContractLessons: []ContractOccurrenceView{}}
	for _, data := range sortedAssignments(request.Assignments) {
		if err := appendTeacherAssignment(&result, data, request, policy); err != nil {
			return TeacherCalendar{}, err
		}
	}
	sortLessons(result.NearTermLessons)
	sortOccurrences(result.LaterContractLessons)
	return result, nil
}

func appendTeacherAssignment(result *TeacherCalendar, data AssignmentData, request CalendarRequest, policy businesspolicy.Policy) error {
	if authorize(data.Assignment, request.Viewer) != nil {
		return nil
	}
	result.Assignments = append(result.Assignments, AssignmentView{ID: data.Assignment.ID, Active: data.Assignment.Active})
	location, err := locationFor(data)
	if err != nil {
		return err
	}
	summary, err := BuildAssignmentSummary(SummaryInput{Assignment: data.Assignment, Contract: data.Contract, Packages: data.Packages, AdHocLessons: data.AdHocLessons, Charges: data.Charges, Credits: data.Credits, Now: request.Now, Location: location, Viewer: request.Viewer})
	if err != nil {
		return err
	}
	result.CommercialSummaries = append(result.CommercialSummaries, summary)
	if err := appendNearLessons(&result.NearTermLessons, data.Assignment.ID, data.Lessons, request.Now, policy, request.Viewer); err != nil {
		return err
	}
	if err := appendTeacherContract(result, data, request.Now, policy); err != nil {
		return err
	}
	appendTeacherWork(result, data, request.Now, location)
	return nil
}

func appendTeacherContract(result *TeacherCalendar, data AssignmentData, now time.Time, policy businesspolicy.Policy) error {
	if data.Contract == nil {
		return nil
	}
	near, later := partitionContract(*data.Contract, now, policy.BookingHorizon)
	for _, occurrence := range near {
		if !containsLesson(data.Lessons, occurrence.ID) {
			lesson, err := occurrenceLessonView(occurrence, policy)
			if err != nil {
				return err
			}
			result.NearTermLessons = append(result.NearTermLessons, lesson)
		}
	}
	for _, occurrence := range later {
		result.LaterContractLessons = append(result.LaterContractLessons, occurrenceView(occurrence))
	}
	return nil
}

func appendTeacherWork(result *TeacherCalendar, data AssignmentData, now time.Time, location *time.Location) {
	result.UnresolvedWork.AwaitingOutcome += awaitingCount(data.Lessons, now)
	result.UnresolvedWork.PendingSettlement += pendingCount(data.AdHocLessons)
	financial := ledger.BuildFinancialSummary(scopedAdHoc(data.AdHocLessons, data.Assignment.ID), scopedCharges(data.Charges, data.Assignment.ID), scopedCredits(data.Credits, data.Assignment.ID), now, location)
	result.UnresolvedWork.UnpaidCharges += financial.UnpaidCharges
	result.UnresolvedWork.OverdueCharges += financial.OverdueCharges
}

func BuildLearnerCalendar(request CalendarRequest) (LearnerCalendar, error) {
	if request.Viewer.Role != LearnerRole || !request.Viewer.valid() || request.Now.IsZero() {
		return LearnerCalendar{}, ErrUnauthorized
	}
	policy, err := readPolicy(request.Policy)
	if err != nil {
		return LearnerCalendar{}, err
	}
	result := LearnerCalendar{Assignments: []AssignmentView{}, CommercialSummaries: []AssignmentCommercialSummary{}, NearTermLessons: []LessonView{}, PaymentSummary: []PaymentSummary{}, HistorySummary: []HistorySummary{}}
	for _, data := range sortedAssignments(request.Assignments) {
		if err := appendLearnerAssignment(&result, data, request, policy); err != nil {
			return LearnerCalendar{}, err
		}
	}
	sortLessons(result.NearTermLessons)
	return result, nil
}

func appendLearnerAssignment(result *LearnerCalendar, data AssignmentData, request CalendarRequest, policy businesspolicy.Policy) error {
	if authorize(data.Assignment, request.Viewer) != nil {
		return nil
	}
	location, err := locationFor(data)
	if err != nil {
		return err
	}
	summary, err := BuildAssignmentSummary(SummaryInput{Assignment: data.Assignment, Contract: data.Contract, Packages: data.Packages, AdHocLessons: data.AdHocLessons, Charges: data.Charges, Credits: data.Credits, Now: request.Now, Location: location, Viewer: request.Viewer})
	if err != nil {
		return err
	}
	result.Assignments = append(result.Assignments, AssignmentView{ID: data.Assignment.ID, Active: data.Assignment.Active})
	result.CommercialSummaries = append(result.CommercialSummaries, summary)
	result.PaymentSummary = append(result.PaymentSummary, summary.Payments)
	result.HistorySummary = append(result.HistorySummary, HistorySummary{Assignment: data.Assignment.ID, EventCount: len(data.HistoryEvents)})
	if err := appendNearLessons(&result.NearTermLessons, data.Assignment.ID, data.Lessons, request.Now, policy, request.Viewer); err != nil {
		return err
	}
	return appendLearnerContract(result, data, request.Now, policy)
}

func appendLearnerContract(result *LearnerCalendar, data AssignmentData, now time.Time, policy businesspolicy.Policy) error {
	if data.Contract == nil {
		return nil
	}
	near, _ := partitionContract(*data.Contract, now, policy.BookingHorizon)
	for _, occurrence := range near {
		if !containsLesson(data.Lessons, occurrence.ID) {
			lesson, err := occurrenceLessonView(occurrence, policy)
			if err != nil {
				return err
			}
			result.NearTermLessons = append(result.NearTermLessons, lesson)
		}
	}
	return nil
}
