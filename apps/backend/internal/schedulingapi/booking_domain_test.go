package schedulingapi

import (
	"errors"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func TestBookingIntervalUsesLearnerMinimumAndTeacherConfirmationBoundary(t *testing.T) {
	now := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	policy := businesspolicy.Current()
	short := now.Add(45 * time.Minute)
	if _, err := bookingInterval(short, now, "learner", policy); !errors.Is(err, errHorizon) {
		t.Fatalf("learner booking inside 24-hour minimum must fail with horizon, got %v", err)
	}
	interval, err := bookingInterval(short, now, "teacher", policy)
	if err != nil || interval.Duration() != policy.LessonDuration {
		t.Fatalf("teacher booking may use confirmed short notice, interval=%#v err=%v", interval, err)
	}
	exact := now.Add(policy.LearnerBookingMinimum)
	if _, err := bookingInterval(exact, now, "learner", policy); err != nil {
		t.Fatalf("exact learner minimum must be eligible: %v", err)
	}
}

func TestBookingIntervalConstrainsStartNotLessonEnd(t *testing.T) {
	now := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	policy := businesspolicy.Current()
	start := now.Add(policy.BookingHorizon)
	interval, err := bookingInterval(start, now, "learner", policy)
	if err != nil {
		t.Fatalf("exact horizon start must be eligible: %v", err)
	}
	if !interval.End.After(now.Add(policy.BookingHorizon)) || interval.Duration() != policy.LessonDuration {
		t.Fatalf("horizon must constrain start only: %#v", interval)
	}
}

func TestMapEligibilityErrorPreservesSharedPlanRules(t *testing.T) {
	if !errors.Is(mapEligibilityError(commercial.ErrContractActive), errPlanPrecedence) {
		t.Fatal("active contract must map to plan precedence")
	}
	if !errors.Is(mapEligibilityError(commercial.ErrPackageUnavailable), errPackageUnavailable) {
		t.Fatal("package exhaustion must map to package unavailable")
	}
	now := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	if _, err := scheduling.ValidateBookingStartWithPolicy(now.Add(48*time.Hour), now, businesspolicy.Current()); err != nil {
		t.Fatalf("shared scheduling policy should remain callable: %v", err)
	}
}
