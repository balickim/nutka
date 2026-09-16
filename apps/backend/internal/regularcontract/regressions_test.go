package regularcontract

import (
	"errors"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func TestRestoreOmittedOnlyRestoresDistantOccurrences(t *testing.T) {
	contract := activatedContract(t)
	now := time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)
	near := &contract.Occurrences[0]
	near.ScheduleState, near.BillingOutcome = Omitted, PlannedOmission
	if err := contract.RestoreOmitted(near.ID, []scheduling.Interval{testInterval(t, near.Interval.Start.Add(-time.Hour), near.Interval.End.Add(time.Hour))}, Actor{Role: Teacher, ID: contract.TeacherID}, now); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("near omission restore error = %v, want unavailable", err)
	}
	if near.ScheduleState != Omitted {
		t.Fatal("near-term omission was silently recreated")
	}
	var distant *Occurrence
	for index := range contract.Occurrences {
		if contract.Occurrences[index].Interval.Start.After(now.Add(contract.Policy.BookingHorizon)) {
			distant = &contract.Occurrences[index]
			break
		}
	}
	if distant == nil {
		t.Fatal("test contract has no distant occurrence")
	}
	distant.ScheduleState, distant.BillingOutcome = Omitted, PlannedOmission
	window := []scheduling.Interval{testInterval(t, distant.Interval.Start.Add(-time.Hour), distant.Interval.End.Add(time.Hour))}
	if err := contract.RestoreOmitted(distant.ID, window, Actor{Role: Teacher, ID: contract.TeacherID}, now); err != nil {
		t.Fatal(err)
	}
	if distant.ScheduleState != Scheduled || distant.BillingOutcome != BillableOrdinary {
		t.Fatalf("distant omission was not restored: %#v", distant)
	}
}

func TestNoticeAndEarlyEndPreserveContractEndAndUseTerminationEffect(t *testing.T) {
	contract := activatedContract(t)
	originalEnd := contract.EndOn
	noticeAt := time.Date(2025, 9, 10, 8, 0, 0, 0, time.UTC)
	if err := contract.SubmitNotice(Actor{Role: Learner, ID: contract.LearnerID}, noticeAt); err != nil {
		t.Fatal(err)
	}
	effective := contract.EffectiveEndOn
	if !contract.EndOn.Equal(originalEnd) || !effective.Before(originalEnd) {
		t.Fatalf("notice changed contractual end: end=%s original=%s effective=%s", contract.EndOn, originalEnd, effective)
	}
	if err := contract.SubmitNotice(Actor{Role: Teacher, ID: contract.TeacherID}, noticeAt.AddDate(0, 1, 0)); err != nil {
		t.Fatal(err)
	}
	if !contract.EffectiveEndOn.Equal(effective) || !contract.NoticeAt.Equal(noticeAt) {
		t.Fatal("repeated notice changed the original notice decision")
	}
	for _, occurrence := range contract.Occurrences {
		if occurrence.ScheduleState == Cancelled && occurrence.OriginalLocalDate > "2025-10-31" && occurrence.BillingOutcome != ContractTermination {
			t.Fatalf("notice used a penalty effect: %#v", occurrence)
		}
	}

	contract = activatedContract(t)
	originalEnd = contract.EndOn
	if err := contract.EndEarly(time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), "mutual end", Actor{Role: Teacher, ID: contract.TeacherID}, noticeAt); err != nil {
		t.Fatal(err)
	}
	if !contract.EndOn.Equal(originalEnd) || contract.Status != Ended {
		t.Fatalf("early end changed contractual end: end=%s original=%s", contract.EndOn, originalEnd)
	}
	effective = contract.EffectiveEndOn
	if err := contract.EndEarly(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), "again", Actor{Role: Teacher, ID: contract.TeacherID}, noticeAt); !errors.Is(err, ErrInvalidEarlyEnd) {
		t.Fatalf("repeated early end error = %v, want invalid early end", err)
	}
	if !contract.EffectiveEndOn.Equal(effective) {
		t.Fatal("repeated early end changed the first decision")
	}
}

