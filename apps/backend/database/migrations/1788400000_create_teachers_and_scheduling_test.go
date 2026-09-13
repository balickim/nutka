package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func newMigrationTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	return newMigrationTestAppAt(t, t.TempDir())
}

func newMigrationTestAppAt(t *testing.T, dataDir string) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}

func createLearnerCollection(t *testing.T, app core.App) *core.Collection {
	t.Helper()
	collection := core.NewAuthCollection(learnersCollection)
	collection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 160})
	if err := app.Save(collection); err != nil {
		t.Fatalf("save learners collection: %v", err)
	}
	return collection
}

func createUsersRecord(t *testing.T, app core.App) (*core.Collection, *core.Record) {
	t.Helper()
	users, err := app.FindCollectionByNameOrId(legacyUsersCollection)
	if err != nil {
		t.Fatalf("find default users collection: %v", err)
	}
	record := core.NewRecord(users)
	record.SetEmail("teacher@example.test")
	record.Set("name", "Existing Teacher")
	record.SetPassword("local-password")
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatalf("save user record: %v", err)
	}
	return users, record
}

func removeDefaultUsers(t *testing.T, app core.App) {
	t.Helper()
	users, err := app.FindCollectionByNameOrId(legacyUsersCollection)
	if err != nil {
		t.Fatalf("find default users collection: %v", err)
	}
	if _, err := app.DB().NewQuery("DROP TABLE [[users]]").Execute(); err != nil {
		t.Fatalf("drop default users table: %v", err)
	}
	if _, err := app.DB().NewQuery("DELETE FROM {{_collections}} WHERE [[id]] = {:id}").Bind(map[string]any{"id": users.Id}).Execute(); err != nil {
		t.Fatalf("remove default users metadata: %v", err)
	}
	if err := app.ReloadCachedCollections(); err != nil {
		t.Fatalf("reload collections: %v", err)
	}
}

func TestTeacherSchedulingMigrationFreshDatabase(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatalf("migrate fresh database: %v", err)
	}
	schedulingstore.RegisterHooks(app)

	teachers, err := app.FindCollectionByNameOrId(teachersCollection)
	if err != nil || !teachers.IsAuth() {
		t.Fatalf("teachers auth collection missing: %v", err)
	}
	if field := teachers.Fields.GetByName(teacherTimezoneField); field == nil {
		t.Fatal("teachers timezone field missing")
	}
	assertClosedTeacherCollection(t, teachers)
	for _, name := range []string{teacherLearnersCollection, availabilityRulesCollection, availabilityExceptions, lessonsCollection, lessonEventsCollection} {
		collection, findErr := app.FindCollectionByNameOrId(name)
		if findErr != nil || !collection.IsBase() {
			t.Fatalf("missing base collection %s: %v", name, findErr)
		}
	}
	assertSchedulingSchema(t, app, teachers)

	record := core.NewRecord(teachers)
	record.SetEmail("new-teacher@example.test")
	record.Set("name", "New Teacher")
	record.SetPassword("local-password")
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatalf("save fresh teacher: %v", err)
	}
	if record.GetString(teacherTimezoneField) != defaultTeacherTimezone {
		t.Fatalf("fresh teacher timezone = %q", record.GetString(teacherTimezoneField))
	}
	if invalid := saveTeacherTimezone(app, teachers, "Mars/Phobos"); invalid == nil {
		t.Fatal("invalid timezone must be rejected")
	}
	learners, err := app.FindCollectionByNameOrId(learnersCollection)
	if err != nil {
		t.Fatal(err)
	}
	assertPersistedDefaults(t, app, teachers, learners, "fresh")
}

func TestTeacherSchedulingMigrationRenamesDefaultUsersAndPreservesCredentials(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	users, original := createUsersRecord(t, app)
	id := original.Id
	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatalf("migrate users database: %v", err)
	}
	if _, err := app.FindCollectionByNameOrId(legacyUsersCollection); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("legacy users collection remains: %v", err)
	}
	teachers, err := app.FindCollectionByNameOrId(teachersCollection)
	if err != nil {
		t.Fatal(err)
	}
	migrated, err := app.FindRecordById(teachers, id)
	if err != nil {
		t.Fatalf("find migrated teacher: %v", err)
	}
	if migrated.Id != id || !migrated.ValidatePassword("local-password") || migrated.GetString(teacherTimezoneField) != defaultTeacherTimezone {
		t.Fatal("rename did not preserve teacher identity, credentials, or timezone")
	}
	if users.Id != teachers.Id {
		t.Fatal("rename must preserve collection identifier")
	}
	assertClosedTeacherCollection(t, teachers)
	assertSchedulingSchema(t, app, teachers)

	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatalf("rerun migration: %v", err)
	}
	if count, countErr := app.CountRecords(teachersCollection); countErr != nil || count != 1 {
		t.Fatalf("rerun changed teacher records: count=%d err=%v", count, countErr)
	}
}

