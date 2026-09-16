// This file centralizes assignment-scoped plan precedence and package selection.
package commercial

import (
	"sort"
	"time"
)

type EligibilityService struct{}

func (EligibilityService) Decide(input EligibilityInput) (EligibilityDecision, error) {
	if err := validateEligibilityInput(input); err != nil {
		return EligibilityDecision{}, err
	}
	if contractBlocks(input) {
		return EligibilityDecision{}, ErrContractActive
	}
	packageValue, token, found := eligiblePackage(input)
	if found {
		if input.RequestedPlan == AdHoc {
			return EligibilityDecision{}, ErrPlanPrecedence
		}
		return EligibilityDecision{Plan: PackagePlan, PackageID: packageValue.ID, TokenID: token.ID, TokenOrdinal: token.Ordinal}, nil
	}
	if input.RequestedPlan == PackagePlan {
		return EligibilityDecision{}, ErrPackageUnavailable
	}
	return EligibilityDecision{Plan: AdHoc}, nil
}

func validateEligibilityInput(input EligibilityInput) error {
	if !validAssignment(input.Assignment) {
		return ErrInvalidAssignment
	}
	if input.RequestedPlan != "" && input.RequestedPlan != PackagePlan && input.RequestedPlan != AdHoc {
		return ErrPlanPrecedence
	}
	return nil
}

func contractBlocks(input EligibilityInput) bool {
	return input.Contract != nil && input.Contract.AssignmentID == input.Assignment.ID && input.Contract.ActiveAt(input.LessonStart)
}

func EvaluateEligibility(input EligibilityInput) (EligibilityDecision, error) {
	return (EligibilityService{}).Decide(input)
}

func eligiblePackage(input EligibilityInput) (*Package, *Token, bool) {
	packages := make([]*Package, 0, len(input.Packages))
	for _, value := range input.Packages {
		if value == nil || !value.Assignment.Matches(input.Assignment.TeacherID, input.Assignment.LearnerID) || value.Assignment.ID != input.Assignment.ID {
			continue
		}
		if value.Status != PackageOpen || !value.ValidOn(input.LessonStart, input.Now) {
			continue
		}
		if value.AvailableCount() == 0 {
			continue
		}
		packages = append(packages, value)
	}
	sort.SliceStable(packages, func(i, j int) bool {
		if packages[i].PurchasedOn.Equal(packages[j].PurchasedOn) {
			return packages[i].ID < packages[j].ID
		}
		return packages[i].PurchasedOn.Before(packages[j].PurchasedOn)
	})
	if len(packages) == 0 {
		return nil, nil, false
	}
	tokens := packages[0].AvailableTokens()
	if len(tokens) == 0 {
		return nil, nil, false
	}
	return packages[0], &tokens[0], true
}

type OverlapState struct {
	Assignment           Assignment
	Now                  time.Time
	ActiveContract       bool
	Contract             *Contract
	Packages             []*Package
	FuturePackageLessons []LessonReference
	FutureAdHocLessons   []LessonReference
}
