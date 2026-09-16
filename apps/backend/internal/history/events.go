// Package history defines immutable business events and pure correction resolution.
// It contains no PocketBase or HTTP dependencies.
package history

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// ActorRole identifies the authenticated or system actor that caused an event.
type ActorRole string

const (
	TeacherActor ActorRole = "teacher"
	LearnerActor ActorRole = "learner"
	SystemActor  ActorRole = "system"
)

// EventType is the stable English machine value persisted for a transition.
type EventType string

const (
	LessonCreated                 EventType = "lesson_created"
	LessonConverted               EventType = "lesson_converted"
	LessonRescheduled             EventType = "lesson_rescheduled"
	LessonCancelled               EventType = "lesson_cancelled"
	LessonOutcomeRecorded         EventType = "lesson_outcome_recorded"
	SettlementChanged             EventType = "settlement_changed"
	PackagePurchased              EventType = "package_purchased"
	PackageTokenReserved          EventType = "package_token_reserved"
	PackageTokenUsed              EventType = "package_token_used"
	PackageTokenReturned          EventType = "package_token_returned"
	PackageTokenExpired           EventType = "package_token_expired"
	PackageTokenExtended          EventType = "package_token_extended"
	PackageTokenInvalidated       EventType = "package_token_invalidated"
	PackageValidityExtended       EventType = "package_validity_extended"
	PackageClosed                 EventType = "package_closed"
	ContractActivated             EventType = "contract_activated"
	ContractScheduleChanged       EventType = "contract_schedule_changed"
	ContractOccurrenceRescheduled EventType = "contract_occurrence_rescheduled"
	ContractOccurrenceOmitted     EventType = "contract_occurrence_omitted"
	ContractOccurrenceRestored    EventType = "contract_occurrence_restored"
	ContractAmended               EventType = "contract_amended"
	ContractNoticeSubmitted       EventType = "contract_notice_submitted"
	ContractRenewed               EventType = "contract_renewed"
	ContractEnded                 EventType = "contract_ended"
	ChargeCreated                 EventType = "charge_created"
	ChargeAdjusted                EventType = "charge_adjusted"
	PaymentRecorded               EventType = "payment_recorded"
	CreditCreated                 EventType = "credit_created"
	CreditApplied                 EventType = "credit_applied"
	RefundRecorded                EventType = "refund_recorded"
	AvailabilityConsequence       EventType = "availability_consequence"
	AvailabilityChanged           EventType = "availability_changed"
	AdministrativeCorrection      EventType = "administrative_correction"
)

var eventTypes = []EventType{
	LessonCreated, LessonConverted, LessonRescheduled, LessonCancelled,
	LessonOutcomeRecorded, SettlementChanged, PackagePurchased,
	PackageTokenReserved, PackageTokenUsed, PackageTokenReturned,
	PackageTokenExpired, PackageTokenExtended, PackageTokenInvalidated,
	PackageValidityExtended, PackageClosed, ContractActivated,
	ContractScheduleChanged, ContractOccurrenceRescheduled,
	ContractOccurrenceOmitted, ContractOccurrenceRestored,
	ContractAmended, ContractNoticeSubmitted,
	ContractRenewed, ContractEnded, ChargeCreated, ChargeAdjusted,
	PaymentRecorded, CreditCreated, CreditApplied, RefundRecorded,
	AvailabilityConsequence, AvailabilityChanged, AdministrativeCorrection,
}

var (
	ErrInvalidEvent     = errors.New("invalid business event")
	ErrCorrectionReason = errors.New("correction reason is required")
	ErrCorrectionTarget = errors.New("correction target is required")
)

// Actor identifies the account or system process responsible for a transition.
type Actor struct {
	Role ActorRole
	ID   string
}

// Event is an append-only transition with reconstructable before and after state.
type Event struct {
	ID              string
	Type            EventType
	AggregateType   string
	AggregateID     string
	AssignmentID    string
	Actor           Actor
	EventAt         time.Time
	RelatedIDs      map[string]string
	PriorState      map[string]any
	NewState        map[string]any
	Reason          string
	InternalNote    string
	CorrectsEventID string
}

// EventInput contains the fields shared by all event constructors.
type EventInput struct {
	AggregateType string
	AggregateID   string
	AssignmentID  string
	Actor         Actor
	EventAt       time.Time
	RelatedIDs    map[string]string
	PriorState    map[string]any
	NewState      map[string]any
	Reason        string
	InternalNote  string
}

// Validate checks stable identifiers, actor attribution, UTC instant, and event vocabulary.
func (e Event) Validate() error {
	for _, validate := range []func() error{e.validateIdentity, e.validateInstant, e.validateActor, e.validateCorrection} {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func (e Event) validateIdentity() error {
	if !slices.Contains(eventTypes, e.Type) || strings.TrimSpace(e.AggregateType) == "" || strings.TrimSpace(e.AggregateID) == "" || strings.TrimSpace(e.AssignmentID) == "" {
		return ErrInvalidEvent
	}
	return nil
}

func (e Event) validateInstant() error {
	if e.EventAt.IsZero() || e.EventAt.Location() != time.UTC {
		return fmt.Errorf("%w: event_at must be UTC", ErrInvalidEvent)
	}
	return nil
}

func (e Event) validateActor() error {
	if e.Actor.Role != TeacherActor && e.Actor.Role != LearnerActor && e.Actor.Role != SystemActor {
		return fmt.Errorf("%w: actor role is invalid", ErrInvalidEvent)
	}
	if e.Actor.Role != SystemActor && strings.TrimSpace(e.Actor.ID) == "" {
		return fmt.Errorf("%w: actor id is required", ErrInvalidEvent)
	}
	return nil
}

func (e Event) validateCorrection() error {
	if e.Type == AdministrativeCorrection && strings.TrimSpace(e.Reason) == "" {
		return ErrCorrectionReason
	}
	if e.CorrectsEventID != "" && e.Type != AdministrativeCorrection {
		return fmt.Errorf("%w: only corrections can reference an event", ErrInvalidEvent)
	}
	return nil
}
