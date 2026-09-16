// Package availabilityimpact plans availability effects without persistence, HTTP, or clocks.
package availabilityimpact

import (
	"errors"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

var (
	ErrInvalidInput        = errors.New("availability impact input is invalid")
	ErrInvalidProposal     = errors.New("availability proposal is invalid")
	ErrInvalidAvailability = errors.New("availability interval is invalid")
	ErrInvalidLesson       = errors.New("near-term lesson is invalid")
	ErrInvalidOccurrence   = errors.New("contract occurrence is invalid")
)

// Operation identifies the availability mutation that produced a proposal.
type Operation string

const (
	Create  Operation = "create"
	Update  Operation = "update"
	Enable  Operation = "enable"
	Disable Operation = "disable"
	Delete  Operation = "delete"
)

// Target identifies whether a proposal changes a recurring rule or exception.
type Target string

const (
	RecurringRule Target = "recurring_rule"
	Exception     Target = "exception"
)

// Proposal contains the normalized mutation and effective availability after it.
// EffectiveAvailability is UTC and is supplied by the availability application service.
type Proposal struct {
	Operation             Operation             `json:"operation"`
	Target                Target                `json:"target,omitempty"`
	EffectiveAvailability []scheduling.Interval `json:"effective_availability,omitempty"`
}

// Lesson is a scheduled lesson that may require an explicit teacher resolution.
type Lesson struct {
	ID       string
	Plan     domain.PlanType
	Interval scheduling.Interval
	State    domain.ScheduleState
}

// ContractOccurrence is a materialized weekly contract lesson.
type ContractOccurrence struct {
	ID             string
	Interval       scheduling.Interval
	State          domain.ScheduleState
	OmissionReason string
}

// Input supplies the current relevant scheduling state to the planner.
type Input struct {
	Now                time.Time
	Proposal           Proposal
	NearTermLessons    []Lesson
	DistantOccurrences []ContractOccurrence
	StateVersion       string
}

// ResolutionAction is the required teacher decision for a near-term conflict.
type ResolutionAction string

const (
	Cancel     ResolutionAction = "cancel"
	Reschedule ResolutionAction = "reschedule"
)

// DistantAction is an automatic effect for a later contract occurrence.
type DistantAction string

const (
	Omit    DistantAction = "omit"
	Restore DistantAction = "restore"
)

// NearTermConflict describes one lesson that blocks the proposed availability.
type NearTermConflict struct {
	LessonID           string             `json:"lesson"`
	Plan               domain.PlanType    `json:"plan"`
	StartAt            time.Time          `json:"start_at"`
	AllowedResolutions []ResolutionAction `json:"allowed_resolutions"`
}

// DistantEffect describes an automatic omission or restoration beyond the horizon.
type DistantEffect struct {
	OccurrenceID string        `json:"occurrence"`
	Effect       DistantAction `json:"effect"`
	StartAt      time.Time     `json:"start_at"`
}

// Preview is the complete pure result consumed by preview and resolved-commit services.
type Preview struct {
	PreviewVersion    string             `json:"preview_version"`
	Proposal          Proposal           `json:"proposal"`
	NearTermConflicts []NearTermConflict `json:"near_term_conflicts"`
	DistantEffects    []DistantEffect    `json:"distant_effects"`
}

// Planner compares effective availability with near and later scheduled obligations.
type Planner struct {
	policy        scheduling.IntervalPolicy
	policyVersion string
}

// NewPlanner validates and creates a planner with the authoritative business policy.
func NewPlanner(policy businesspolicy.Policy) (Planner, error) {
	if err := policy.Validate(); err != nil {
		return Planner{}, err
	}
	return Planner{policy: scheduling.IntervalPolicyFromBusinessPolicy(policy), policyVersion: policy.Version}, nil
}

// Preview builds a deterministic impact preview. It does not mutate input slices.
func (p Planner) Preview(input Input) (Preview, error) {
	if input.Now.IsZero() {
		return Preview{}, ErrInvalidInput
	}
	if err := p.policy.Validate(); err != nil {
		return Preview{}, err
	}
	policy := p.policy
	proposal, availability, err := normalizeProposal(input.Proposal)
	if err != nil {
		return Preview{}, err
	}
	now := input.Now.UTC()
	horizon := now.Add(policy.Horizon)
	conflicts, err := nearTermConflicts(now, horizon, availability, input.NearTermLessons)
	if err != nil {
		return Preview{}, err
	}
	effects, err := distantEffects(horizon, availability, input.DistantOccurrences)
	if err != nil {
		return Preview{}, err
	}
	preview := Preview{Proposal: proposal, NearTermConflicts: conflicts, DistantEffects: effects}
	preview.PreviewVersion = stateVersion(input, policy, p.policyVersion, proposal, availability, conflicts, effects)
	return preview, nil
}

func normalizeProposal(value Proposal) (Proposal, []scheduling.Interval, error) {
	if !validOperation(value.Operation) {
		return Proposal{}, nil, ErrInvalidProposal
	}
	if value.Target != RecurringRule && value.Target != Exception {
		return Proposal{}, nil, ErrInvalidProposal
	}
	availability := make([]scheduling.Interval, len(value.EffectiveAvailability))
	for index, interval := range value.EffectiveAvailability {
		normalized, err := scheduling.NewInterval(interval.Start, interval.End)
		if err != nil {
			return Proposal{}, nil, ErrInvalidAvailability
		}
		availability[index] = normalized
	}
	sort.Slice(availability, func(left, right int) bool {
		if availability[left].Start.Equal(availability[right].Start) {
			return availability[left].End.Before(availability[right].End)
		}
		return availability[left].Start.Before(availability[right].Start)
	})
	proposal := value
	proposal.EffectiveAvailability = append([]scheduling.Interval(nil), availability...)
	return proposal, availability, nil
}

func nearTermConflicts(now, horizon time.Time, availability []scheduling.Interval, lessons []Lesson) ([]NearTermConflict, error) {
	values := make([]Lesson, len(lessons))
	copy(values, lessons)
	sort.SliceStable(values, func(left, right int) bool {
		if values[left].ID == values[right].ID {
			return values[left].Interval.Start.Before(values[right].Interval.Start)
		}
		return values[left].ID < values[right].ID
	})
	conflicts := make([]NearTermConflict, 0)
	for _, lesson := range values {
		interval, err := normalizeInterval(lesson.Interval)
		if err != nil || !validLesson(lesson) {
			return nil, ErrInvalidLesson
		}
		if !nearTermAffected(now, horizon, availability, interval, lesson.State) {
			continue
		}
		conflicts = append(conflicts, NearTermConflict{LessonID: lesson.ID, Plan: lesson.Plan, StartAt: interval.Start, AllowedResolutions: []ResolutionAction{Cancel, Reschedule}})
	}
	return conflicts, nil
}

func validLesson(lesson Lesson) bool {
	return lesson.ID != "" && lesson.Plan.Valid() && (lesson.State == "" || lesson.State.Valid())
}

func nearTermAffected(now, horizon time.Time, availability []scheduling.Interval, interval scheduling.Interval, state domain.ScheduleState) bool {
	return isScheduled(state) && interval.Start.After(now) && !interval.Start.After(horizon) && !contains(availability, interval)
}

func distantEffects(horizon time.Time, availability []scheduling.Interval, occurrences []ContractOccurrence) ([]DistantEffect, error) {
	values := make([]ContractOccurrence, len(occurrences))
	copy(values, occurrences)
	sort.SliceStable(values, func(left, right int) bool {
		if values[left].ID == values[right].ID {
			return values[left].Interval.Start.Before(values[right].Interval.Start)
		}
		return values[left].ID < values[right].ID
	})
	effects := make([]DistantEffect, 0)
	for _, occurrence := range values {
		interval, err := normalizeInterval(occurrence.Interval)
		if err != nil || !validOccurrence(occurrence) {
			return nil, ErrInvalidOccurrence
		}
		if !interval.Start.After(horizon) {
			continue
		}
		if effect, ok := occurrenceEffect(occurrence, interval, availability); ok {
			effects = append(effects, DistantEffect{OccurrenceID: occurrence.ID, Effect: effect, StartAt: interval.Start})
		}
	}
	return effects, nil
}

func validOccurrence(occurrence ContractOccurrence) bool {
	return occurrence.ID != "" && (occurrence.State == "" || occurrence.State.Valid())
}

func occurrenceEffect(occurrence ContractOccurrence, interval scheduling.Interval, availability []scheduling.Interval) (DistantAction, bool) {
	if isScheduled(occurrence.State) && !contains(availability, interval) {
		return Omit, true
	}
	if occurrence.State == domain.OmittedState && plannedOmission(occurrence.OmissionReason) && contains(availability, interval) {
		return Restore, true
	}
	return "", false
}

func normalizeInterval(value scheduling.Interval) (scheduling.Interval, error) {
	return scheduling.NewInterval(value.Start, value.End)
}

func contains(availability []scheduling.Interval, interval scheduling.Interval) bool {
	for _, window := range availability {
		normalized, err := normalizeInterval(window)
		if err == nil && normalized.Contains(interval) {
			return true
		}
	}
	return false
}

func isScheduled(state domain.ScheduleState) bool {
	return state == "" || state == domain.ScheduledState
}

func plannedOmission(reason string) bool {
	return reason == "planned_unavailability"
}

func validOperation(value Operation) bool {
	return value == Create || value == Update || value == Enable || value == Disable || value == Delete
}