func TestNoticeAndEarlyEndUseTeacherLocalEndDateAtTermBoundary(t *testing.T) {
	location := mustLocationForTest(t, "Europe/Warsaw")
	availability := []scheduling.Interval{testInterval(t, time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))}
	newContract := func(t *testing.T) RegularContract {
		t.Helper()
		contract, err := Activate(ActivationRequest{ID: "boundary", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", TeacherTimezone: location.String(), Actor: Actor{Role: Teacher, ID: "teacher-1"}, StartOn: time.Date(2025, 9, 2, 0, 0, 0, 0, location), Weekday: time.Tuesday, StartMinute: 9 * 60, Now: time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC), Policy: DefaultPolicy(), Availability: availability})
		if err != nil {
			t.Fatal(err)
		}
		return contract
	}
	now := time.Date(2026, 6, 20, 8, 0, 0, 0, time.UTC)
	contract := newContract(t)
	if err := contract.SubmitNotice(Actor{Role: Teacher, ID: contract.TeacherID}, now); err != nil {
		t.Fatal(err)
	}
	if !contract.EffectiveEndOn.Equal(contract.EndOn) || contract.EffectiveEndOn.Format("2006-01-02") != "2026-06-30" {
		t.Fatalf("notice must clamp to the contractual end: effective=%s end=%s", contract.EffectiveEndOn, contract.EndOn)
	}
	for _, occurrence := range contract.Occurrences {
		if occurrence.OriginalLocalDate == "2026-06-30" && (occurrence.ScheduleState != Scheduled || occurrence.BillingOutcome == ContractTermination) {
			t.Fatalf("occurrence on effective local end was terminated: %#v", occurrence)
		}
	}

	contract = newContract(t)
	if err := contract.EndEarly(time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), "mutual end", Actor{Role: Teacher, ID: contract.TeacherID}, now); err != nil {
		t.Fatal(err)
	}
	for _, occurrence := range contract.Occurrences {
		if occurrence.OriginalLocalDate == "2026-06-30" && (occurrence.ScheduleState != Scheduled || occurrence.BillingOutcome == ContractTermination) {
			t.Fatalf("occurrence on early local end was terminated: %#v", occurrence)
		}
	}
}

func TestContractUsesSnapshotSchedulingValuesAndExcludesSourceRecord(t *testing.T) {
	contract := activatedContract(t)
	contract.Policy.StartGrid = 30 * time.Minute
	contract.Policy.ParticipantBuffer = 0
	source := contract.Occurrences[1]
	replacement := source.Interval.Start.Add(7 * 24 * time.Hour)
	other := scheduling.Lesson{TeacherID: contract.TeacherID, LearnerID: contract.LearnerID, Interval: source.Interval, Status: scheduling.Scheduled}
	available := []scheduling.Interval{testInterval(t, replacement.Add(-time.Hour), replacement.Add(2*time.Hour))}
	request := RescheduleRequest{OccurrenceID: source.ID, AssignmentID: contract.AssignmentID, ReplacementStart: replacement, Now: time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC), Actor: Actor{Role: Teacher, ID: contract.TeacherID}, Available: available,
		ParticipantLessonRecords: []ParticipantLesson{{ID: source.ID, TeacherID: source.TeacherID, LearnerID: source.LearnerID, Interval: source.Interval, Status: scheduling.Scheduled}},
		ParticipantLessons:       []scheduling.Lesson{other}}
	if err := contract.Reschedule(request); err != nil {
		t.Fatalf("source identity should be excluded from conflict records: %v", err)
	}
}

func TestCorrectionOfCorrectionRestoresOriginalCancellationEffect(t *testing.T) {
	contract := activatedContract(t)
	occurrence := contract.Occurrences[0]
	at := occurrence.Interval.Start.Add(-48 * time.Hour)
	if err := contract.Cancel(CancellationRequest{OccurrenceID: occurrence.ID, Now: at, Actor: Actor{Role: Learner, ID: contract.LearnerID}}); err != nil {
		t.Fatal(err)
	}
	original := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(original, "restore cancellation", Actor{Role: Teacher, ID: contract.TeacherID}, at); err != nil {
		t.Fatal(err)
	}
	correction := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(correction, "reapply cancellation", Actor{Role: Teacher, ID: contract.TeacherID}, at); err != nil {
		t.Fatal(err)
	}
	secondCorrection := contract.Events[len(contract.Events)-1].ID
	current, _ := contract.occurrence(occurrence.ID)
	if current.ScheduleState != Cancelled || current.BillingOutcome != FreeLearnerCancel {
		t.Fatalf("correction chain did not restore cancellation: %#v", current)
	}
	if err := contract.CorrectEvent(secondCorrection, "restore cancellation again", Actor{Role: Teacher, ID: contract.TeacherID}, at); err != nil {
		t.Fatal(err)
	}
	current, _ = contract.occurrence(occurrence.ID)
	if current.ScheduleState != Scheduled || current.BillingOutcome != BillableOrdinary {
		t.Fatalf("third correction did not invert cancellation: %#v", current)
	}
	if err := contract.CorrectEvent(original, "already compensated", Actor{Role: Teacher, ID: contract.TeacherID}, at); !errors.Is(err, ErrInvalidContract) {
		t.Fatalf("already compensated correction error = %v, want invalid contract", err)
	}
	if err := contract.CorrectEvent("missing", "missing target", Actor{Role: Teacher, ID: contract.TeacherID}, at); !errors.Is(err, ErrInvalidContract) {
		t.Fatalf("missing correction target error = %v, want invalid contract", err)
	}
}

