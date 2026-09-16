// Package scheduling contains pure time and participant rules for lesson booking.
// It has no persistence, HTTP, or framework dependencies.
package scheduling

import (
	"errors"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
)

const (
	DefaultTimezone       = "Europe/Warsaw"
	SlotDuration          = businesspolicy.StartGridMinutes * time.Minute
	ProtectedBuffer       = businesspolicy.ParticipantBufferMinutes * time.Minute
	DefaultLessonDuration = businesspolicy.LessonDurationMinutes * time.Minute
	HorizonDuration       = businesspolicy.BookingHorizonDays * 24 * time.Hour
)

var (
	ErrInvalidInterval       = errors.New("interval must have a UTC start before its end")
	ErrInvalidDuration       = errors.New("duration must be positive and a multiple of 15 minutes")
	ErrInvalidGrid           = errors.New("start must be aligned to a 15-minute UTC boundary")
	ErrInvalidTimezone       = errors.New("timezone must be a valid IANA identifier")
	ErrInvalidRule           = errors.New("availability rule is invalid")
	ErrInvalidException      = errors.New("availability exception is invalid")
	ErrInvalidExceptionKind  = errors.New("availability exception kind is invalid")
	ErrHorizon               = errors.New("start is outside the rolling 14-day horizon")
	ErrConflict              = errors.New("participant has a protected interval conflict")
	ErrLearnerDurationChange = errors.New("learners cannot override lesson duration")
)

// Interval is a half-open UTC interval [Start, End).
type Interval struct {
	Start time.Time
	End   time.Time
}

func NewInterval(start, end time.Time) (Interval, error) {
	interval := Interval{Start: start.UTC(), End: end.UTC()}
	if !interval.Valid() {
		return Interval{}, ErrInvalidInterval
	}
	return interval, nil
}

func (i Interval) Valid() bool {
	return !i.Start.IsZero() && !i.End.IsZero() && i.Start.Before(i.End) && i.Start.Location() == time.UTC && i.End.Location() == time.UTC
}

func (i Interval) Duration() time.Duration { return i.End.Sub(i.Start) }

func (i Interval) Contains(other Interval) bool {
	return i.Valid() && other.Valid() && !other.Start.Before(i.Start) && !other.End.After(i.End)
}

func (i Interval) Intersects(other Interval) bool {
	return i.Valid() && other.Valid() && i.Start.Before(other.End) && other.Start.Before(i.End)
}

// Protected expands an interval by the configured buffer for participant conflict checks.
func (i Interval) Protected(buffer ...time.Duration) Interval {
	if !i.Valid() {
		return Interval{}
	}
	amount := ProtectedBuffer
	if len(buffer) > 0 {
		amount = buffer[0]
	}
	if amount < 0 {
		return Interval{}
	}
	return Interval{Start: i.Start.Add(-amount), End: i.End.Add(amount)}
}

// WeekdayRule stores a recurring local wall-clock interval. Times are minutes after midnight.
type WeekdayRule struct {
	Weekday     time.Weekday
	StartMinute int
	EndMinute   int
	Enabled     bool
}

// RecurringRule is retained as an alias for callers that use the specification name.
type RecurringRule = WeekdayRule

func NewWeekdayRule(day time.Weekday, startMinute, endMinute int) WeekdayRule {
	return WeekdayRule{Weekday: day, StartMinute: startMinute, EndMinute: endMinute, Enabled: true}
}

func (r WeekdayRule) Valid() bool {
	return r.Enabled && r.Weekday >= time.Sunday && r.Weekday <= time.Saturday && r.StartMinute >= 0 && r.EndMinute <= 24*60 && r.StartMinute < r.EndMinute
}

func (r WeekdayRule) fieldsValid() bool {
	return r.Weekday >= time.Sunday && r.Weekday <= time.Saturday && r.StartMinute >= 0 && r.EndMinute <= 24*60 && r.StartMinute < r.EndMinute
}

type ExceptionKind string

const (
	AvailableException   ExceptionKind = "available"
	UnavailableException ExceptionKind = "unavailable"
)

type AvailabilityException struct {
	Interval Interval
	Kind     ExceptionKind
	Note     string
}

type LessonStatus string

const (
	Scheduled LessonStatus = "scheduled"
	Cancelled LessonStatus = "cancelled"
	// CancelledState names the persisted commercial schedule state.
	CancelledState LessonStatus = Cancelled
)

type Lesson struct {
	TeacherID string
	LearnerID string
	Interval  Interval
	Status    LessonStatus
}

type Slot struct {
	Interval  Interval
	Protected Interval
}

type ParticipantConflict struct {
	ParticipantID string
	Lesson        Lesson
	Interval      Interval
}

type ParticipantRole string

const (
	TeacherRole ParticipantRole = "teacher"
	LearnerRole ParticipantRole = "learner"
)

func ValidateDuration(duration time.Duration) error {
	if duration <= 0 || duration%SlotDuration != 0 {
		return ErrInvalidDuration
	}
	return nil
}

func ValidateAssignmentDuration(duration time.Duration) error { return ValidateDuration(duration) }

func ValidateLessonDuration(duration time.Duration) error { return ValidateDuration(duration) }

func NormalizeAssignmentDuration(duration time.Duration) (time.Duration, error) {
	if duration == 0 {
		return DefaultLessonDuration, nil
	}
	if err := ValidateAssignmentDuration(duration); err != nil {
		return 0, err
	}
	return duration, nil
}

// ValidateDurationOverride permits omission for either role and values only for teachers.
func ValidateDurationOverride(role ParticipantRole, duration *time.Duration) error {
	if duration == nil {
		return nil
	}
	if role != TeacherRole {
		return ErrLearnerDurationChange
	}
	return ValidateLessonDuration(*duration)
}

func ValidateTeacherDurationOverride(duration time.Duration) error {
	return ValidateLessonDuration(duration)
}

func IsOnSlotGrid(t time.Time) bool {
	return isOnGrid(t, SlotDuration)
}

func HorizonEnd(now time.Time) time.Time {
	return now.UTC().Add(businesspolicy.Current().BookingHorizon)
}

func sortIntervals(intervals []Interval) {
	sort.Slice(intervals, func(a, b int) bool { return intervals[a].Start.Before(intervals[b].Start) })
}

func mergeIntervals(intervals []Interval) []Interval {
	if len(intervals) < 2 {
		return intervals
	}
	sortIntervals(intervals)
	merged := make([]Interval, 0, len(intervals))
	for _, current := range intervals {
		if len(merged) == 0 || merged[len(merged)-1].End.Before(current.Start) {
			merged = append(merged, current)
			continue
		}
		if current.End.After(merged[len(merged)-1].End) {
			merged[len(merged)-1].End = current.End
		}
	}
	return merged
}
