// Package domain contains framework-independent values shared by scheduling and commercial rules.
package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidMoney       = errors.New("money must use a non-negative minor amount and ISO currency")
	ErrInvalidTeacherDate = errors.New("teacher-local date must use YYYY-MM-DD")
	ErrInvalidCorrection  = errors.New("correction reason must not be empty")
)

// Money stores an integer amount in the smallest unit of Currency.
type Money struct {
	Minor    int64  `json:"minor"`
	Currency string `json:"currency"`
}

func NewMoney(currency string, minor int64) (Money, error) {
	value := Money{Minor: minor, Currency: currency}
	if err := value.Validate(); err != nil {
		return Money{}, err
	}
	return value, nil
}

func (m Money) Validate() error {
	if m.Minor < 0 || len(m.Currency) != 3 || m.Currency != strings.ToUpper(m.Currency) {
		return ErrInvalidMoney
	}
	for _, character := range m.Currency {
		if character < 'A' || character > 'Z' {
			return ErrInvalidMoney
		}
	}
	return nil
}

// TeacherLocalDate is a normalized calendar date interpreted in a teacher timezone.
type TeacherLocalDate string

func NewTeacherLocalDate(value string) (TeacherLocalDate, error) {
	date := TeacherLocalDate(value)
	if err := date.Validate(); err != nil {
		return "", err
	}
	return date, nil
}

func (d TeacherLocalDate) Validate() error {
	parsed, err := time.Parse("2006-01-02", string(d))
	if err != nil || parsed.Format("2006-01-02") != string(d) {
		return ErrInvalidTeacherDate
	}
	return nil
}

func (d TeacherLocalDate) String() string { return string(d) }

type ActorRole string

const (
	TeacherActor ActorRole = "teacher"
	LearnerActor ActorRole = "learner"
	SystemActor  ActorRole = "system"
)

func (a ActorRole) Valid() bool { return a == TeacherActor || a == LearnerActor || a == SystemActor }

// Actor identifies the account or system that caused a domain transition.
type Actor struct {
	Role ActorRole `json:"role"`
	ID   string    `json:"id,omitempty"`
}

func (a Actor) Valid() bool { return a.Role.Valid() && (a.Role == SystemActor || a.ID != "") }

type PlanType string

const (
	RegularContractPlan PlanType = "regular_contract"
	PackagePlan         PlanType = "package"
	AdHocPlan           PlanType = "ad_hoc"
)

func (p PlanType) Valid() bool {
	return p == RegularContractPlan || p == PackagePlan || p == AdHocPlan
}

type ScheduleState string

const (
	ScheduledState ScheduleState = "scheduled"
	CancelledState ScheduleState = "cancelled"
	OmittedState   ScheduleState = "omitted"
)

func (s ScheduleState) Valid() bool {
	return s == ScheduledState || s == CancelledState || s == OmittedState
}

type Outcome string

const (
	AwaitingOutcome Outcome = "awaiting_outcome"
	Completed       Outcome = "completed"
	LearnerNoShow   Outcome = "learner_no_show"
)

func (o Outcome) Valid() bool {
	return o == AwaitingOutcome || o == Completed || o == LearnerNoShow
}

type SettlementState string

const (
	PendingSettlement   SettlementState = "pending_settlement"
	Paid                SettlementState = "paid"
	IntentionallyUnpaid SettlementState = "intentionally_unpaid"
	NotApplicable       SettlementState = "not_applicable"
	PendingPayment      SettlementState = "pending"
)

func (s SettlementState) Valid() bool {
	return s == PendingSettlement || s == Paid || s == IntentionallyUnpaid || s == NotApplicable || s == PendingPayment
}

// CorrectionReason is a required explanation for an administrative correction.
type CorrectionReason string

func NewCorrectionReason(value string) (CorrectionReason, error) {
	reason := CorrectionReason(strings.TrimSpace(value))
	if reason == "" {
		return "", ErrInvalidCorrection
	}
	return reason, nil
}

func (r CorrectionReason) Validate() error {
	if strings.TrimSpace(string(r)) == "" {
		return ErrInvalidCorrection
	}
	return nil
}
