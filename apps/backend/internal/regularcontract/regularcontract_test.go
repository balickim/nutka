package regularcontract

import (
	"errors"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func testInterval(t *testing.T, start, end time.Time) scheduling.Interval {
	t.Helper()
	interval, err := scheduling.NewInterval(start, end)
	if err != nil {
		t.Fatal(err)
	}
	return interval
}

func activatedContract(t *testing.T) RegularContract {
	t.Helper()
	location, _ := time.LoadLocation("Europe/Warsaw")
	now := time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)
	availability := []scheduling.Interval{testInterval(t, time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))}
	contract, err := Activate(ActivationRequest{ID: "contract-1", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", TeacherTimezone: location.String(), Actor: Actor{Role: Teacher, ID: "teacher-1"}, StartOn: time.Date(2025, 9, 1, 0, 0, 0, 0, location), Weekday: time.Monday, StartMinute: 9 * 60, Now: now, Policy: DefaultPolicy(), Availability: availability})
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func TestActivateMaterializesSeriesAndKeepsDSTLocalIdentity(t *testing.T) {
	contract := activatedContract(t)
	if len(contract.Occurrences) != 44 {
		t.Fatalf("occurrences = %d, want 44", len(contract.Occurrences))
	}
	var before, after Occurrence
	for _, occurrence := range contract.Occurrences {
		if occurrence.OriginalLocalDate == "2025-10-20" {
			before = occurrence
		}
		if occurrence.OriginalLocalDate == "2025-10-27" {
			after = occurrence
		}
	}
	if before.Interval.Start.In(mustLocationForTest(t, contract.TeacherTimezone)).Hour() != 9 || after.Interval.Start.In(mustLocationForTest(t, contract.TeacherTimezone)).Hour() != 9 {
		t.Fatal("DST changed the selected local start")
	}
	if before.Interval.Start.Equal(after.Interval.Start) || before.OriginalLocalDate != "2025-10-20" || after.OriginalLocalDate != "2025-10-27" {
		t.Fatal("occurrence identity or UTC resolution is not stable")
	}
}

func mustLocationForTest(t *testing.T, name string) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return location
}

func TestActivateRejectsBackdateWithoutEveryPastOutcome(t *testing.T) {
	location := mustLocationForTest(t, "Europe/Warsaw")
	start := time.Date(2025, 8, 1, 0, 0, 0, 0, location)
	now := time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)
	availability := []scheduling.Interval{testInterval(t, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))}
	_, err := Activate(ActivationRequest{ID: "c", AssignmentID: "a", TeacherID: "t", LearnerID: "l", TeacherTimezone: location.String(), Actor: Actor{Role: Teacher, ID: "t"}, StartOn: start, Weekday: time.Friday, StartMinute: 9 * 60, Now: now, Policy: DefaultPolicy(), Availability: availability, BackdateReason: "admin correction", PastOutcomes: map[string]string{"2025-08-01": "completed"}})
	if !errors.Is(err, ErrMissingPastOutcome) {
		t.Fatalf("error = %v, want missing past outcome", err)
	}
}

func TestLearnerAllowanceUsesOriginalMonthAndDoesNotAccumulate(t *testing.T) {
	contract := activatedContract(t)
	location := mustLocationForTest(t, contract.TeacherTimezone)
	var source Occurrence
	for _, occurrence := range contract.Occurrences {
		if occurrence.OriginalLocalDate == "2025-09-29" {
			source = occurrence
			break
		}
	}
	now := time.Date(2025, 9, 20, 8, 0, 0, 0, time.UTC)
	replacementDate := time.Date(2025, 10, 3, 0, 0, 0, 0, location)
	replacement, _ := scheduling.ResolveWallTime(location, replacementDate.Year(), replacementDate.Month(), replacementDate.Day(), 9, 0, 0, 0)
	available := []scheduling.Interval{testInterval(t, replacement.Add(-time.Hour), replacement.Add(2*time.Hour))}
	if err := contract.Reschedule(RescheduleRequest{OccurrenceID: source.ID, AssignmentID: contract.AssignmentID, ReplacementStart: replacement, Now: now, Actor: Actor{Role: Learner, ID: contract.LearnerID}, Available: available}); err != nil {
		t.Fatal(err)
	}
	if got := contract.Allowances("2025-09").MonthlyReschedulesRemaining; got != 0 {
		t.Fatalf("September allowance = %d, want 0", got)
	}
	if got := contract.Allowances("2025-10").MonthlyReschedulesRemaining; got != 1 {
		t.Fatalf("October allowance = %d, want 1", got)
	}
	if err := contract.Reschedule(RescheduleRequest{OccurrenceID: source.ID, AssignmentID: contract.AssignmentID, ReplacementStart: replacement.Add(7 * 24 * time.Hour), Now: now, Actor: Actor{Role: Learner, ID: contract.LearnerID}, Available: available}); !errors.Is(err, ErrAlreadyRescheduled) {
		t.Fatalf("second reschedule error = %v, want already rescheduled", err)
	}
}