func TestTeacherSchedulingMigrationRejectsDualAuthCollections(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	createUsersRecord(t, app)
	teachers := core.NewAuthCollection(teachersCollection)
	if err := app.Save(teachers); err != nil {
		t.Fatalf("save teachers collection: %v", err)
	}
	err := migrateTeachersAndScheduling(app)
	if err == nil || !strings.Contains(err.Error(), "both collections exist") {
		t.Fatalf("expected clear dual-collection error, got %v", err)
	}
}

func TestTeacherSchedulingMigrationCreatesTeacherWithoutUsers(t *testing.T) {
	app := newMigrationTestApp(t)
	createLearnerCollection(t, app)
	removeDefaultUsers(t, app)
	if err := migrateTeachersAndScheduling(app); err != nil {
		t.Fatalf("migrate database without users: %v", err)
	}
	teachers, err := app.FindCollectionByNameOrId(teachersCollection)
	if err != nil || !teachers.IsAuth() {
		t.Fatalf("fresh teachers collection missing: %v", err)
	}
	assertClosedTeacherCollection(t, teachers)
}

func TestTeacherTimezoneValidationIsRegisteredAfterMigrationRerun(t *testing.T) {
	dataDir := t.TempDir()
	first := newMigrationTestAppAt(t, dataDir)
	createLearnerCollection(t, first)
	if err := migrateTeachersAndScheduling(first); err != nil {
		t.Fatalf("migrate initial database: %v", err)
	}
	if err := first.ResetBootstrapState(); err != nil {
		t.Fatalf("close initial app: %v", err)
	}

	second := newMigrationTestAppAt(t, dataDir)
	schedulingstore.RegisterHooks(second)
	teachers, err := second.FindCollectionByNameOrId(teachersCollection)
	if err != nil {
		t.Fatalf("find teachers after reboot: %v", err)
	}
	if invalid := saveTeacherTimezone(second, teachers, "Mars/Phobos"); invalid == nil {
		t.Fatal("invalid timezone must remain rejected after migration is applied")
	}
	learners, err := second.FindCollectionByNameOrId(learnersCollection)
	if err != nil {
		t.Fatal(err)
	}
	assertPersistedDefaults(t, second, teachers, learners, "reboot")
}

func saveTeacherTimezone(app core.App, collection *core.Collection, timezone string) error {
	record := core.NewRecord(collection)
	record.SetEmail("invalid@example.test")
	record.Set("name", "Invalid Teacher")
	record.SetPassword("local-password")
	record.SetVerified(true)
	record.Set(teacherTimezoneField, timezone)
	return app.Save(record)
}

func assertClosedTeacherCollection(t *testing.T, collection *core.Collection) {
	t.Helper()
	if collection.CreateRule != nil || collection.UpdateRule != nil || collection.DeleteRule != nil || collection.ListRule != nil || collection.ViewRule != nil {
		t.Fatal("teacher collection rules must be closed")
	}
	if collection.AuthRule == nil || collection.AuthToken.Duration != int64(authconfig.SessionDuration/time.Second) || !collection.PasswordAuth.Enabled || collection.MFA.Enabled || collection.OTP.Enabled {
		t.Fatal("teacher auth configuration is not closed and password-enabled")
	}
	if collection.Fields.GetByName("avatar") == nil {
		t.Fatal("teacher avatar field missing")
	}
}

