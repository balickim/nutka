package commercialread

import (
	"encoding/json"
	"strings"
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

func TestBuildAssignmentSummaryUsesPrecedenceAndSafeFields(t *testing.T) {
	now := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	assignment := Assignment{ID: "a1", TeacherID: "t1", LearnerID: "l1", Active: true}
	pkg := &commercial.Package{ID: "p1", Assignment: commercial.Assignment{ID: "a1", TeacherID: "t1", LearnerID: "l1"}, Status: commercial.PackageOpen, PurchasedOn: now.AddDate(0, 0, -1), ValidThrough: now.AddDate(0, 0, 30), TeacherZone: "UTC", Tokens: []commercial.Token{{ID: "tok1", Ordinal: 1, State: commercial.TokenAvailable}, {ID: "tok2", Ordinal: 2, State: commercial.TokenReserved}}}
	value, err := BuildAssignmentSummary(SummaryInput{Assignment: assignment, Packages: []*commercial.Package{pkg}, Now: now, Viewer: Viewer{Role: LearnerRole, ID: "l1"}})
	if err != nil {
		t.Fatal(err)
	}
	if value.ActivePlan != domain.PackagePlan || value.Package == nil || value.Package.TokenBalance.Available != 1 || value.Package.TokenBalance.Reserved != 1 || value.Package.ValidThrough != "2030-01-31" {
		t.Fatalf("unexpected summary: %+v", value)
	}
	encoded, _ := json.Marshal(value)
	if strings.Contains(string(encoded), "PocketBase") || strings.Contains(string(encoded), "teacher_id") {
		t.Fatalf("summary leaked persistence fields: %s", encoded)
	}
}

func TestCalendarsPartitionAndHorizonScope(t *testing.T) {
	now := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	policy := businesspolicy.Current()
	assignment := Assignment{ID: "a1", TeacherID: "t1", LearnerID: "l1", Active: true}
	other := Assignment{ID: "a2", TeacherID: "t2", LearnerID: "l2", Active: true}
	contract := regularcontract.RegularContract{ID: "c1", AssignmentID: "a1", TeacherID: "t1", LearnerID: "l1", Status: regularcontract.Active, StartOn: now.AddDate(0, 0, -1), EndOn: now.AddDate(0, 6, 0), EffectiveEndOn: now.AddDate(0, 6, 0), TeacherTimezone: "Europe/Warsaw", Policy: policy, PolicySnapshot: policy.Snapshot(), Occurrences: []regularcontract.Occurrence{occurrence("o-near", "a1", now.AddDate(0, 0, 14)), occurrence("o-later", "a1", now.AddDate(0, 0, 15))}}
	data := []AssignmentData{{Assignment: assignment, Contract: &contract}, {Assignment: other, Lessons: []LessonInput{{ID: "other-lesson", AssignmentID: "a2", LearnerID: "l2", TeacherID: "t2", StartAt: now.AddDate(0, 0, 1), EndAt: now.AddDate(0, 0, 1).Add(45 * time.Minute), Plan: domain.AdHocPlan, ScheduleState: domain.ScheduledState}}}}
	teacher, err := BuildTeacherCalendar(CalendarRequest{Assignments: data, Now: now, Policy: policy, Viewer: Viewer{Role: TeacherRole, ID: "t1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(teacher.NearTermLessons) != 1 || len(teacher.LaterContractLessons) != 1 || teacher.LaterContractLessons[0].ID != "o-later" {
		t.Fatalf("unexpected teacher partition: %+v", teacher)
	}
	learner, err := BuildLearnerCalendar(CalendarRequest{Assignments: data, Now: now, Policy: policy, Viewer: Viewer{Role: LearnerRole, ID: "l1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(learner.NearTermLessons) != 1 || learner.NearTermLessons[0].ID != "o-near" || len(learner.Assignments) != 1 {
		t.Fatalf("unexpected learner calendar: %+v", learner)
	}
}

func TestFocusedReadsAreBoundedAndRoleSafe(t *testing.T) {
	now := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)
	policy := businesspolicy.Current()
	assignment := Assignment{ID: "a1", TeacherID: "t1", LearnerID: "l1", Active: true}
	contract := regularcontract.RegularContract{ID: "c1", AssignmentID: "a1", TeacherID: "t1", LearnerID: "l1", TeacherTimezone: "Europe/Warsaw", Policy: policy, PolicySnapshot: policy.Snapshot(), Occurrences: []regularcontract.Occurrence{occurrence("o1", "a1", now.AddDate(0, 0, 1)), occurrence("o2", "a1", now.AddDate(0, 0, 20)), occurrence("o3", "a1", now.AddDate(0, 0, 21))}}
	series, err := QueryContractSeries(ContractSeriesRequest{Assignment: assignment, Contract: contract, Now: now, Policy: policy, Viewer: Viewer{Role: TeacherRole, ID: "t1"}, Page: 2, PerPage: 1})
	if err != nil || series.TotalItems != 3 || len(series.Later) != 1 || series.Later[0].ID != "o2" {
		t.Fatalf("unexpected series page: %+v, %v", series, err)
	}
	entry := FinancialEntryInput{FinancialEntry: ledger.FinancialEntry{ID: "f1", AssignmentID: "a1", AmountMinor: 5000, Currency: "PLN", Reason: "private"}}
	finance, err := QueryFinancialHistory(FinancialHistoryRequest{Assignment: assignment, Viewer: Viewer{Role: LearnerRole, ID: "l1"}, Entries: []FinancialEntryInput{entry, {FinancialEntry: ledger.FinancialEntry{ID: "other", AssignmentID: "a2", AmountMinor: 100}}}, Page: 1, PerPage: 10})
	if err != nil || len(finance.Items) != 1 || finance.Items[0].Reason != "" {
		t.Fatalf("unexpected learner finance: %+v, %v", finance, err)
	}
	event := history.Event{ID: "e1", Type: history.LessonCancelled, AssignmentID: "a1", AggregateType: "lesson", AggregateID: "l", Actor: history.Actor{Role: history.TeacherActor, ID: "t1"}, EventAt: now.UTC(), PriorState: map[string]any{"teacher_note": "hide", "value": "keep"}, InternalNote: "hide"}
	historyPage, err := QueryBusinessHistory(HistoryRequest{Assignment: assignment, Viewer: Viewer{Role: LearnerRole, ID: "l1"}, Events: []history.Event{event}, Page: 1, PerPage: 10})
	if err != nil || len(historyPage.Items) != 1 || historyPage.Items[0].PriorState["teacher_note"] != nil {
		t.Fatalf("unexpected learner history: %+v, %v", historyPage, err)
	}
	work, err := QueryUnresolvedWork(UnresolvedRequest{Viewer: Viewer{Role: TeacherRole, ID: "t1"}, Assignments: []Assignment{assignment}, Work: []UnresolvedInput{{PendingSettlements: []UnresolvedItem{{ID: "ok", Assignment: "a1"}, {ID: "no", Assignment: "a2"}}}}})
	if err != nil || len(work.PendingSettlements) != 1 || work.PendingSettlements[0].ID != "ok" {
		t.Fatalf("unexpected unresolved work: %+v, %v", work, err)
	}
	if _, err := QueryBusinessHistory(HistoryRequest{Assignment: assignment, Viewer: Viewer{Role: TeacherRole, ID: "other"}, Page: 1, PerPage: 10}); err != ErrUnauthorized {
		t.Fatalf("expected authorization error, got %v", err)
	}
}

func occurrence(id, assignment string, start time.Time) regularcontract.Occurrence {
	interval, _ := scheduling.NewInterval(start, start.Add(45*time.Minute))
	return regularcontract.Occurrence{ID: id, ContractID: "c1", AssignmentID: assignment, TeacherID: "t1", LearnerID: "l1", OriginalLocalDate: start.Format("2006-01-02"), Interval: interval, ScheduleState: regularcontract.Scheduled, UnitPriceMinor: 5000, Currency: "PLN"}
}
