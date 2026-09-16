// Package regularcontract models fixed weekly contracts and their materialized lesson series.
// It contains no persistence, HTTP, clock, or framework dependencies.
package regularcontract

import (
	"errors"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

var (
	ErrInvalidContract       = errors.New("contract is invalid")
	ErrUnauthorized          = errors.New("actor is not a contract participant")
	ErrTeacherRequired       = errors.New("assigned teacher is required")
	ErrPastStart             = errors.New("past contract start requires an administrative reason and outcomes")
	ErrMissingPastOutcome    = errors.New("every past occurrence requires an explicit outcome")
	ErrUnavailable           = errors.New("contract occurrence does not fit teacher availability")
	ErrParticipantConflict   = errors.New("contract occurrence conflicts with a participant lesson")
	ErrCommercialOverlap     = errors.New("contract overlaps an existing commercial obligation")
	ErrAlreadyRescheduled    = errors.New("contract occurrence was already rescheduled by the learner")
	ErrAllowanceExhausted    = errors.New("contract allowance is exhausted")
	ErrCutoff                = errors.New("learner change is inside the 24-hour cutoff")
	ErrReplacementDeadline   = errors.New("replacement exceeds the contract deadline")
	ErrReplacementHorizon    = errors.New("learner replacement is outside the booking horizon")
	ErrStartedOccurrence     = errors.New("started or past occurrence is immutable")
	ErrInvalidAmendment      = errors.New("price amendment must start on a future month boundary")
	ErrInvalidEarlyEnd       = errors.New("earlier mutual end requires a reason")
	ErrTimezoneChangeBlocked = errors.New("timezone change is blocked by future obligations")
	ErrAssignmentBlocked     = errors.New("assignment has unresolved commercial obligations")
)

type Status string

const (
	Active      Status = "active"
	NoticeGiven Status = "notice_given"
	Ended       Status = "ended"
)

type ScheduleState = domain.ScheduleState

const (
	Scheduled = domain.ScheduledState
	Cancelled = domain.CancelledState
	Omitted   = domain.OmittedState
)

type BillingOutcome = ledger.BillingOutcome

const (
	BillableOrdinary    BillingOutcome = ledger.BillableOrdinary
	BillableLateCancel  BillingOutcome = ledger.BillableLateCancel
	BillableExhausted   BillingOutcome = ledger.BillableExhausted
	BillableNoShow      BillingOutcome = ledger.NoShow
	FreeLearnerCancel   BillingOutcome = ledger.FreeLearnerCancel
	TeacherCancel       BillingOutcome = ledger.TeacherCancel
	PlannedOmission     BillingOutcome = ledger.PlannedOmission
	ContractTermination BillingOutcome = ledger.ContractTermination
)

type ActorRole = domain.ActorRole

const (
	Teacher = domain.TeacherActor
	Learner = domain.LearnerActor
)

type Actor = domain.Actor

// Policy contains the values that affect a contract's lifecycle. Records keep a copy as a snapshot.
type Policy = businesspolicy.Policy

func DefaultPolicy() Policy {
	return businesspolicy.Current()
}

// Occurrence is a stable contract identity with a mutable current interval.
type Occurrence struct {
	ID                      string
	ContractID              string
	AssignmentID            string
	TeacherID               string
	LearnerID               string
	OriginalLocalDate       string
	OriginalStartAt         time.Time
	Interval                scheduling.Interval
	ScheduleState           ScheduleState
	Outcome                 string
	BillingOutcome          BillingOutcome
	UnitPriceMinor          int64
	Currency                string
	IndividuallyRescheduled bool
	OmissionReason          string
}

func (o Occurrence) Billable() bool {
	if o.ScheduleState == Omitted {
		return false
	}
	return o.BillingOutcome != FreeLearnerCancel && o.BillingOutcome != TeacherCancel && o.BillingOutcome != PlannedOmission && o.BillingOutcome != ContractTermination
}

// Event is immutable history used to derive contract allowances.
type Event struct {
	ID            string
	Type          string
	ContractID    string
	OccurrenceID  string
	OriginalMonth string
	Actor         Actor
	At            time.Time
	Reason        string
	CorrectsEvent string
	PriorInterval scheduling.Interval
	NewInterval   scheduling.Interval
}

const (
	EventActivated             = "contract_activated"
	EventLearnerRescheduled    = "learner_rescheduled"
	EventTeacherRescheduled    = "teacher_rescheduled"
	EventLearnerFreeCancel     = "learner_free_cancellation"
	EventLearnerLateCancel     = "learner_late_cancellation"
	EventLearnerBillableCancel = "learner_billable_cancellation"
	EventTeacherCancel         = "teacher_cancellation"
	EventOutcomeCompleted      = "lesson_completed"
	EventOutcomeNoShow         = "learner_no_show"
	EventScheduleChanged       = "contract_schedule_changed"
	EventNoticeSubmitted       = "contract_notice_submitted"
	EventEarlyEnded            = "contract_ended_early"
	EventPriceAmended          = "contract_price_amended"
	EventRenewed               = "contract_renewed"
)

// Amendment changes only obligations in its effective month and later.
type Amendment struct {
	EffectiveMonth string
	PriceMinor     int64
	Currency       string
	CreatedAt      time.Time
	Actor          Actor
}

type RegularContract struct {
	ID              string
	AssignmentID    string
	TeacherID       string
	LearnerID       string
	TeacherTimezone string
	StartOn         time.Time
	EndOn           time.Time
	EffectiveEndOn  time.Time
	Weekday         time.Weekday
	StartMinute     int
	PriceMinor      int64
	Currency        string
	Status          Status
	NoticeAt        time.Time
	Policy          Policy
	PolicySnapshot  businesspolicy.PolicySnapshot
	Occurrences     []Occurrence
	Amendments      []Amendment
	Events          []Event
}

type AllowanceBalance struct {
	MonthlyReschedulesRemaining int
	FreeCancellationsRemaining  int
}

type SeriesPartition struct {
	Near  []Occurrence
	Later []Occurrence
}

type MonthSnapshot struct {
	Month         string
	OccurrenceIDs []string
	BillableCount int
	AmountMinor   int64
	Forecast      bool
	Charge        bool
	DueOn         time.Time
}

type ObligationSet struct {
	ActiveContract         bool
	AvailablePackageTokens int
	ReservedPackageLessons int
	FutureScheduledLessons int
}