func TestLearnerCannotUseSecondMonthlyReschedule(t *testing.T) {
	contract := activatedContract(t)
	location := mustLocationForTest(t, contract.TeacherTimezone)
	now := time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)
	var first, second Occurrence
	for _, occurrence := range contract.Occurrences {
		if occurrence.OriginalLocalDate == "2025-09-08" {
			first = occurrence
		}
		if occurrence.OriginalLocalDate == "2025-09-15" {
			second = occurrence
		}
	}
	replacementDate := time.Date(2025, 9, 10, 0, 0, 0, 0, location)
	replacement, _ := scheduling.ResolveWallTime(location, replacementDate.Year(), replacementDate.Month(), replacementDate.Day(), 9, 0, 0, 0)
	available := []scheduling.Interval{testInterval(t, replacement.Add(-time.Hour), replacement.Add(2*time.Hour))}
	request := RescheduleRequest{AssignmentID: contract.AssignmentID, Now: now, Actor: Actor{Role: Learner, ID: contract.LearnerID}, Available: available, ReplacementStart: replacement}
	request.OccurrenceID = first.ID
	if err := contract.Reschedule(request); err != nil {
		t.Fatal(err)
	}
	request.OccurrenceID = second.ID
	request.ReplacementStart = replacement.Add(5 * 24 * time.Hour)
	if err := contract.Reschedule(request); !errors.Is(err, ErrAllowanceExhausted) {
		t.Fatalf("second monthly reschedule error = %v, want allowance exhausted", err)
	}
}

func TestContractCancellationsAndMonthlySnapshots(t *testing.T) {
	contract := activatedContract(t)
	now := time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)
	for _, occurrence := range contract.Occurrences[:3] {
		cancelAt := occurrence.Interval.Start.Add(-48 * time.Hour)
		if err := contract.Cancel(CancellationRequest{OccurrenceID: occurrence.ID, Now: cancelAt, Actor: Actor{Role: Learner, ID: contract.LearnerID}}); err != nil {
			t.Fatal(err)
		}
	}
	balance := contract.Allowances("2025-09")
	if balance.FreeCancellationsRemaining != 0 {
		t.Fatalf("free cancellations = %d, want 0", balance.FreeCancellationsRemaining)
	}
	snapshots, err := contract.MonthSnapshots(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) == 0 || snapshots[0].AmountMinor <= 0 || snapshots[0].BillableCount <= 0 {
		t.Fatalf("invalid monthly snapshot: %+v", snapshots)
	}
}

func TestLateCancellationRemainsBillableAndNoShowIsExplicit(t *testing.T) {
	contract := activatedContract(t)
	occurrence := contract.Occurrences[0]
	if err := contract.Cancel(CancellationRequest{OccurrenceID: occurrence.ID, Now: occurrence.Interval.Start.Add(-time.Hour), Actor: Actor{Role: Learner, ID: contract.LearnerID}}); err != nil {
		t.Fatal(err)
	}
	cancelled, _ := contract.occurrence(occurrence.ID)
	if cancelled.BillingOutcome != BillableLateCancel || !cancelled.Billable() {
		t.Fatal("late cancellation should remain billable while retaining cancellation state")
	}
	noShow := contract.Occurrences[1]
	if err := contract.RecordOutcome(noShow.ID, "learner_no_show", Actor{Role: Teacher, ID: contract.TeacherID}, noShow.Interval.End.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got, _ := contract.occurrence(noShow.ID); got.BillingOutcome != BillableNoShow || got.Outcome != "learner_no_show" {
		t.Fatal("no-show outcome was not retained")
	}
}

func TestTeacherRescheduleCanCrossContractEndAndPermanentChangePreservesIndividualMove(t *testing.T) {
	contract := activatedContract(t)
	location := mustLocationForTest(t, contract.TeacherTimezone)
	var source Occurrence
	for _, occurrence := range contract.Occurrences {
		if occurrence.OriginalLocalDate == "2026-06-29" {
			source = occurrence
		}
	}
	if source.ID == "" {
		return
	}
	now := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	replacementDate := time.Date(2026, 7, 5, 0, 0, 0, 0, location)
	replacement, _ := scheduling.ResolveWallTime(location, replacementDate.Year(), replacementDate.Month(), replacementDate.Day(), 9, 0, 0, 0)
	available := []scheduling.Interval{testInterval(t, replacement.Add(-time.Hour), replacement.Add(2*time.Hour))}
	if err := contract.Reschedule(RescheduleRequest{OccurrenceID: source.ID, AssignmentID: contract.AssignmentID, ReplacementStart: replacement, Now: now, Actor: Actor{Role: Teacher, ID: contract.TeacherID}, Available: available}); err != nil {
		t.Fatal(err)
	}
	updated, _ := contract.occurrence(source.ID)
	if !updated.IndividuallyRescheduled || !updated.Interval.Start.Equal(replacement) {
		t.Fatal("teacher replacement was not retained")
	}
}

func TestAmendmentIsProspectiveAndRenewalStartsDistinctContract(t *testing.T) {
	contract := activatedContract(t)
	now := time.Date(2025, 9, 10, 8, 0, 0, 0, time.UTC)
	if err := contract.AmendPrice(PriceAmendmentRequest{EffectiveMonth: "2025-11", PriceMinor: 6000, Currency: "PLN", Now: now, Actor: Actor{Role: Teacher, ID: contract.TeacherID}}); err != nil {
		t.Fatal(err)
	}
	if contract.PriceMinor != 5000 {
		t.Fatal("base contract price changed")
	}
	if contract.Status != Active {
		t.Fatal("amendment changed contract state")
	}
	oldPrice := contract.Occurrences[0].UnitPriceMinor
	var November *Occurrence
	for index := range contract.Occurrences {
		if contract.Occurrences[index].OriginalLocalDate == "2025-11-03" {
			November = &contract.Occurrences[index]
		}
	}
	if oldPrice != 5000 || November == nil || November.UnitPriceMinor != 6000 {
		t.Fatalf("prospective amendment snapshots = old %d, November %+v", oldPrice, November)
	}
}
