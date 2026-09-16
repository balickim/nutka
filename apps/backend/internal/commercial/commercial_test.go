package commercial

import (
	"errors"
	"testing"
	"time"
)

func instant(year int, month time.Month, day, hour, minute int, zone string) time.Time {
	location, err := time.LoadLocation(zone)
	if err != nil {
		panic(err)
	}
	return time.Date(year, month, day, hour, minute, 0, 0, location)
}

func assignment() Assignment { return Assignment{ID: "a1", TeacherID: "t1", LearnerID: "l1"} }

func packageForTest(t *testing.T, purchase, now time.Time) Package {
	t.Helper()
	value, err := NewPackage(PurchaseRequest{
		ID: "p1", Assignment: assignment(), PurchaseDate: purchase, Now: now, TeacherZone: "Europe/Warsaw",
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func lesson(id string, start time.Time) LessonReference {
	a := assignment()
	return LessonReference{ID: id, AssignmentID: a.ID, TeacherID: a.TeacherID, LearnerID: a.LearnerID, Plan: AdHoc, StartAt: start}
}

func TestEligibilityUsesContractThenPackageThenAdHocPrecedence(t *testing.T) {
	now := instant(2026, time.January, 10, 12, 0, "Europe/Warsaw")
	start := now.Add(48 * time.Hour)
	p := packageForTest(t, now, now)
	input := EligibilityInput{Assignment: assignment(), Now: now, LessonStart: start, Packages: []*Package{&p}}
	decision, err := EvaluateEligibility(input)
	if err != nil || decision.Plan != PackagePlan || decision.TokenOrdinal != 1 {
		t.Fatalf("expected first package token, got %#v, %v", decision, err)
	}
	input.RequestedPlan = AdHoc
	if _, err := EvaluateEligibility(input); !errors.Is(err, ErrPlanPrecedence) {
		t.Fatalf("forced ad hoc must be rejected while token is usable: %v", err)
	}
	input.Contract = &Contract{ID: "c1", AssignmentID: "a1", Status: ContractActive}
	if _, err := EvaluateEligibility(input); !errors.Is(err, ErrContractActive) {
		t.Fatalf("contract must reject flexible booking: %v", err)
	}
	input.Contract = nil
	p.Tokens[0].State = TokenUsed
	p.Tokens[1].State = TokenExpired
	p.Tokens[2].State = TokenInvalidated
	p.Tokens[3].State = TokenUsed
	input.Packages = []*Package{&p}
	input.RequestedPlan = ""
	decision, err = EvaluateEligibility(input)
	if err != nil || decision.Plan != AdHoc {
		t.Fatalf("no higher plan must fall back to ad hoc, got %#v, %v", decision, err)
	}
}

func TestEligibilityUsesLessonStartForFutureContractCoverage(t *testing.T) {
	now := instant(2026, time.January, 10, 12, 0, "Europe/Warsaw")
	p := packageForTest(t, now, now)
	contract := &Contract{ID: "c1", AssignmentID: "a1", Status: ContractActive,
		StartOn: now.AddDate(0, 0, 10), EndOn: now.AddDate(0, 0, 20)}
	inside := EligibilityInput{Assignment: assignment(), Now: now, LessonStart: now.AddDate(0, 0, 12), Contract: contract, Packages: []*Package{&p}}
	if _, err := EvaluateEligibility(inside); !errors.Is(err, ErrContractActive) {
		t.Fatalf("lesson inside future contract term must be blocked: %v", err)
	}
	before := inside
	before.LessonStart = now.AddDate(0, 0, 5)
	decision, err := EvaluateEligibility(before)
	if err != nil || decision.Plan != PackagePlan {
		t.Fatalf("lesson before contract term may use an eligible package: %#v, %v", decision, err)
	}
}

func TestEligibilityScopesPackagesToAssignment(t *testing.T) {
	now := instant(2026, time.January, 10, 12, 0, "Europe/Warsaw")
	other := Assignment{ID: "a2", TeacherID: "t2", LearnerID: "l1"}
	p, err := NewPackage(PurchaseRequest{ID: "p2", Assignment: other, PurchaseDate: now, Now: now, TeacherZone: "Europe/Warsaw"})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := EvaluateEligibility(EligibilityInput{Assignment: assignment(), Now: now, LessonStart: now.Add(48 * time.Hour), Packages: []*Package{&p}})
	if err != nil || decision.Plan != AdHoc {
		t.Fatalf("another teacher package must not fund this booking: %#v, %v", decision, err)
	}
}

func TestPackagePurchaseValidityAndDSTUsesLocalDates(t *testing.T) {
	now := instant(2026, time.March, 1, 12, 0, "America/New_York")
	p, err := NewPackage(PurchaseRequest{ID: "p1", Assignment: assignment(), PurchaseDate: now, Now: now, TeacherZone: "America/New_York"})
	if err != nil {
		t.Fatal(err)
	}
	if got := p.ValidThrough.Format("2006-01-02"); got != "2026-04-29" {
		t.Fatalf("60 local dates must end on April 29, got %s", got)
	}
	final := lesson("final", instant(2026, time.April, 29, 23, 0, "America/New_York"))
	if _, err := p.Reserve(final, instant(2026, time.April, 1, 12, 0, "America/New_York")); err != nil {
		t.Fatalf("final validity date must permit booking: %v", err)
	}
	late := lesson("late", instant(2026, time.April, 30, 0, 1, "America/New_York"))
	if _, err := p.Reserve(late, instant(2026, time.April, 1, 12, 0, "America/New_York")); !errors.Is(err, ErrPackageExpired) {
		t.Fatalf("lesson after final local date must be rejected: %v", err)
	}
	if err := p.ReconcileExpiry(instant(2026, time.April, 30, 0, 1, "America/New_York")); err != nil {
		t.Fatal(err)
	}
	if p.AvailableCount() != 0 {
		t.Fatalf("all available tokens must expire after the final local date, got %d", p.AvailableCount())
	}
}

func TestPackageReserveAndLifecycleTransitions(t *testing.T) {
	now := instant(2026, time.January, 10, 12, 0, "Europe/Warsaw")
	p := packageForTest(t, now, now)
	first, err := p.Reserve(lesson("l1", now.Add(48*time.Hour)), now)
	if err != nil || first.Ordinal != 1 || p.AvailableCount() != 3 {
		t.Fatalf("lowest available token must reserve: %#v, %v", first, err)
	}
	if err := p.Reschedule("l1", lesson("l1", now.Add(72*time.Hour)), now); err != nil {
		t.Fatal(err)
	}
	if p.Tokens[0].State != TokenReserved || p.Tokens[0].LessonID != "l1" {
		t.Fatalf("reschedule must retain token reservation: %#v", p.Tokens[0])
	}
	if err := p.Reschedule("l1", lesson("different", now.Add(96*time.Hour)), now); !errors.Is(err, ErrInvalidLesson) {
		t.Fatalf("reschedule must preserve lesson identity: %v", err)
	}
	if err := p.CancelLearner("l1", now); err != nil {
		t.Fatal(err)
	}
	if p.Tokens[0].State != TokenAvailable {
		t.Fatalf("timely learner cancellation must return token: %#v", p.Tokens[0])
	}
	second, err := p.Reserve(lesson("l2", now.Add(96*time.Hour)), now)
	if err != nil || second.Ordinal != 1 {
		t.Fatalf("returned token must be selected first: %#v, %v", second, err)
	}
	if err := p.TeacherCancel("l2", now); err != nil {
		t.Fatal(err)
	}
	if p.ValidThrough.Format("2006-01-02") != "2026-03-17" || p.Tokens[0].State != TokenAvailable {
		t.Fatalf("teacher cancellation must extend by seven dates and return token: %s %#v", p.ValidThrough, p.Tokens[0])
	}
	if _, err := p.Close(now, "teacher correction", nil); err != nil {
		t.Fatal(err)
	}
	if p.Status != PackageClosed || p.AvailableCount() != 0 {
		t.Fatalf("closure must invalidate remaining available tokens: %#v", p)
	}
}

func TestPackageLateCancellationNoShowAndClosureGuard(t *testing.T) {
	now := instant(2026, time.January, 10, 12, 0, "Europe/Warsaw")
	p := packageForTest(t, now, now)
	start := now.Add(48 * time.Hour)
	if _, err := p.Reserve(lesson("late", start), now); err != nil {
		t.Fatal(err)
	}
	if err := p.CancelLearner("late", start.Add(-23*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if p.Tokens[0].State != TokenUsed {
		t.Fatalf("late learner cancellation must consume token: %#v", p.Tokens[0])
	}
	if _, err := p.Reserve(lesson("future", now.Add(72*time.Hour)), now); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Close(now, "", nil); !errors.Is(err, ErrCloseReasonRequired) {
		t.Fatalf("closure requires reason: %v", err)
	}
	if _, err := p.Close(now, "close", nil); !errors.Is(err, ErrFuturePackageLesson) {
		t.Fatalf("future reserved lesson must block closure: %v", err)
	}
}

func TestPackageClosureRefundAndCompensatingTokenCorrection(t *testing.T) {
	now := instant(2026, time.January, 10, 12, 0, "Europe/Warsaw")
	p := packageForTest(t, now, now)
	if _, err := p.Reserve(lesson("l1", now.Add(48*time.Hour)), now); err != nil {
		t.Fatal(err)
	}
	if err := p.Complete("l1"); err != nil {
		t.Fatal(err)
	}
	event, err := p.CorrectTokenState(CorrectionRequest{TokenID: p.Tokens[0].ID, Target: TokenAvailable, CorrectsEvent: "event-1", Reason: "erroneous completion", At: now})
	if err != nil || event.CorrectsEvent != "event-1" || event.ID != "event-1:correction" || len(p.Events) != 1 {
		t.Fatalf("correction must preserve a compensating event: %#v, %v", event, err)
	}
	refund := &RefundInfo{Amount: Money{Minor: 10000, Currency: "PLN"}, Note: "unused balance"}
	decision, err := p.Close(now, "learner request", refund)
	if err != nil || decision.Invalidated != 4 || decision.Refund != refund {
		t.Fatalf("closure must return invalidation and refund metadata: %#v, %v", decision, err)
	}
}

func TestPackagePurchaseAndContractActivationConversionsAreAtomic(t *testing.T) {
	now := instant(2026, time.January, 10, 12, 0, "Europe/Warsaw")
	first := lesson("adhoc-1", now.Add(48*time.Hour))
	state := OverlapState{Assignment: assignment(), Now: now, FutureAdHocLessons: []LessonReference{first}}
	decision, err := DecidePackagePurchase(PurchaseRequest{ID: "p1", Assignment: assignment(), PurchaseDate: now, Now: now, TeacherZone: "Europe/Warsaw"}, state, []string{"adhoc-1"})
	if err != nil || len(decision.ConvertedLessons) != 1 || decision.ConvertedLessons[0].Plan != PackagePlan {
		t.Fatalf("package conversion failed: %#v, %v", decision, err)
	}
	if decision.Package.Tokens[0].State != TokenReserved || decision.Package.Tokens[0].LessonID != "adhoc-1" {
		t.Fatalf("conversion must reserve one token: %#v", decision.Package.Tokens[0])
	}
	if err := ValidatePackagePurchase(state, nil); !errors.Is(err, ErrAdHocOverlap) {
		t.Fatalf("unconverted ad hoc lesson must block purchase: %v", err)
	}
	contract, err := DecideContractActivation(state, []string{"adhoc-1"})
	if err != nil || len(contract.ConvertedLessons) != 1 || contract.ConvertedLessons[0].Plan != RegularContract {
		t.Fatalf("contract conversion failed: %#v, %v", contract, err)
	}
	if _, err := DecidePackagePurchase(PurchaseRequest{ID: "p2", Assignment: assignment(), PurchaseDate: now, Now: now, TeacherZone: "Europe/Warsaw"}, OverlapState{Assignment: assignment(), Now: now, ActiveContract: true}, nil); !errors.Is(err, ErrActiveContractOverlap) {
		t.Fatalf("active contract must block package purchase: %v", err)
	}
}
