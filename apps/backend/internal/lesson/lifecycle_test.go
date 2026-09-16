package lesson

import (
	"errors"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func testInterval(t *testing.T, start time.Time) scheduling.Interval {
	t.Helper()
	interval, err := scheduling.NewInterval(start, start.Add(45*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return interval
}

func baseLesson(plan commercial.PlanType, now, start time.Time) Lesson {
	interval, _ := scheduling.NewInterval(start, start.Add(45*time.Minute))
	return Lesson{ID: "lesson-1", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", Plan: plan, Interval: interval, ScheduleState: domain.ScheduledState, Outcome: domain.AwaitingOutcome, Settlement: ledger.PendingSettlement, UnitPriceMinor: 8000, Currency: "PLN"}
}

func packageCommand(t *testing.T, now, start time.Time, actor history.Actor) Command {
	t.Helper()
	value, err := commercial.NewPackage(commercial.PurchaseRequest{ID: "package-1", Assignment: commercial.Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, PurchaseDate: now, Now: now, TeacherZone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = value.Reserve(commercial.LessonReference{ID: "lesson-1", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", StartAt: start}, now)
	if err != nil {
		t.Fatal(err)
	}
	lesson := baseLesson(commercial.PackagePlan, now, start)
	return Command{Lesson: lesson, Actor: actor, Now: now, ReplacementStart: start.Add(48 * time.Hour), Availability: []scheduling.Interval{testInterval(t, start.Add(48*time.Hour))}, Package: &value}
}

func TestReschedulePackageKeepsTokenAndLessonIdentity(t *testing.T) {
	now := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	command := packageCommand(t, now, now.Add(48*time.Hour), history.Actor{Role: history.LearnerActor, ID: "learner-1"})
	before := command.Package.Tokens[0]
	decision, err := RescheduleLesson(command)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Lesson.ID != before.LessonID || decision.Package.Tokens[0].State != commercial.TokenReserved || decision.Package.Tokens[0].LessonID != before.LessonID {
		t.Fatalf("reschedule changed token identity: %+v", decision.Package.Tokens[0])
	}
	if decision.Events[0].Type != history.LessonRescheduled || decision.PlanEffect == "lesson_cancelled" {
		t.Fatalf("unexpected reschedule decision: %+v", decision)
	}
}

func TestRescheduleContractUsesOriginalMonthAllowance(t *testing.T) {
	now := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	start := now.Add(48 * time.Hour)
	interval := testInterval(t, start)
	policy := businesspolicy.Current()
	contract := &regularcontract.RegularContract{
		ID: "contract-1", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", TeacherTimezone: "UTC", StartOn: now.Add(-24 * time.Hour), EndOn: now.AddDate(0, 0, 90), EffectiveEndOn: now.AddDate(0, 0, 90), Status: regularcontract.Active, Policy: policy, PolicySnapshot: businesspolicy.CurrentSnapshot(),
		Occurrences: []regularcontract.Occurrence{{ID: "lesson-1", ContractID: "contract-1", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", OriginalLocalDate: "2030-01-03", OriginalStartAt: start, Interval: interval, ScheduleState: regularcontract.Scheduled, BillingOutcome: regularcontract.BillableOrdinary, UnitPriceMinor: 5000, Currency: "PLN"}},
	}
	replacement := start.Add(48 * time.Hour)
	command := Command{Lesson: baseLesson(commercial.RegularContract, now, start), Actor: history.Actor{Role: history.LearnerActor, ID: "learner-1"}, Now: now, ReplacementStart: replacement, Availability: []scheduling.Interval{testInterval(t, replacement)}, Contract: contract}
	decision, err := RescheduleLesson(command)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Contract.Occurrences[0].IndividuallyRescheduled || decision.Contract.Allowances("2030-01").MonthlyReschedulesRemaining != 0 {
		t.Fatalf("contract allowance or replacement missing: %+v", decision.Contract)
	}
}

func TestCancellationExactBoundaryAndLateAdHocNoPenalty(t *testing.T) {
	now := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	command := packageCommand(t, now, now.Add(24*time.Hour), history.Actor{Role: history.LearnerActor, ID: "learner-1"})
	decision, err := CancelLesson(command)
	if err != nil || decision.PlanEffect != "package_token_returned" || decision.Events[0].NewState["cutoff"] != "timely" {
		t.Fatalf("exact cutoff decision: %+v, %v", decision, err)
	}

	adhoc := Command{Lesson: baseLesson(commercial.AdHoc, now, now.Add(23*time.Hour)), Actor: history.Actor{Role: history.LearnerActor, ID: "learner-1"}, Now: now}
	adhoc.Lesson.Settlement = ledger.PendingSettlement
	decision, err = CancelLesson(adhoc)
	if err != nil || decision.Lesson.Settlement != ledger.NotApplicable || decision.PlanEffect != "ad_hoc_not_applicable" {
		t.Fatalf("late ad hoc cancellation: %+v, %v", decision, err)
	}
	if _, err := RescheduleLesson(adhoc); !errors.Is(err, ErrReplacementRequired) {
		t.Fatalf("missing replacement error: %v", err)
	}
}

func TestOutcomeRequiresTeacherAndDoesNotRewriteInterval(t *testing.T) {
	now := time.Date(2030, 1, 1, 10, 46, 0, 0, time.UTC)
	start := now.Add(-60 * time.Minute)
	command := Command{Lesson: baseLesson(commercial.AdHoc, now, start), Actor: history.Actor{Role: history.TeacherActor, ID: "teacher-1"}, Now: now}
	before := command.Lesson.Interval
	decision, err := RecordOutcome(command, domain.Completed)
	if err != nil || !decision.Lesson.Interval.Start.Equal(before.Start) || decision.Lesson.Outcome != domain.Completed {
		t.Fatalf("completed outcome: %+v, %v", decision, err)
	}
	if _, err := RecordOutcome(decisionCommand(decision), domain.Completed); !errors.Is(err, ErrOutcomeAlreadyStored) {
		t.Fatalf("repeated outcome error: %v", err)
	}
	learner := command
	learner.Actor = history.Actor{Role: history.LearnerActor, ID: "learner-1"}
	if _, err := RecordOutcome(learner, domain.Completed); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("learner outcome error: %v", err)
	}
}

func decisionCommand(value Decision) Command {
	return Command{Lesson: value.Lesson, Actor: history.Actor{Role: history.TeacherActor, ID: value.Lesson.TeacherID}, Now: time.Date(2030, 1, 1, 11, 0, 0, 0, time.UTC), Package: value.Package, Contract: value.Contract, Charge: value.Charge, Credits: value.Credits}
}

func TestInvalidActorAndCorrectionReasonLeaveStateUntouched(t *testing.T) {
	now := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	command := packageCommand(t, now, now.Add(48*time.Hour), history.Actor{Role: history.LearnerActor, ID: "other-learner"})
	if _, err := CancelLesson(command); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unrelated learner error: %v", err)
	}
	command.Actor = history.Actor{Role: history.TeacherActor, ID: "teacher-1"}
	before := command.Package.Tokens[0]
	if _, err := Correct(command, "event-1", " "); !errors.Is(err, ErrCorrectionTarget) {
		t.Fatalf("empty correction reason: %v", err)
	}
	if command.Package.Tokens[0] != before {
		t.Fatal("invalid correction changed token")
	}
	if businesspolicy.CurrentVersion == "" {
		t.Fatal("policy version must be stable")
	}
}
