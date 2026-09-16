package availabilityimpact

import (
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func TestPlanSeparatesNearConflictsAndDistantOmissions(t *testing.T) {
	now := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	near := interval(t, now.Add(48*time.Hour), now.Add(48*time.Hour+45*time.Minute))
	distant := interval(t, now.Add(21*24*time.Hour), now.Add(21*24*time.Hour+45*time.Minute))
	preview, err := planner(t).Preview(Input{Now: now, Proposal: Proposal{Operation: Create, Target: Exception}, NearTermLessons: []Lesson{{ID: "near", Plan: domain.AdHocPlan, Interval: near, State: domain.ScheduledState}}, DistantOccurrences: []ContractOccurrence{{ID: "later", Interval: distant, State: domain.ScheduledState}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.NearTermConflicts) != 1 || preview.NearTermConflicts[0].LessonID != "near" {
		t.Fatalf("near-term conflicts = %#v", preview.NearTermConflicts)
	}
	if got := preview.NearTermConflicts[0].AllowedResolutions; len(got) != 2 || got[0] != Cancel || got[1] != Reschedule {
		t.Fatalf("allowed resolutions = %#v", got)
	}
	if len(preview.DistantEffects) != 1 || preview.DistantEffects[0].Effect != Omit {
		t.Fatalf("distant effects = %#v", preview.DistantEffects)
	}
}

func TestPlanRestoresOnlyDistantPlannedOmissions(t *testing.T) {
	now := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	near := interval(t, now.Add(24*time.Hour), now.Add(24*time.Hour+45*time.Minute))
	distant := interval(t, now.Add(21*24*time.Hour), now.Add(21*24*time.Hour+45*time.Minute))
	preview, err := planner(t).Preview(Input{Now: now, Proposal: Proposal{Operation: Enable, Target: Exception, EffectiveAvailability: []scheduling.Interval{near, distant}}, NearTermLessons: []Lesson{{ID: "cancelled", Plan: domain.AdHocPlan, Interval: near, State: domain.CancelledState}}, DistantOccurrences: []ContractOccurrence{{ID: "restored", Interval: distant, State: domain.OmittedState, OmissionReason: "planned_unavailability"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.NearTermConflicts) != 0 {
		t.Fatalf("cancelled near lesson was recreated: %#v", preview.NearTermConflicts)
	}
	if len(preview.DistantEffects) != 1 || preview.DistantEffects[0].Effect != Restore {
		t.Fatalf("distant restoration = %#v", preview.DistantEffects)
	}
}

func TestPlanDoesNotRestoreNonAvailabilityOmissions(t *testing.T) {
	now := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	distant := interval(t, now.Add(21*24*time.Hour), now.Add(21*24*time.Hour+45*time.Minute))
	for _, reason := range []string{"", "planned_omission", "contract_termination", "billing_adjustment"} {
		preview, err := planner(t).Preview(Input{Now: now, Proposal: Proposal{Operation: Enable, Target: Exception, EffectiveAvailability: []scheduling.Interval{distant}}, DistantOccurrences: []ContractOccurrence{{ID: reason + "-occurrence", Interval: distant, State: domain.OmittedState, OmissionReason: reason}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(preview.DistantEffects) != 0 {
			t.Fatalf("omission reason %q was restored: %#v", reason, preview.DistantEffects)
		}
	}
}

func TestPlanVersionIsOpaqueDeterministicAndOrderIndependent(t *testing.T) {
	now := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	a := interval(t, now.Add(48*time.Hour), now.Add(48*time.Hour+45*time.Minute))
	b := interval(t, now.Add(72*time.Hour), now.Add(72*time.Hour+45*time.Minute))
	first := Input{Now: now, StateVersion: "records-1", Proposal: Proposal{Operation: Disable, Target: RecurringRule, EffectiveAvailability: []scheduling.Interval{a}}, NearTermLessons: []Lesson{{ID: "b", Plan: domain.AdHocPlan, Interval: b}, {ID: "a", Plan: domain.AdHocPlan, Interval: a}}}
	second := first
	second.NearTermLessons = []Lesson{{ID: "a", Plan: domain.AdHocPlan, Interval: a}, {ID: "b", Plan: domain.AdHocPlan, Interval: b}}
	left, err := planner(t).Preview(first)
	if err != nil {
		t.Fatal(err)
	}
	right, err := planner(t).Preview(second)
	if err != nil {
		t.Fatal(err)
	}
	if left.PreviewVersion == "" || left.PreviewVersion != right.PreviewVersion {
		t.Fatalf("versions = %q and %q", left.PreviewVersion, right.PreviewVersion)
	}
	second.StateVersion = "records-2"
	changed, err := planner(t).Preview(second)
	if err != nil {
		t.Fatal(err)
	}
	if changed.PreviewVersion == left.PreviewVersion {
		t.Fatal("version did not change after relevant state changed")
	}
	advanced := first
	advanced.Now = now.Add(time.Minute)
	stable, err := planner(t).Preview(advanced)
	if err != nil {
		t.Fatal(err)
	}
	if stable.PreviewVersion != left.PreviewVersion {
		t.Fatalf("version changed without a classification change: %q and %q", left.PreviewVersion, stable.PreviewVersion)
	}
}

func TestPlanUsesExactHorizonStartAndDoesNotInferHolidays(t *testing.T) {
	now := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	policy := scheduling.DefaultIntervalPolicy()
	atHorizon := interval(t, now.Add(policy.Horizon), now.Add(policy.Horizon+45*time.Minute))
	availability := []scheduling.Interval{interval(t, atHorizon.Start.Add(-time.Hour), atHorizon.End.Add(time.Hour))}
	preview, err := planner(t).Preview(Input{Now: now, Proposal: Proposal{Operation: Create, Target: RecurringRule, EffectiveAvailability: availability}, NearTermLessons: []Lesson{{ID: "horizon", Plan: domain.AdHocPlan, Interval: atHorizon, State: domain.ScheduledState}}, DistantOccurrences: []ContractOccurrence{{ID: "holiday", Interval: atHorizon, State: domain.ScheduledState}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.NearTermConflicts) != 0 || len(preview.DistantEffects) != 0 {
		t.Fatalf("available horizon or unmarked holiday changed: %#v", preview)
	}
}

func TestPlanVersionChangesWhenHorizonClassificationChanges(t *testing.T) {
	now := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	policy := businesspolicy.Current()
	occurrenceStart := now.Add(policy.BookingHorizon + 30*time.Second)
	occurrence := interval(t, occurrenceStart, occurrenceStart.Add(45*time.Minute))
	input := Input{Now: now, Proposal: Proposal{Operation: Disable, Target: RecurringRule}, DistantOccurrences: []ContractOccurrence{{ID: "later", Interval: occurrence, State: domain.ScheduledState}}}
	first, err := planner(t).Preview(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Now = now.Add(time.Minute)
	second, err := planner(t).Preview(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.PreviewVersion == second.PreviewVersion || len(first.DistantEffects) != 1 || len(second.DistantEffects) != 0 {
		t.Fatalf("horizon classification did not change: first=%#v second=%#v", first, second)
	}
}

func TestPlanRejectsIncompleteOrUnknownProposal(t *testing.T) {
	for name, proposal := range map[string]Proposal{
		"missing operation": {Target: Exception},
		"missing target":    {Operation: Create},
		"unknown operation": {Operation: "replace", Target: Exception},
		"unknown target":    {Operation: Create, Target: "holiday"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := planner(t).Preview(Input{Now: time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC), Proposal: proposal}); err != ErrInvalidProposal {
				t.Fatalf("error = %v, want %v", err, ErrInvalidProposal)
			}
		})
	}
}

func planner(t *testing.T) Planner {
	t.Helper()
	value, err := NewPlanner(businesspolicy.Current())
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func interval(t *testing.T, start, end time.Time) scheduling.Interval {
	t.Helper()
	value, err := scheduling.NewInterval(start, end)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
