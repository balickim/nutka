// This file loads availability state, builds planner inputs, and commits one resolved mutation atomically.
package schedulingapi

import (
	"errors"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/availabilityimpact"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

var (
	errStalePreview      = errors.New("availability preview is stale")
	errIncompletePreview = errors.New("availability preview resolutions are incomplete")
	errInvalidResolution = errors.New("availability preview resolution is invalid")
)

// AvailabilityMutation is the complete proposed change to one rule or exception.
type AvailabilityMutation struct {
	Operation availabilityimpact.Operation
	Target    availabilityimpact.Target
	ID        string
	Rule      *RuleMutation
	Exception *ExceptionMutation
}

// RuleMutation contains recurring local wall-clock fields. Nil fields retain existing values on update.
type RuleMutation struct {
	Weekday   *int
	StartTime *string
	EndTime   *string
	Enabled   *bool
}

// ExceptionMutation contains concrete UTC fields. Nil fields retain existing values on update.
type ExceptionMutation struct {
	StartAt *string
	EndAt   *string
	Kind    *string
	Note    *string
	Enabled *bool
}

// Resolution is one teacher decision for one near-term lesson conflict.
type Resolution struct {
	LessonID         string
	Action           availabilityimpact.ResolutionAction
	ReplacementStart time.Time
}

// CommitResult contains the persisted availability records and planner effects.
type CommitResult struct {
	Preview   availabilityimpact.Preview
	Rule      *core.Record
	Exception *core.Record
}

// LessonCommands is the command boundary for plan-aware teacher lifecycle effects.
// Production callers can inject the full lifecycle service without changing availability orchestration.
type LessonCommands interface {
	Cancel(app core.App, lesson *core.Record, actorID string, now time.Time) error
	Reschedule(app core.App, lesson *core.Record, replacement time.Time, availability []scheduling.Interval, actorID string, now time.Time) error
}

// AvailabilityService owns preview and resolved-commit orchestration for one PocketBase app.
type AvailabilityService struct {
	Planner  availabilityimpact.Planner
	Policy   businesspolicy.Policy
	Clock    func() time.Time
	Commands LessonCommands
}

// NewAvailabilityService creates a service using the authoritative policy and default lesson commands.
func NewAvailabilityService(policy businesspolicy.Policy, clock func() time.Time, commands LessonCommands) (*AvailabilityService, error) {
	planner, err := availabilityimpact.NewPlanner(policy)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		clock = time.Now
	}
	if commands == nil {
		commands = pocketBaseLessonCommands{}
	}
	return &AvailabilityService{Planner: planner, Policy: policy, Clock: clock, Commands: commands}, nil
}

// Preview computes normalized availability and all near or distant effects without persistence.
func (s *AvailabilityService) Preview(app core.App, teacherID string, mutation AvailabilityMutation) (availabilityimpact.Preview, error) {
	now := s.Clock().UTC()
	state, err := loadAvailabilityState(app, teacherID, mutation, now, s.Policy)
	if err != nil {
		return availabilityimpact.Preview{}, err
	}
	return s.Planner.Preview(state.input)
}

// Commit re-plans against records re-read inside one transaction before saving any record.
func (s *AvailabilityService) Commit(app core.App, teacherID string, mutation AvailabilityMutation, version string, resolutions []Resolution) (CommitResult, error) {
	var result CommitResult
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err := app.RunInTransaction(func(tx core.App) error {
		now := s.Clock().UTC()
		state, err := loadAvailabilityState(tx, teacherID, mutation, now, s.Policy)
		if err != nil {
			return err
		}
		preview, err := s.Planner.Preview(state.input)
		if err != nil {
			return err
		}
		if version == "" || version != preview.PreviewVersion {
			return errStalePreview
		}
		if err := validateResolutions(preview.NearTermConflicts, resolutions); err != nil {
			return err
		}
		if err := applyResolutions(tx, state.lessons, resolutions, s.Commands, teacherID, now, preview.Proposal.EffectiveAvailability, s.Policy); err != nil {
			return err
		}
		if err := applyMutation(tx, teacherID, mutation, &result); err != nil {
			return err
		}
		if err := applyDistantEffects(tx, preview.DistantEffects, now); err != nil {
			return err
		}
		if err := saveAvailabilityEvents(tx, teacherID, mutation, result, now); err != nil {
			return err
		}
		result.Preview = preview
		return nil
	})
	if err != nil {
		return CommitResult{}, err
	}
	return result, nil
}

type availabilityState struct {
	input   availabilityimpact.Input
	lessons map[string]*core.Record
}

