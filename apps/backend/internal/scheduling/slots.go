// This file validates booking starts and derives grid-aligned slots within the rolling horizon.
package scheduling

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
)

func GenerateSlotsWithPolicy(now time.Time, timezone string, rules []WeekdayRule, exceptions []AvailabilityException, teacherID, learnerID string, lessons []Lesson, policy businesspolicy.Policy) ([]Slot, error) {
	return GenerateSlots(now, timezone, rules, exceptions, teacherID, learnerID, lessons, 0, IntervalPolicyFromBusinessPolicy(policy))
}

// GenerateSlots returns UTC, grid-aligned lesson starts in the configured rolling horizon.
func GenerateSlots(now time.Time, timezone string, rules []WeekdayRule, exceptions []AvailabilityException, teacherID, learnerID string, lessons []Lesson, duration time.Duration, options ...IntervalPolicy) ([]Slot, error) {
	policy, err := resolveIntervalPolicy(options...)
	if err != nil {
		return nil, err
	}
	duration, err = requestedDuration(duration, policy, len(options) > 0)
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	if now.IsZero() {
		return nil, ErrHorizon
	}
	horizon := policy.HorizonEnd(now)
	available, err := EffectiveAvailability(now, horizon, timezone, rules, exceptions)
	if err != nil {
		return nil, err
	}
	first := nextGridBoundaryWithGrid(now, policy.Grid)
	minimum := now.Add(policy.BookingMinimum)
	if first.Before(minimum) {
		first = ceilGridBoundaryWithGrid(minimum, policy.Grid)
	}
	slots := make([]Slot, 0)
	for _, window := range available {
		slots = append(slots, slotsInWindow(window, first, horizon, duration, policy, teacherID, learnerID, lessons)...)
	}
	return slots, nil
}

func ValidateBookingStart(start time.Time, now time.Time, duration time.Duration, options ...IntervalPolicy) (Interval, error) {
	policy, err := resolveIntervalPolicy(options...)
	if err != nil {
		return Interval{}, err
	}
	duration, err = requestedDuration(duration, policy, len(options) > 0)
	if err != nil {
		return Interval{}, err
	}
	start = start.UTC()
	now = now.UTC()
	if err := validateBookingBounds(start, now, policy); err != nil {
		return Interval{}, err
	}
	end := start.Add(duration)
	return NewInterval(start, end)
}

func ValidateBookingStartWithPolicy(start, now time.Time, policy businesspolicy.Policy) (Interval, error) {
	return ValidateBookingStart(start, now, 0, IntervalPolicyFromBusinessPolicy(policy))
}

func nextGridBoundary(value time.Time) time.Time {
	return nextGridBoundaryWithGrid(value, SlotDuration)
}

func nextGridBoundaryWithGrid(value time.Time, grid time.Duration) time.Time {
	value = value.UTC()
	boundary := value.Truncate(grid)
	return boundary.Add(grid)
}

func ceilGridBoundary(value time.Time) time.Time {
	return ceilGridBoundaryWithGrid(value, SlotDuration)
}

func ceilGridBoundaryWithGrid(value time.Time, grid time.Duration) time.Time {
	value = value.UTC()
	boundary := value.Truncate(grid)
	if boundary.Equal(value) {
		return boundary
	}
	return boundary.Add(grid)
}

func validateDuration(duration, grid time.Duration) error {
	if duration <= 0 || grid <= 0 || duration%grid != 0 {
		return ErrInvalidDuration
	}
	return nil
}

func requestedDuration(duration time.Duration, policy IntervalPolicy, policyInjected bool) (time.Duration, error) {
	if policyInjected && duration != 0 && duration != policy.Duration {
		return 0, ErrInvalidDuration
	}
	if duration == 0 {
		duration = policy.Duration
	}
	if err := validateDuration(duration, policy.Grid); err != nil {
		return 0, err
	}
	return duration, nil
}

func validateBookingBounds(start, now time.Time, policy IntervalPolicy) error {
	if !isOnGrid(start, policy.Grid) {
		return ErrInvalidGrid
	}
	if !start.After(now) || start.Before(now.Add(policy.BookingMinimum)) || start.After(policy.HorizonEnd(now)) {
		return ErrHorizon
	}
	return nil
}

func slotsInWindow(window Interval, first, horizon time.Time, duration time.Duration, policy IntervalPolicy, teacherID, learnerID string, lessons []Lesson) []Slot {
	candidate := first
	windowFirst := ceilGridBoundaryWithGrid(window.Start, policy.Grid)
	if candidate.Before(windowFirst) {
		candidate = windowFirst
	}
	slots := make([]Slot, 0)
	for candidate.Before(window.End) {
		end := candidate.Add(duration)
		if end.After(window.End) || candidate.After(horizon) {
			break
		}
		interval := Interval{Start: candidate, End: end}
		if len(DetectConflicts(interval, teacherID, learnerID, lessons, policy)) == 0 {
			slots = append(slots, Slot{Interval: interval, Protected: interval.Protected(policy.Buffer)})
		}
		candidate = candidate.Add(policy.Grid)
	}
	return slots
}
