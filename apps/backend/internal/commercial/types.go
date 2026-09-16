// Package commercial contains pure plan, package, token, and eligibility rules.
// It has no persistence, transport, clock, or framework dependencies.
package commercial

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/domain"
)

type PlanType = domain.PlanType

const (
	RegularContract = domain.RegularContractPlan
	PackagePlan     = domain.PackagePlan
	AdHoc           = domain.AdHocPlan
)

type Money = domain.Money
type PolicySnapshot = businesspolicy.PolicySnapshot

type TokenState string

const (
	TokenAvailable   TokenState = "available"
	TokenReserved    TokenState = "reserved"
	TokenUsed        TokenState = "used"
	TokenExpired     TokenState = "expired"
	TokenInvalidated TokenState = "invalidated"
)

type PackageStatus string

const (
	PackageOpen   PackageStatus = "open"
	PackageClosed PackageStatus = "closed"
)

type Token struct {
	ID          string
	Ordinal     int
	State       TokenState
	LessonID    string
	LessonStart time.Time
}

func (t Token) Available() bool { return t.State == TokenAvailable }

func (t Token) Reserved() bool { return t.State == TokenReserved }

type Assignment struct {
	ID        string
	TeacherID string
	LearnerID string
}

func (a Assignment) Matches(teacherID, learnerID string) bool {
	return a.TeacherID == teacherID && a.LearnerID == learnerID
}

type ContractStatus string

const (
	ContractActive  ContractStatus = "active"
	ContractEnded   ContractStatus = "ended"
	ContractPending ContractStatus = "pending"
)

type Contract struct {
	ID           string
	AssignmentID string
	Status       ContractStatus
	StartOn      time.Time
	EndOn        time.Time
}

type RegularContractRef = Contract

func (c Contract) ActiveAt(now time.Time) bool {
	if c.Status != ContractActive {
		return false
	}
	if c.StartOn.IsZero() || c.EndOn.IsZero() {
		return true
	}
	nowDate := dateUTC(now)
	return !nowDate.Before(dateUTC(c.StartOn)) && !nowDate.After(dateUTC(c.EndOn))
}

type LessonReference struct {
	ID           string
	AssignmentID string
	TeacherID    string
	LearnerID    string
	Plan         PlanType
	StartAt      time.Time
}

func (l LessonReference) BelongsTo(assignment Assignment) bool {
	return l.AssignmentID == assignment.ID && l.TeacherID == assignment.TeacherID && l.LearnerID == assignment.LearnerID
}

type EligibilityInput struct {
	Assignment      Assignment
	Now             time.Time
	LessonStart     time.Time
	RequestedPlan   PlanType
	Contract        *Contract
	Packages        []*Package
	TeacherTimezone string
}

type EligibilityDecision struct {
	Plan         PlanType
	PackageID    string
	TokenID      string
	TokenOrdinal int
}

var (
	ErrInvalidAssignment     = errors.New("assignment is required")
	ErrAssignmentMismatch    = errors.New("commercial obligation belongs to another assignment")
	ErrInvalidPlan           = errors.New("commercial plan is invalid")
	ErrPlanPrecedence        = errors.New("requested plan is lower than the active plan")
	ErrContractActive        = errors.New("regular contract is active")
	ErrPackageUnavailable    = errors.New("no valid package token is available")
	ErrPackageExpired        = errors.New("package is expired")
	ErrPackageClosed         = errors.New("package is closed")
	ErrInvalidPurchaseDate   = errors.New("purchase date cannot be in the future")
	ErrInvalidTimezone       = errors.New("teacher timezone is invalid")
	ErrInvalidPolicy         = errors.New("commercial policy is invalid")
	ErrInvalidLesson         = errors.New("lesson reference is invalid")
	ErrTokenNotFound         = errors.New("package token is not reserved for the lesson")
	ErrLearnerChangeTooLate  = errors.New("learner change is inside the 24-hour cutoff")
	ErrReplacementExpired    = errors.New("replacement starts after package validity")
	ErrFuturePackageLesson   = errors.New("future package lesson must be resolved before closure")
	ErrCloseReasonRequired   = errors.New("package closure reason is required")
	ErrPackageRenewal        = errors.New("a prior package still has an available token")
	ErrActiveContractOverlap = errors.New("regular contract overlaps package obligations")
	ErrAdHocOverlap          = errors.New("future ad hoc lessons require conversion or cancellation")
	ErrConversionLimit       = errors.New("selected ad hoc lessons exceed package token capacity")
	ErrConversionNotEligible = errors.New("selected lesson cannot be converted")
	ErrInvalidRefund         = errors.New("refund information is invalid")
	ErrCorrectionReason      = errors.New("package correction requires a reason")
	ErrCorrectionEvent       = errors.New("correction requires an event reference and timestamp")
	ErrInvalidTokenState     = errors.New("token state transition is invalid")
)

func dateUTC(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func localDate(value time.Time, timezone string) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, ErrInvalidLesson
	}
	location, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" {
		return time.Time{}, ErrInvalidTimezone
	}
	year, month, day := value.In(location).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func cloneTokens(tokens []Token) []Token {
	copyTokens := append([]Token(nil), tokens...)
	sort.Slice(copyTokens, func(i, j int) bool { return copyTokens[i].Ordinal < copyTokens[j].Ordinal })
	return copyTokens
}

func validatePlan(plan PlanType) error {
	if plan != RegularContract && plan != PackagePlan && plan != AdHoc {
		return fmt.Errorf("%w: %q", ErrInvalidPlan, plan)
	}
	return nil
}

func validReason(reason string) bool { return strings.TrimSpace(reason) != "" }
