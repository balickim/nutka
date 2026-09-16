package migrations

import (
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestCommercialMigrationCreatesSchemaAndIndexes(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatal(err)
	}
	schedulingstore.RegisterHooks(app)
	if err := migrateCommercialSchema(app); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		schedulingstore.LessonPackagesCollectionName,
		schedulingstore.PackageTokensCollectionName,
		schedulingstore.RegularContractsCollectionName,
		schedulingstore.ContractAmendmentsCollectionName,
		schedulingstore.ContractMonthsCollectionName,
		schedulingstore.ChargesCollectionName,
		schedulingstore.FinancialEntriesCollectionName,
		schedulingstore.BusinessEventsCollectionName,
	} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil || !collection.IsBase() {
			t.Fatalf("commercial collection %s missing: %v", name, err)
		}
	}
	assignments, err := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	if assignments.Fields.GetByName(schedulingstore.DefaultDurationMinutesField) != nil {
		t.Fatal("legacy assignment duration field must be removed")
	}
	lessons, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range commercialLessonFields {
		if lessons.Fields.GetByName(name) == nil {
			t.Fatalf("lesson field %s missing", name)
		}
	}
	for _, name := range []string{"idx_lessons_contract_occurrence"} {
		if lessons.GetIndex(name) == "" {
			t.Fatalf("lesson index %s missing", name)
		}
	}
	for collection, indexes := range map[string][]string{
		schedulingstore.PackageTokensCollectionName:    {"idx_package_tokens_package_ordinal", "idx_package_tokens_active_lesson"},
		schedulingstore.ContractMonthsCollectionName:   {"idx_contract_months_contract_month"},
		schedulingstore.ChargesCollectionName:          {"idx_charges_source"},
		schedulingstore.FinancialEntriesCollectionName: {"idx_financial_entries_assignment_event_at", "idx_financial_entries_source"},
		schedulingstore.BusinessEventsCollectionName:   {"idx_business_events_assignment_event_at"},
	} {
		row, _ := app.FindCollectionByNameOrId(collection)
		for _, index := range indexes {
			if row.GetIndex(index) == "" {
				t.Fatalf("index %s missing on %s", index, collection)
			}
		}
	}
	packages, _ := app.FindCollectionByNameOrId(schedulingstore.LessonPackagesCollectionName)
	if got := strings.Join(packages.Fields.GetByName(schedulingstore.PackageStatusField).(*core.SelectField).Values, "|"); got != "open|closed" {
		t.Fatalf("package status values must be canonical, got %q", got)
	}
	contracts, _ := app.FindCollectionByNameOrId(schedulingstore.RegularContractsCollectionName)
	if got := strings.Join(contracts.Fields.GetByName(schedulingstore.StatusField).(*core.SelectField).Values, "|"); got != "active|notice_given|ended" {
		t.Fatalf("contract status values must be canonical, got %q", got)
	}
	tokens, _ := app.FindCollectionByNameOrId(schedulingstore.PackageTokensCollectionName)
	ordinal := tokens.Fields.GetByName(schedulingstore.OrdinalField).(*core.NumberField)
	if ordinal.Max == nil || *ordinal.Max != float64(businesspolicy.PackageTokenCount) {
		t.Fatalf("token ordinal max must use policy token count, got %v", ordinal.Max)
	}
}

func TestCommercialMigrationRunsThroughPocketBaseBootstrap(t *testing.T) {
	app := newMigrationTestApp(t)
	schedulingstore.RegisterHooks(app)
	if err := app.RunAppMigrations(); err != nil {
		t.Fatal(err)
	}
	if _, err := app.FindCollectionByNameOrId(schedulingstore.BusinessEventsCollectionName); err != nil {
		t.Fatalf("bootstrap did not apply commercial migration: %v", err)
	}
}