func assertSchedulingSchema(t *testing.T, app core.App, teachers *core.Collection) {
	t.Helper()
	learners, err := app.FindCollectionByNameOrId(learnersCollection)
	if err != nil {
		t.Fatal(err)
	}
	assignments, err := app.FindCollectionByNameOrId(teacherLearnersCollection)
	if err != nil {
		t.Fatal(err)
	}
	assertRelationTarget(t, assignments, "teacher", teachers.Id)
	assertRelationTarget(t, assignments, "learner", learners.Id)
	if assignments.GetIndex("idx_teacher_learners_pair") == "" {
		t.Fatal("teacher-learner pair must have a unique index")
	}
	exceptions, err := app.FindCollectionByNameOrId(availabilityExceptions)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"start_at", "end_at"} {
		if _, ok := exceptions.Fields.GetByName(name).(*core.DateField); !ok {
			t.Fatalf("exception %s must be a date field", name)
		}
	}
	lessons, err := app.FindCollectionByNameOrId(lessonsCollection)
	if err != nil {
		t.Fatal(err)
	}
	assertRelationTarget(t, lessons, "teacher", teachers.Id)
	assertRelationTarget(t, lessons, "learner", learners.Id)
	for _, name := range []string{"start_at", "end_at", "cancelled_at"} {
		if _, ok := lessons.Fields.GetByName(name).(*core.DateField); !ok {
			t.Fatalf("lesson %s must be a date field", name)
		}
	}
	events, err := app.FindCollectionByNameOrId(lessonEventsCollection)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := events.Fields.GetByName("event_at").(*core.DateField); !ok {
		t.Fatal("lesson event timestamp must be a date field")
	}
	for _, name := range []string{"prior_duration_minutes", "new_duration_minutes"} {
		if _, ok := events.Fields.GetByName(name).(*core.NumberField); !ok {
			t.Fatalf("lesson event %s must be a number field", name)
		}
	}
}

func assertRelationTarget(t *testing.T, collection *core.Collection, name, target string) {
	t.Helper()
	field, ok := collection.Fields.GetByName(name).(*core.RelationField)
	if !ok {
		t.Fatalf("%s.%s must be a relation field", collection.Name, name)
	}
	if field.CollectionId != target {
		t.Fatalf("%s.%s relation target = %q, want %q", collection.Name, name, field.CollectionId, target)
	}
}

func assertPersistedDefaults(t *testing.T, app core.App, teachers, learners *core.Collection, suffix string) {
	t.Helper()
	teacher := core.NewRecord(teachers)
	teacher.SetEmail(fmt.Sprintf("defaults-teacher-%s@example.test", suffix))
	teacher.Set("name", "Defaults Teacher")
	teacher.SetPassword("local-password")
	teacher.SetVerified(true)
	if err := app.Save(teacher); err != nil {
		t.Fatalf("save default teacher: %v", err)
	}
	learner := core.NewRecord(learners)
	learner.SetEmail(fmt.Sprintf("defaults-learner-%s@example.test", suffix))
	learner.Set("name", "Defaults Learner")
	learner.SetPassword("local-password")
	learner.SetVerified(true)
	if err := app.Save(learner); err != nil {
		t.Fatalf("save default learner: %v", err)
	}

	assignments, _ := app.FindCollectionByNameOrId(teacherLearnersCollection)
	assignment := core.NewRecord(assignments)
	assignment.Set("teacher", teacher.Id)
	assignment.Set("learner", learner.Id)
	if err := app.Save(assignment); err != nil {
		t.Fatalf("save default assignment: %v", err)
	}
	if !assignment.GetBool("active") || assignment.GetInt("default_duration_minutes") != 45 {
		t.Fatalf("assignment defaults not persisted: active=%v duration=%v", assignment.GetBool("active"), assignment.Get("default_duration_minutes"))
	}

	rules, _ := app.FindCollectionByNameOrId(availabilityRulesCollection)
	rule := core.NewRecord(rules)
	rule.Set("teacher", teacher.Id)
	rule.Set("weekday", 1)
	rule.Set("start_time", "09:00")
	rule.Set("end_time", "10:00")
	rule.Set("enabled", true)
	if err := app.Save(rule); err != nil {
		t.Fatalf("save default availability rule: %v", err)
	}

	lessons, _ := app.FindCollectionByNameOrId(lessonsCollection)
	lesson := core.NewRecord(lessons)
	lesson.Set("teacher", teacher.Id)
	lesson.Set("learner", learner.Id)
	lesson.Set("assignment", assignment.Id)
	lesson.Set("start_at", "2030-01-01 09:00:00.000Z")
	lesson.Set("end_at", "2030-01-01 09:45:00.000Z")
	lesson.Set("duration_minutes", 45)
	if err := app.Save(lesson); err != nil {
		t.Fatalf("save default lesson: %v", err)
	}
	if lesson.GetString("status") != "scheduled" {
		t.Fatalf("lesson status default = %q", lesson.GetString("status"))
	}
}