func loadAvailabilityState(app core.App, teacherID string, mutation AvailabilityMutation, now time.Time, policy businesspolicy.Policy) (availabilityState, error) {
	if teacherID == "" {
		return availabilityState{}, errForbidden
	}
	if !validMutation(mutation) {
		return availabilityState{}, errInvalid
	}
	teacher, err := app.FindRecordById("teachers", teacherID)
	if err != nil {
		return availabilityState{}, errForbidden
	}
	ruleRows, exceptionRows, err := availabilitySourceRows(app, teacherID, mutation)
	if err != nil {
		return availabilityState{}, err
	}
	rules, err := availabilityRulesAfterMutation(ruleRows, mutation)
	if err != nil {
		return availabilityState{}, err
	}
	exceptions, err := availabilityExceptionsAfterMutation(exceptionRows, mutation)
	if err != nil {
		return availabilityState{}, err
	}
	lessons, err := recordsByField(app, schedulingstore.LessonsCollectionName, "teacher", teacherID)
	if err != nil {
		return availabilityState{}, err
	}
	horizon := now.Add(policy.BookingHorizon)
	near, distant, lessonMap, latest := plannerRecords(lessons, horizon)
	if latest.Before(horizon) {
		latest = horizon
	}
	available, err := scheduling.EffectiveAvailability(now, latest, teacher.GetString(schedulingstore.TeacherTimezoneField), rules, exceptions)
	if err != nil {
		return availabilityState{}, errInvalid
	}
	return availabilityState{input: availabilityimpact.Input{Now: now, Proposal: availabilityimpact.Proposal{Operation: mutation.Operation, Target: mutation.Target, EffectiveAvailability: available}, NearTermLessons: near, DistantOccurrences: distant, StateVersion: sourceVersion(ruleRows, exceptionRows)}, lessons: lessonMap}, nil
}

func availabilitySourceRows(app core.App, teacherID string, mutation AvailabilityMutation) ([]*core.Record, []*core.Record, error) {
	rules, err := recordsByField(app, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacherID)
	if err != nil {
		return nil, nil, err
	}
	exceptions, err := recordsByField(app, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacherID)
	if err != nil {
		return nil, nil, err
	}
	if err := verifyMutationOwner(app, teacherID, mutation); err != nil {
		return nil, nil, err
	}
	return rules, exceptions, nil
}

func verifyMutationOwner(app core.App, teacherID string, mutation AvailabilityMutation) error {
	if mutation.Operation == availabilityimpact.Create {
		return nil
	}
	collection := schedulingstore.AvailabilityRulesCollectionName
	if mutation.Target == availabilityimpact.Exception {
		collection = schedulingstore.AvailabilityExceptionsCollectionName
	}
	_, err := ownedRecord(app, collection, mutation.ID, "teacher", teacherID)
	return err
}

func validMutation(mutation AvailabilityMutation) bool {
	if !validMutationTarget(mutation) || !validAvailabilityOperation(mutation.Operation) {
		return false
	}
	if !validMutationPayloadType(mutation) {
		return false
	}
	if mutation.Operation == availabilityimpact.Create {
		return validCreateMutation(mutation)
	}
	if mutation.ID == "" {
		return false
	}
	if mutation.Operation == availabilityimpact.Update {
		return validUpdateMutation(mutation)
	}
	return mutation.Rule == nil && mutation.Exception == nil
}

func validMutationTarget(mutation AvailabilityMutation) bool {
	return mutation.Target == availabilityimpact.RecurringRule || mutation.Target == availabilityimpact.Exception
}

func validMutationPayloadType(mutation AvailabilityMutation) bool {
	if mutation.Target == availabilityimpact.RecurringRule {
		return mutation.Exception == nil
	}
	return mutation.Rule == nil
}

func validCreateMutation(mutation AvailabilityMutation) bool {
	if mutation.ID != "" {
		return false
	}
	if mutation.Target == availabilityimpact.RecurringRule {
		return mutation.Rule != nil
	}
	return mutation.Exception != nil
}

func validUpdateMutation(mutation AvailabilityMutation) bool {
	if mutation.Target == availabilityimpact.RecurringRule {
		return hasRuleFields(mutation.Rule)
	}
	return hasExceptionFields(mutation.Exception)
}

func hasRuleFields(value *RuleMutation) bool {
	return value != nil && (value.Weekday != nil || value.StartTime != nil || value.EndTime != nil || value.Enabled != nil)
}

func hasExceptionFields(value *ExceptionMutation) bool {
	return value != nil && (value.StartAt != nil || value.EndAt != nil || value.Kind != nil || value.Note != nil || value.Enabled != nil)
}

func validAvailabilityOperation(operation availabilityimpact.Operation) bool {
	switch operation {
	case availabilityimpact.Create, availabilityimpact.Update, availabilityimpact.Enable, availabilityimpact.Disable, availabilityimpact.Delete:
		return true
	default:
		return false
	}
}
