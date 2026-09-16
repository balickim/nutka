// Package lesson defines plan-aware lifecycle commands without persistence or transport dependencies.
package lesson

import (
	"errors"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

var (
	ErrInvalidCommand       = errors.New("lesson lifecycle command is invalid")
	ErrUnauthorized         = errors.New("lesson lifecycle actor is unauthorized")
	ErrStarted              = errors.New("started lesson cannot change")
	ErrCancelled            = errors.New("cancelled lesson cannot change")
	ErrReplacementRequired  = errors.New("reschedule requires a replacement")
	ErrOutcomeUnavailable   = errors.New("lesson outcome is unavailable")
	ErrOutcomeAlreadyStored = errors.New("lesson outcome was already recorded")
	ErrCorrectionTarget     = errors.New("lesson correction target is invalid")
)

// Lesson is the lifecycle projection required by one command. Interval and plan
// references retain the stable lesson identity while projections remain mutable.
type Lesson struct {
	ID, AssignmentID, TeacherID, LearnerID string
	Plan                                   commercial.PlanType
	Interval                               scheduling.Interval
	OriginalLocalDate                      string
	ScheduleState                          domain.ScheduleState
	Outcome                                domain.Outcome
	Settlement                             ledger.SettlementState
	UnitPriceMinor                         int64
	Currency                               string
}

// Command contains all state loaded by an application adapter in one transaction.
type Command struct {
	Lesson             Lesson
	Actor              history.Actor
	Now                time.Time
	ReplacementStart   time.Time
	Availability       []scheduling.Interval
	ParticipantLessons []regularcontract.ParticipantLesson
	Package            *commercial.Package
	Contract           *regularcontract.RegularContract
	Charge             *ledger.Charge
	Credits            []ledger.Credit
	EntitlementEventID string
}

// Decision is the complete next projection. The adapter persists every returned
// aggregate and event in the transaction that loaded Command.
type Decision struct {
	Lesson     Lesson
	Package    *commercial.Package
	Contract   *regularcontract.RegularContract
	Charge     *ledger.Charge
	Credits    []ledger.Credit
	PlanEffect string
	Events     []history.Event
}

// RescheduleLesson moves one identity and never emits a cancellation effect.
func RescheduleLesson(command Command) (Decision, error) {
	if err := command.validate(); err != nil {
		return Decision{}, err
	}
	if command.ReplacementStart.IsZero() {
		return Decision{}, ErrReplacementRequired
	}
	if err := command.futureScheduled(); err != nil {
		return Decision{}, err
	}
	if command.Actor.Role == history.LearnerActor && command.Lesson.Interval.Start.Sub(command.Now) < learnerCutoff(command) {
		return Decision{}, commercial.ErrLearnerChangeTooLate
	}
	replacement, err := replacementInterval(command)
	if err != nil {
		return Decision{}, err
	}
	if err := validateReplacementTiming(command, replacement.Start); err != nil {
		return Decision{}, err
	}
	prior := command.Lesson.Interval
	if err := applyReschedule(&command, replacement); err != nil {
		return Decision{}, err
	}
	command.Lesson.Interval = replacement
	command.Lesson.Outcome = domain.AwaitingOutcome
	event, err := lifecycleEvent(history.LessonRescheduled, command, map[string]any{
		"start_at": prior.Start.UTC().Format(time.RFC3339), "end_at": prior.End.UTC().Format(time.RFC3339),
	}, map[string]any{
		"start_at": replacement.Start.UTC().Format(time.RFC3339), "end_at": replacement.End.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return Decision{}, err
	}
	return decision(command, "lesson_rescheduled", event), nil
}

// CancelLesson releases one lesson without selecting a replacement.
func CancelLesson(command Command) (Decision, error) {
	if err := command.validate(); err != nil {
		return Decision{}, err
	}
	if err := command.futureScheduled(); err != nil {
		return Decision{}, err
	}
	timely := command.Actor.Role != history.LearnerActor || command.Lesson.Interval.Start.Sub(command.Now) >= learnerCutoff(command)
	effect, err := applyCancellation(&command, timely)
	if err != nil {
		return Decision{}, err
	}
	command.Lesson.ScheduleState = domain.CancelledState
	command.Lesson.Outcome = ""
	event, err := lifecycleEvent(history.LessonCancelled, command,
		map[string]any{"schedule_state": string(domain.ScheduledState)},
		map[string]any{"schedule_state": string(domain.CancelledState), "cutoff": cutoffClass(timely), "plan_effect": effect})
	if err != nil {
		return Decision{}, err
	}
	return decision(command, effect, event), nil
}

// RecordOutcome stores an explicit teacher outcome after the scheduled interval.
func RecordOutcome(command Command, outcome domain.Outcome) (Decision, error) {
	if err := command.validate(); err != nil {
		return Decision{}, err
	}
	if err := validateOutcome(command, outcome); err != nil {
		return Decision{}, err
	}
	command.Lesson.Outcome = outcome
	if err := applyOutcome(&command, outcome); err != nil {
		return Decision{}, err
	}
	event, err := lifecycleEvent(history.LessonOutcomeRecorded, command,
		map[string]any{"outcome": string(domain.AwaitingOutcome)}, map[string]any{"outcome": string(outcome)})
	if err != nil {
		return Decision{}, err
	}
	return decision(command, "outcome_recorded", event), nil
}

func validateOutcome(command Command, outcome domain.Outcome) error {
	if command.Actor.Role != history.TeacherActor || command.Actor.ID != command.Lesson.TeacherID {
		return ErrUnauthorized
	}
	if command.Lesson.ScheduleState != domain.ScheduledState || command.Lesson.Interval.End.After(command.Now) {
		return ErrOutcomeUnavailable
	}
	if command.Lesson.Outcome != "" && command.Lesson.Outcome != domain.AwaitingOutcome {
		return ErrOutcomeAlreadyStored
	}
	if outcome != domain.Completed && outcome != domain.LearnerNoShow {
		return ErrInvalidCommand
	}
	return nil
}

// Correct reverses one package or contract lifecycle effect through a named,
// reasoned compensating event. It does not edit the original event.
func Correct(command Command, correctedEventID, reason string) (Decision, error) {
	if err := command.validate(); err != nil {
		return Decision{}, err
	}
	if !validCorrectionCommand(command, correctedEventID, reason) {
		return Decision{}, ErrCorrectionTarget
	}
	if err := correctEntitlement(&command, correctedEventID, reason); err != nil {
		return Decision{}, err
	}
	event, err := history.NewEvent(history.AdministrativeCorrection, history.EventInput{AggregateType: "lesson", AggregateID: command.Lesson.ID, AssignmentID: command.Lesson.AssignmentID, Actor: command.Actor, EventAt: command.Now.UTC(), RelatedIDs: map[string]string{"plan": string(command.Lesson.Plan)}, NewState: map[string]any{"corrects_event": correctedEventID, "reason": strings.TrimSpace(reason)}, Reason: strings.TrimSpace(reason)})
	if err != nil {
		return Decision{}, err
	}
	correction := decision(command, "entitlement_corrected", event)
	correction.Events[0].CorrectsEventID = correctedEventID
	return correction, nil
}

func validCorrectionCommand(command Command, eventID, reason string) bool {
	return command.Actor.Role == history.TeacherActor && command.Actor.ID == command.Lesson.TeacherID && strings.TrimSpace(eventID) != "" && strings.TrimSpace(reason) != ""
}

func correctEntitlement(command *Command, eventID, reason string) error {
	switch command.Lesson.Plan {
	case commercial.PackagePlan:
		if command.Package == nil {
			return ErrCorrectionTarget
		}
		return correctPackage(command.Package, command.Lesson.ID, eventID, reason, command.Now)
	case commercial.RegularContract:
		if command.Contract == nil {
			return ErrCorrectionTarget
		}
		return correctContractEntitlement(command, eventID, reason)
	default:
		return ErrCorrectionTarget
	}
}

func correctContractEntitlement(command *Command, eventID, reason string) error {
	contractEventID := eventID
	if command.EntitlementEventID != "" {
		contractEventID = command.EntitlementEventID
	}
	return command.Contract.CorrectEvent(contractEventID, reason, regularcontract.Actor{Role: regularcontract.Teacher, ID: command.Actor.ID}, command.Now)
}