func TestAllowanceProjectionFollowsCancellationCorrectionParity(t *testing.T) {
	contract := activatedContract(t)
	source := contract.Occurrences[1]
	at := source.Interval.Start.Add(-48 * time.Hour)
	actor := Actor{Role: Learner, ID: contract.LearnerID}
	teacher := Actor{Role: Teacher, ID: contract.TeacherID}
	if err := contract.Cancel(CancellationRequest{OccurrenceID: source.ID, Now: at, Actor: actor}); err != nil {
		t.Fatal(err)
	}
	if got := contract.Allowances(source.OriginalLocalMonth()).FreeCancellationsRemaining; got != 1 {
		t.Fatalf("after cancellation free cancellations = %d, want 1", got)
	}
	c1 := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(c1, "reverse cancellation", teacher, at); err != nil {
		t.Fatal(err)
	}
	if got := contract.Allowances(source.OriginalLocalMonth()).FreeCancellationsRemaining; got != 2 {
		t.Fatalf("after C1 free cancellations = %d, want 2", got)
	}
	c2 := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(c2, "reapply cancellation", teacher, at); err != nil {
		t.Fatal(err)
	}
	if got := contract.Allowances(source.OriginalLocalMonth()).FreeCancellationsRemaining; got != 1 {
		t.Fatalf("after C2 free cancellations = %d, want 1", got)
	}
	c3 := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(c3, "reverse cancellation again", teacher, at); err != nil {
		t.Fatal(err)
	}
	if got := contract.Allowances(source.OriginalLocalMonth()).FreeCancellationsRemaining; got != 2 {
		t.Fatalf("after C3 free cancellations = %d, want 2", got)
	}
}

func TestRescheduleEligibilityFollowsCorrectionParity(t *testing.T) {
	contract := activatedContract(t)
	source := contract.Occurrences[1]
	now := time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)
	replacement := source.Interval.Start.Add(7 * 24 * time.Hour)
	available := []scheduling.Interval{testInterval(t, replacement.Add(-time.Hour), replacement.Add(2*time.Hour))}
	learner := Actor{Role: Learner, ID: contract.LearnerID}
	teacher := Actor{Role: Teacher, ID: contract.TeacherID}
	request := RescheduleRequest{OccurrenceID: source.ID, AssignmentID: contract.AssignmentID, ReplacementStart: replacement, Now: now, Actor: learner, Available: available}
	if err := contract.Reschedule(request); err != nil {
		t.Fatal(err)
	}
	if !contract.occurrenceWasLearnerRescheduled(source.ID) || contract.Allowances(source.OriginalLocalMonth()).MonthlyReschedulesRemaining != 0 {
		t.Fatal("initial learner reschedule was not projected")
	}
	c1 := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(c1, "reverse reschedule", teacher, now); err != nil {
		t.Fatal(err)
	}
	if contract.occurrenceWasLearnerRescheduled(source.ID) || contract.Allowances(source.OriginalLocalMonth()).MonthlyReschedulesRemaining != 1 {
		t.Fatal("C1 did not restore once-per-occurrence eligibility")
	}
	c2 := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(c2, "reapply reschedule", teacher, now); err != nil {
		t.Fatal(err)
	}
	if !contract.occurrenceWasLearnerRescheduled(source.ID) || contract.Allowances(source.OriginalLocalMonth()).MonthlyReschedulesRemaining != 0 {
		t.Fatal("C2 did not restore the learner reschedule effect")
	}
	c3 := contract.Events[len(contract.Events)-1].ID
	if err := contract.CorrectEvent(c3, "reverse reschedule again", teacher, now); err != nil {
		t.Fatal(err)
	}
	if contract.occurrenceWasLearnerRescheduled(source.ID) || contract.Allowances(source.OriginalLocalMonth()).MonthlyReschedulesRemaining != 1 {
		t.Fatal("C3 did not restore once-per-occurrence eligibility")
	}
}

func TestRecordOutcomeIsSingleSubmission(t *testing.T) {
	contract := activatedContract(t)
	occurrence := contract.Occurrences[0]
	at := occurrence.Interval.End.Add(time.Minute)
	actor := Actor{Role: Teacher, ID: contract.TeacherID}
	if err := contract.RecordOutcome(occurrence.ID, "completed", actor, at); err != nil {
		t.Fatal(err)
	}
	if err := contract.RecordOutcome(occurrence.ID, "learner_no_show", actor, at.Add(time.Minute)); !errors.Is(err, ErrInvalidContract) {
		t.Fatalf("second outcome error = %v, want invalid contract", err)
	}
}

func TestPriceAmendmentRequiresCanonicalPLN(t *testing.T) {
	contract := activatedContract(t)
	if err := contract.AmendPrice(PriceAmendmentRequest{EffectiveMonth: "2025-11", PriceMinor: 6000, Currency: "pln", Now: time.Date(2025, 9, 10, 8, 0, 0, 0, time.UTC), Actor: Actor{Role: Teacher, ID: contract.TeacherID}}); !errors.Is(err, ErrInvalidAmendment) {
		t.Fatalf("lowercase amendment currency error = %v, want invalid amendment", err)
	}
}