func TestCommercialRepairMigrationUpgradesAnAlreadyMigratedDatabase(t *testing.T) {
	app := newMigrationTestApp(t)
	schedulingstore.RegisterHooks(app)
	if err := app.RunAppMigrations(); err != nil {
		t.Fatal(err)
	}
	teachers, _ := app.FindCollectionByNameOrId(teachersCollection)
	learners, _ := app.FindCollectionByNameOrId(learnersCollection)
	assignments, _ := app.FindCollectionByNameOrId(teacherLearnersCollection)
	teacher := newAuthRecord(t, app, teachers, "repair-teacher@example.test", "Teacher")
	learner := newAuthRecord(t, app, learners, "repair-learner@example.test", "Learner")
	assignment := core.NewRecord(assignments)
	assignment.Set("teacher", teacher.Id)
	assignment.Set("learner", learner.Id)
	if err := app.Save(assignment); err != nil {
		t.Fatal(err)
	}
	lessons, _ := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	lesson := core.NewRecord(lessons)
	lesson.Set("teacher", teacher.Id)
	lesson.Set("learner", learner.Id)
	lesson.Set(schedulingstore.AssignmentField, assignment.Id)
	lesson.Set(schedulingstore.StartAtField, "2030-01-07T09:00:00Z")
	lesson.Set(schedulingstore.EndAtField, "2030-01-07T09:45:00Z")
	lesson.Set(schedulingstore.DurationMinutesField, commercialLessonMinutes)
	lesson.Set(schedulingstore.StatusField, "scheduled")
	lesson.Set(schedulingstore.PlanTypeField, "ad_hoc")
	lesson.Set(schedulingstore.OriginalLocalDateField, "2030-01-07")
	lesson.Set(schedulingstore.OriginalStartAtField, "2030-01-07T09:00:00Z")
	lesson.Set(schedulingstore.PolicyVersionField, commercialPolicyVersion)
	lesson.Set(schedulingstore.PolicySnapshotField, policySnapshotJSON())
	lesson.Set(schedulingstore.UnitPriceMinorField, commercialLessonPrice)
	lesson.Set(schedulingstore.CurrencyField, commercialCurrency)
	lesson.Set(schedulingstore.ScheduleStateField, "scheduled")
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}

	entries, err := app.FindCollectionByNameOrId(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	entries.RemoveIndex("idx_financial_entries_assignment_event_at")
	entries.RemoveIndex("idx_financial_entries_source")
	entries.Fields.RemoveByName(schedulingstore.AssignmentField)
	if err := app.Save(entries); err != nil {
		t.Fatal(err)
	}
	legacyEntry := core.NewRecord(entries)
	legacyEntry.Set(schedulingstore.EntryTypeField, "charge_created")
	legacyEntry.Set(schedulingstore.AmountMinorField, commercialLessonPrice)
	legacyEntry.Set(schedulingstore.CurrencyField, commercialCurrency)
	legacyEntry.Set(schedulingstore.RelatedLessonField, lesson.Id)
	legacyEntry.Set(schedulingstore.ActorRoleField, "system")
	legacyEntry.Set(schedulingstore.EventAtField, "2030-01-07T08:00:00Z")
	if err := app.Save(legacyEntry); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB().NewQuery("UPDATE {{financial_entries}} SET [[entry_type]] = 'creation' WHERE [[id]] = {:id}").Bind(dbx.Params{"id": legacyEntry.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB().NewQuery("DELETE FROM {{_migrations}} WHERE [[file]] = {:file}").Bind(dbx.Params{
		"file": "1788600000_repair_commercial_schema.go",
	}).Execute(); err != nil {
		t.Fatal(err)
	}

	if err := app.RunAppMigrations(); err != nil {
		t.Fatal(err)
	}
	if _, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName, dbx.HashExp{
		schedulingstore.AssignmentField: "assignment-id",
	}); err != nil {
		t.Fatalf("filter repaired financial entries by assignment: %v", err)
	}
	repaired, err := app.FindRecordById(schedulingstore.FinancialEntriesCollectionName, legacyEntry.Id)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.GetString(schedulingstore.AssignmentField) != assignment.Id || repaired.GetString(schedulingstore.SourceTypeField) != "ad_hoc" || repaired.GetString(schedulingstore.SourceIDField) != lesson.Id || repaired.GetString(schedulingstore.EntryTypeField) != "charge_created" {
		t.Fatalf("legacy financial entry was not repaired: %v", repaired.Original())
	}
}

func TestCommercialMigrationDownRestoresLegacyDurationSchema(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatal(err)
	}
	schedulingstore.RegisterHooks(app)
	if err := migrateCommercialSchema(app); err != nil {
		t.Fatal(err)
	}
	assignments, _ := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	if assignments.Fields.GetByName(schedulingstore.DefaultDurationMinutesField) != nil {
		t.Fatal("commercial migration must remove legacy duration before down migration")
	}
	if err := revertCommercialSchema(app); err != nil {
		t.Fatal(err)
	}
	assignments, err := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	if err != nil || assignments.Fields.GetByName(schedulingstore.DefaultDurationMinutesField) == nil {
		t.Fatalf("down migration did not restore duration field: %v", err)
	}
}

