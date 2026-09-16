// This file validates plan transitions and returns atomic conversion decisions.
package commercial

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
)

type ConversionDecision struct {
	Package          *Package
	ConvertedLessons []LessonReference
}

type ContractConversionDecision struct {
	ConvertedLessons []LessonReference
}

func ValidatePackagePurchase(state OverlapState, selectedIDs []string) error {
	if !validAssignment(state.Assignment) {
		return ErrInvalidAssignment
	}
	if packageContractBlocks(state) {
		return ErrActiveContractOverlap
	}
	if hasRenewablePackage(state) {
		return ErrPackageRenewal
	}
	if len(selectedIDs) > businesspolicy.Current().PackageTokenCount {
		return ErrConversionLimit
	}
	selected, err := selectedLessons(state, selectedIDs)
	if err != nil {
		return err
	}
	if len(selected) != len(state.FutureAdHocLessons) {
		return ErrAdHocOverlap
	}
	return nil
}

func ValidateContractActivation(state OverlapState, selectedIDs []string) error {
	if !validAssignment(state.Assignment) {
		return ErrInvalidAssignment
	}
	if contractActivationBlocked(state) {
		return ErrContractActive
	}
	if hasContractPackageObligation(state) {
		return ErrActiveContractOverlap
	}
	if len(selectedIDs) != len(state.FutureAdHocLessons) {
		return ErrAdHocOverlap
	}
	_, err := selectedLessons(state, selectedIDs)
	return err
}

func packageContractBlocks(state OverlapState) bool {
	return state.ActiveContract || (state.Contract != nil && state.Contract.ActiveAt(state.Now))
}

func contractActivationBlocked(state OverlapState) bool {
	return state.ActiveContract || (state.Contract != nil && state.Contract.ActiveAt(state.Now))
}

func hasRenewablePackage(state OverlapState) bool {
	for _, value := range state.Packages {
		if matchesPackage(value, state.Assignment) && value.Status == PackageOpen && validToday(value, state.Now) && !value.CanRenew() {
			return true
		}
	}
	return false
}

func hasContractPackageObligation(state OverlapState) bool {
	for _, value := range state.Packages {
		if !matchesPackage(value, state.Assignment) || value.Status != PackageOpen {
			continue
		}
		if (value.AvailableCount() > 0 && validToday(value, state.Now)) || hasFuturePackage(value, state.Now) {
			return true
		}
	}
	return false
}

func matchesPackage(value *Package, assignment Assignment) bool {
	return value != nil && value.Assignment.ID == assignment.ID && value.Assignment.Matches(assignment.TeacherID, assignment.LearnerID)
}

func DecidePackagePurchase(request PurchaseRequest, state OverlapState, selectedIDs []string) (ConversionDecision, error) {
	if err := ValidatePackagePurchase(state, selectedIDs); err != nil {
		return ConversionDecision{}, err
	}
	created, err := NewPackage(request)
	if err != nil {
		return ConversionDecision{}, err
	}
	selected, err := selectedLessons(state, selectedIDs)
	if err != nil {
		return ConversionDecision{}, err
	}
	converted := make([]LessonReference, 0, len(selected))
	for _, lesson := range selected {
		if _, err := created.Reserve(lesson, request.Now); err != nil {
			return ConversionDecision{}, err
		}
		lesson.Plan = PackagePlan
		converted = append(converted, lesson)
	}
	return ConversionDecision{Package: &created, ConvertedLessons: converted}, nil
}

func DecideContractActivation(state OverlapState, selectedIDs []string) (ContractConversionDecision, error) {
	if err := ValidateContractActivation(state, selectedIDs); err != nil {
		return ContractConversionDecision{}, err
	}
	selected, err := selectedLessons(state, selectedIDs)
	if err != nil {
		return ContractConversionDecision{}, err
	}
	for index := range selected {
		selected[index].Plan = RegularContract
	}
	return ContractConversionDecision{ConvertedLessons: selected}, nil
}

func ConvertFutureAdHocLessons(state OverlapState, selectedIDs []string) ([]LessonReference, error) {
	decision, err := DecideContractActivation(state, selectedIDs)
	if err != nil {
		return nil, err
	}
	return decision.ConvertedLessons, nil
}

func validAssignment(assignment Assignment) bool {
	return assignment.ID != "" && assignment.TeacherID != "" && assignment.LearnerID != ""
}

func hasFuturePackage(value *Package, now time.Time) bool {
	for _, token := range value.Tokens {
		if token.State == TokenReserved && token.LessonStart.After(now) {
			return true
		}
	}
	return false
}

func validToday(value *Package, now time.Time) bool {
	date, err := localDate(now, value.TeacherZone)
	return err == nil && !date.After(value.ValidThrough)
}

func selectedLessons(state OverlapState, selectedIDs []string) ([]LessonReference, error) {
	if len(selectedIDs) == 0 {
		return nil, nil
	}
	selected := make([]LessonReference, 0, len(selectedIDs))
	seen := make(map[string]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		if _, exists := seen[id]; exists {
			return nil, ErrConversionNotEligible
		}
		seen[id] = struct{}{}
		lesson, found := findSelectableLesson(state, id)
		if !found {
			return nil, ErrConversionNotEligible
		}
		selected = append(selected, lesson)
	}
	return selected, nil
}

func findSelectableLesson(state OverlapState, id string) (LessonReference, bool) {
	if id == "" {
		return LessonReference{}, false
	}
	for _, lesson := range state.FutureAdHocLessons {
		if lesson.ID == id && lesson.BelongsTo(state.Assignment) && lesson.Plan == AdHoc && !lesson.StartAt.IsZero() {
			return lesson, true
		}
	}
	return LessonReference{}, false
}
