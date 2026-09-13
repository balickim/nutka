package scheduling

import (
	"errors"
	"testing"
	"time"
)

func utc(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

func interval(t *testing.T, start, end time.Time) Interval {
	t.Helper()
	value, err := NewInterval(start, end)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestIntervalsUseHalfOpenBoundsAndFiveMinuteBuffers(t *testing.T) {
	left := interval(t, utc(2026, 1, 1, 10, 0), utc(2026, 1, 1, 10, 45))
	right := interval(t, utc(2026, 1, 1, 10, 45), utc(2026, 1, 1, 11, 0))
	if left.Intersects(right) {
		t.Fatal("touching half-open intervals must not intersect")
	}
	protected := left.Protected()
	if !protected.Start.Equal(utc(2026, 1, 1, 9, 55)) || !protected.End.Equal(utc(2026, 1, 1, 10, 50)) {
		t.Fatalf("unexpected protected interval: %#v", protected)
	}
	if !protected.Contains(left) {
		t.Fatal("protected interval must contain lesson interval")
	}
}

func TestDurationDefaultsAndRoleValidation(t *testing.T) {
	value, err := NormalizeAssignmentDuration(0)
	if err != nil || value != DefaultLessonDuration {
		t.Fatalf("zero assignment duration must use default: %v %s", err, value)
	}
	for _, invalid := range []time.Duration{-15 * time.Minute, 0, 10 * time.Minute, 15*time.Minute + time.Second} {
		if err := ValidateAssignmentDuration(invalid); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("duration %s should be invalid, got %v", invalid, err)
		}
	}
	valid := 60 * time.Minute
	if err := ValidateDurationOverride(TeacherRole, &valid); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDurationOverride(LearnerRole, &valid); !errors.Is(err, ErrLearnerDurationChange) {
		t.Fatalf("learner override should be rejected: %v", err)
	}
	if err := ValidateDurationOverride(LearnerRole, nil); err != nil {
		t.Fatalf("omitted learner override should be allowed: %v", err)
	}
}

func TestResolveWallTimeDSTPolicy(t *testing.T) {
	location, err := LoadTimezone("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	nonexistent, err := ResolveWallTime(location, 2025, time.March, 9, 2, 30, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !nonexistent.Equal(utc(2025, time.March, 9, 7, 30)) {
		t.Fatalf("nonexistent wall time must shift over the gap: %s", nonexistent)
	}
	ambiguous, err := ResolveWallTime(location, 2025, time.November, 2, 1, 30, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !ambiguous.Equal(utc(2025, time.November, 2, 5, 30)) {
		t.Fatalf("ambiguous wall time must choose the earliest instant: %s", ambiguous)
	}
}

func TestTimezoneDefaultsAndInvalidZones(t *testing.T) {
	value, err := NormalizeTimezone("")
	if err != nil || value != DefaultTimezone {
		t.Fatalf("missing timezone must default to %s: %q %v", DefaultTimezone, value, err)
	}
	if err := ValidateTimezone("Mars/Olympus"); !errors.Is(err, ErrInvalidTimezone) {
		t.Fatalf("unknown timezone must be rejected: %v", err)
	}
	if _, err := EffectiveAvailability(utc(2026, time.January, 1, 0, 0), utc(2026, time.January, 2, 0, 0), "Mars/Olympus", nil, nil); !errors.Is(err, ErrInvalidTimezone) {
		t.Fatalf("availability must reject unknown timezone: %v", err)
	}
}

func TestRecurringExpansionConvertsAcrossDST(t *testing.T) {
	start := utc(2025, time.March, 1, 0, 0)
	end := utc(2025, time.April, 1, 0, 0)
	rule := NewWeekdayRule(time.Sunday, 9*60, 10*60)
	values, err := ExpandWeeklyRules(start, end, "America/New_York", []WeekdayRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 5 {
		t.Fatalf("expected five Sunday occurrences, got %d", len(values))
	}
	if !values[1].Start.Equal(utc(2025, time.March, 9, 13, 0)) || !values[2].Start.Equal(utc(2025, time.March, 16, 13, 0)) {
		t.Fatalf("recurring local 09:00 must follow DST offset: %#v", values)
	}
}

func TestAvailableExceptionOpensClosedDate(t *testing.T) {
	start := utc(2026, time.January, 6, 0, 0)
	end := utc(2026, time.January, 7, 0, 0)
	exception := AvailabilityException{Interval: interval(t, utc(2026, time.January, 6, 14, 0), utc(2026, time.January, 6, 15, 0)), Kind: AvailableException}
	values, err := EffectiveAvailability(start, end, "UTC", nil, []AvailabilityException{exception})
	if err != nil || len(values) != 1 || !values[0].Start.Equal(exception.Interval.Start) || !values[0].End.Equal(exception.Interval.End) {
		t.Fatalf("available exception must open closed date: %v %#v", err, values)
	}
}

func TestEffectiveAvailabilityUnavailablePrecedence(t *testing.T) {
	start := utc(2026, time.January, 5, 0, 0)
	end := utc(2026, time.January, 6, 0, 0)
	rule := NewWeekdayRule(time.Monday, 9*60, 12*60)
	available := AvailabilityException{Interval: interval(t, utc(2026, time.January, 5, 12, 0), utc(2026, time.January, 5, 13, 0)), Kind: AvailableException}
	unavailable := AvailabilityException{Interval: interval(t, utc(2026, time.January, 5, 10, 0), utc(2026, time.January, 5, 12, 30)), Kind: UnavailableException}
	values, err := EffectiveAvailability(start, end, "UTC", []WeekdayRule{rule}, []AvailabilityException{available, unavailable})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || !values[0].Start.Equal(utc(2026, time.January, 5, 9, 0)) || !values[0].End.Equal(utc(2026, time.January, 5, 10, 0)) || !values[1].Start.Equal(utc(2026, time.January, 5, 12, 30)) || !values[1].End.Equal(utc(2026, time.January, 5, 13, 0)) {
		t.Fatalf("unavailable precedence produced %#v", values)
	}
}

func TestGenerateSlotsUsesGridHorizonAndAvailabilityFit(t *testing.T) {
	now := utc(2026, time.January, 5, 9, 7)
	rule := NewWeekdayRule(time.Monday, 9*60, 11*60)
	slots, err := GenerateSlots(now, "UTC", []WeekdayRule{rule}, nil, "teacher", "learner", nil, 45*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 11 {
		t.Fatalf("expected eleven 45-minute starts across two Mondays, got %d", len(slots))
	}
	if !slots[0].Interval.Start.Equal(utc(2026, time.January, 5, 9, 15)) || !slots[4].Interval.Start.Equal(utc(2026, time.January, 5, 10, 15)) {
		t.Fatalf("unexpected slot bounds: %#v", slots)
	}
	if _, err := ValidateBookingStart(utc(2026, time.January, 5, 9, 16), now, 45*time.Minute); !errors.Is(err, ErrInvalidGrid) {
		t.Fatalf("off-grid start must fail with grid error: %v", err)
	}
	if _, err := ValidateBookingStart(HorizonEnd(now).Truncate(SlotDuration).Add(SlotDuration), now, 45*time.Minute); !errors.Is(err, ErrHorizon) {
		t.Fatalf("outside-horizon start must fail: %v", err)
	}
	shortRule := NewWeekdayRule(time.Monday, 10*60+15, 11*60)
	slots, err = GenerateSlots(now, "UTC", []WeekdayRule{shortRule}, nil, "teacher", "learner", nil, 45*time.Minute)
	if err != nil || len(slots) != 2 || !slots[0].Interval.Start.Equal(utc(2026, time.January, 5, 10, 15)) {
		t.Fatalf("lesson must fit effective availability: %v %#v", err, slots)
	}
}

func TestValidateBookingStartRequiresCompleteHorizonFit(t *testing.T) {
	now := utc(2026, time.January, 5, 9, 0)
	boundary := HorizonEnd(now)
	if _, err := ValidateBookingStart(boundary.Add(-45*time.Minute), now, 45*time.Minute); err != nil {
		t.Fatalf("lesson ending exactly at horizon must be allowed: %v", err)
	}
	if _, err := ValidateBookingStart(boundary, now, 15*time.Minute); !errors.Is(err, ErrHorizon) {
		t.Fatalf("lesson extending beyond horizon must be rejected: %v", err)
	}
}

func TestNoRecurringRuleProducesNoSlotsAndDisabledInvalidRuleIsIgnored(t *testing.T) {
	now := utc(2026, time.January, 5, 9, 0)
	if slots, err := GenerateSlots(now, "UTC", nil, nil, "teacher", "learner", nil, 45*time.Minute); err != nil || len(slots) != 0 {
		t.Fatalf("no recurring rule must produce no slots: %v %#v", err, slots)
	}
	disabledInvalid := WeekdayRule{Weekday: time.Monday, StartMinute: -1, EndMinute: 2000, Enabled: false}
	if slots, err := GenerateSlots(now, "UTC", []WeekdayRule{disabledInvalid}, nil, "teacher", "learner", nil, 45*time.Minute); err != nil || len(slots) != 0 {
		t.Fatalf("disabled invalid rule must remain inactive: %v %#v", err, slots)
	}
}

func TestParticipantConflictsIncludeBuffersAndIgnoreCancelled(t *testing.T) {
	proposed := interval(t, utc(2026, time.January, 5, 10, 0), utc(2026, time.January, 5, 10, 45))
	near := Lesson{TeacherID: "teacher", LearnerID: "other", Interval: interval(t, utc(2026, time.January, 5, 10, 50), utc(2026, time.January, 5, 11, 30)), Status: Scheduled}
	far := Lesson{TeacherID: "other", LearnerID: "learner", Interval: interval(t, utc(2026, time.January, 5, 10, 50), utc(2026, time.January, 5, 11, 30)), Status: Scheduled}
	cancelled := Lesson{TeacherID: "teacher", LearnerID: "learner", Interval: interval(t, utc(2026, time.January, 5, 9, 0), utc(2026, time.January, 5, 10, 45)), Status: Cancelled}
	conflicts := DetectConflicts(proposed, "teacher", "learner", []Lesson{near, far, cancelled})
	if len(conflicts) != 2 {
		t.Fatalf("expected teacher and learner conflicts, got %d", len(conflicts))
	}
	if err := ValidateNoParticipantConflict(proposed, "teacher", "learner", []Lesson{near}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict error: %v", err)
	}
}

func TestProtectedGapBoundary(t *testing.T) {
	proposed := interval(t, utc(2026, time.January, 5, 10, 0), utc(2026, time.January, 5, 10, 45))
	exactGap := Lesson{TeacherID: "teacher", Interval: interval(t, utc(2026, time.January, 5, 10, 55), utc(2026, time.January, 5, 11, 40)), Status: Scheduled}
	lessGap := Lesson{TeacherID: "teacher", Interval: interval(t, utc(2026, time.January, 5, 10, 54), utc(2026, time.January, 5, 11, 39)), Status: Scheduled}
	if HasParticipantConflict(proposed, "teacher", "", []Lesson{exactGap}) {
		t.Fatal("ten minutes between lesson intervals must satisfy five-minute buffers")
	}
	if !HasParticipantConflict(proposed, "teacher", "", []Lesson{lessGap}) {
		t.Fatal("less than ten minutes between lesson intervals must conflict")
	}
}