func TestCommercialMigrationBackfillsLessonsAndLegacyEvents(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatal(err)
	}
	schedulingstore.RegisterHooks(app)
	teachers, _ := app.FindCollectionByNameOrId(teachersCollection)
	learners, _ := app.FindCollectionByNameOrId(learnersCollection)
	assignments, _ := app.FindCollectionByNameOrId(teacherLearnersCollection)
	teacher := newAuthRecord(t, app, teachers, "audit-teacher@example.test", "Teacher")
	learner := newAuthRecord(t, app, learners, "audit-learner@example.test", "Learner")
	assignment := core.NewRecord(assignments)
	assignment.Set("teacher", teacher.Id)
	assignment.Set("learner", learner.Id)
	assignment.Set(schedulingstore.DefaultDurationMinutesField, 30)
	if err := app.Save(assignment); err != nil {
		t.Fatal(err)
	}
	lessons, _ := app.FindCollectionByNameOrId(lessonsCollection)
	lesson := core.NewRecord(lessons)
	lesson.Set("teacher", teacher.Id)
	lesson.Set("learner", learner.Id)
	lesson.Set("assignment", assignment.Id)
	lesson.Set("start_at", "2020-01-01 09:00:00.000Z")
	lesson.Set("end_at", "2020-01-01 09:30:00.000Z")
	lesson.Set("duration_minutes", 30)
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	legacy, _ := app.FindCollectionByNameOrId(lessonEventsCollection)
	event := core.NewRecord(legacy)
	event.Set("lesson", lesson.Id)
	event.Set("kind", "created")
	event.Set("initiator_role", "teacher")
	event.Set("initiator_id", teacher.Id)
	event.Set("event_at", "2020-01-01 08:00:00.000Z")
	event.Set("duration_minutes", 30)
	if err := app.Save(event); err != nil {
		t.Fatal(err)
	}

	if err := migrateCommercialSchema(app); err != nil {
		t.Fatal(err)
	}
	reloaded, err := app.FindRecordById(lessonsCollection, lesson.Id)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GetInt(schedulingstore.DurationMinutesField) != businesspolicy.LessonDurationMinutes || reloaded.GetString(schedulingstore.PlanTypeField) != "ad_hoc" || reloaded.GetString(schedulingstore.OutcomeField) != "awaiting_outcome" {
		t.Fatalf("lesson was not normalized: duration=%d plan=%q outcome=%q", reloaded.GetInt(schedulingstore.DurationMinutesField), reloaded.GetString(schedulingstore.PlanTypeField), reloaded.GetString(schedulingstore.OutcomeField))
	}
	businessEvents, _ := app.FindCollectionByNameOrId(schedulingstore.BusinessEventsCollectionName)
	rows, err := app.FindAllRecords(businessEvents)
	if err != nil || len(rows) != 1 || rows[0].GetString(schedulingstore.LegacyEventIDField) != event.Id {
		t.Fatalf("legacy event was not preserved: rows=%d err=%v", len(rows), err)
	}
}

func TestCommercialHooksProtectEventsAndOwnership(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatal(err)
	}
	schedulingstore.RegisterHooks(app)
	if err := migrateCommercialSchema(app); err != nil {
		t.Fatal(err)
	}
	teachers, _ := app.FindCollectionByNameOrId(teachersCollection)
	learners, _ := app.FindCollectionByNameOrId(learnersCollection)
	assignments, _ := app.FindCollectionByNameOrId(teacherLearnersCollection)
	teacher := newAuthRecord(t, app, teachers, "hook-teacher@example.test", "Teacher")
	learner := newAuthRecord(t, app, learners, "hook-learner@example.test", "Learner")
	assignment := core.NewRecord(assignments)
	assignment.Set("teacher", teacher.Id)
	assignment.Set("learner", learner.Id)
	if err := app.Save(assignment); err != nil {
		t.Fatal(err)
	}
	events, _ := app.FindCollectionByNameOrId(schedulingstore.BusinessEventsCollectionName)
	event := core.NewRecord(events)
	event.Set(schedulingstore.AssignmentField, assignment.Id)
	event.Set(schedulingstore.AggregateTypeField, "availability")
	event.Set(schedulingstore.AggregateIDField, "availability-1")
	event.Set(schedulingstore.EventTypeField, "availability_changed")
	event.Set(schedulingstore.ActorRoleField, "teacher")
	event.Set(schedulingstore.EventAtField, time.Now().UTC().Format(time.RFC3339Nano))
	if err := app.Save(event); err != nil {
		t.Fatal(err)
	}
	event.Set(schedulingstore.EventTypeField, "changed")
	if err := app.Save(event); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("event update should be rejected: %v", err)
	}
	if err := app.Delete(event); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("event delete should be rejected: %v", err)
	}
}

func newAuthRecord(t *testing.T, app core.App, collection *core.Collection, email, name string) *core.Record {
	t.Helper()
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.Set("name", name)
	record.SetPassword("local-password")
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}
