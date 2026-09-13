// This file validates booking starts and derives grid-aligned slots within the rolling horizon.
package scheduling

import "time"

// GenerateSlots returns UTC, grid-aligned lesson starts in the rolling horizon.
func GenerateSlots(now time.Time, timezone string, rules []WeekdayRule, exceptions []AvailabilityException, teacherID, learnerID string, lessons []Lesson, duration time.Duration) ([]Slot, error) {
	if err := ValidateLessonDuration(duration); err != nil {
		return nil, err
	}
	now = now.UTC()
	if now.IsZero() {
		return nil, ErrHorizon
	}
	horizon := HorizonEnd(now)
	available, err := EffectiveAvailability(now, horizon, timezone, rules, exceptions)
	if err != nil {
		return nil, err
	}
	first := nextGridBoundary(now)
	slots := make([]Slot, 0)
	for _, window := range available {
		candidate := first
		if candidate.Before(window.Start) {
			candidate = ceilGridBoundary(window.Start)
		}
		for candidate.Before(window.End) {
			end := candidate.Add(duration)
			if end.After(window.End) || candidate.After(horizon) {
				break
			}
			interval := Interval{Start: candidate, End: end}
			if len(DetectConflicts(interval, teacherID, learnerID, lessons)) == 0 {
				slots = append(slots, Slot{Interval: interval, Protected: interval.Protected()})
			}
			candidate = candidate.Add(SlotDuration)
		}
	}
	return slots, nil
}

func ValidateBookingStart(start time.Time, now time.Time, duration time.Duration) (Interval, error) {
	if err := ValidateLessonDuration(duration); err != nil {
		return Interval{}, err
	}
	start = start.UTC()
	now = now.UTC()
	if !IsOnSlotGrid(start) {
		return Interval{}, ErrInvalidGrid
	}
	if !start.After(now) || start.After(HorizonEnd(now)) {
		return Interval{}, ErrHorizon
	}
	end := start.Add(duration)
	if end.After(HorizonEnd(now)) {
		return Interval{}, ErrHorizon
	}
	return NewInterval(start, end)
}

func nextGridBoundary(value time.Time) time.Time {
	value = value.UTC()
	boundary := value.Truncate(SlotDuration)
	return boundary.Add(SlotDuration)
}

func ceilGridBoundary(value time.Time) time.Time {
	value = value.UTC()
	boundary := value.Truncate(SlotDuration)
	if boundary.Equal(value) {
		return boundary
	}
	return boundary.Add(SlotDuration)
}
