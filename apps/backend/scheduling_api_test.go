// Verifies scheduling API authorization, DTO boundaries, defaults, and UTC exception behavior.
package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func seedAssignment(t *testing.T, app core.App, teacher, learner string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	row := core.NewRecord(collection)
	row.Set("teacher", teacher)
	row.Set("learner", learner)
	if err := app.Save(row); err != nil {
		t.Fatal(err)
	}
	return row
}

func loginCookie(t *testing.T, server http.Handler, path, identity string) *http.Cookie {
	t.Helper()
	response := request(t, server, http.MethodPost, path, `{"identity":"`+identity+`","password":"local-password"}`, nil, true)
	if response.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", identity, response.Code, response.Body.String())
	}
	return response.Result().Cookies()[0]
}

func TestSchedulingAssignmentsArePrivateAndTeacherControlled(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	assignment := seedAssignment(t, app, findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test"), findID(t, app, authconfig.LearnersCollectionName, "learner@example.test"))
	if response := request(t, server, http.MethodGet, "/api/teachers/assignments", "", nil, false); response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"code":"unauthenticated"`) {
		t.Fatalf("unauthenticated assignments: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, "/api/teachers/assignments", "", learnerCookie, false); response.Code != http.StatusForbidden {
		t.Fatalf("wrong realm assignments: %d %s", response.Code, response.Body.String())
	}
	response := request(t, server, http.MethodGet, "/api/teachers/assignments", "", teacherCookie, false)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), assignment.Id) {
		t.Fatalf("teacher assignment list: %d %s", response.Code, response.Body.String())
	}
	response = request(t, server, http.MethodGet, "/api/learners/assignments", "", learnerCookie, false)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), assignment.Id) {
		t.Fatalf("learner assignment list: %d %s", response.Code, response.Body.String())
	}
	response = request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"default_duration_minutes":60}`, teacherCookie, true)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("teacher duration override: %d %s", response.Code, response.Body.String())
	}
	response = request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"default_duration_minutes":75}`, learnerCookie, true)
	if response.Code != http.StatusForbidden {
		t.Fatalf("learner assignment mutation: %d %s", response.Code, response.Body.String())
	}
	response = request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"teacher":"spoofed"}`, teacherCookie, true)
	if response.Code != http.StatusForbidden {
		t.Fatalf("spoofed assignment actor: %d %s", response.Code, response.Body.String())
	}
}

func TestCalendarEndpointsWorkAfterCommercialSchemaRepair(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	seedAssignment(t, app, findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test"), findID(t, app, authconfig.LearnersCollectionName, "learner@example.test"))

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
	if _, err := app.DB().NewQuery("DELETE FROM {{_migrations}} WHERE [[file]] = {:file}").Bind(dbx.Params{
		"file": "1788600000_repair_commercial_schema.go",
	}).Execute(); err != nil {
		t.Fatal(err)
	}
	if err := app.RunAppMigrations(); err != nil {
		t.Fatal(err)
	}

	for path, cookie := range map[string]*http.Cookie{
		"/api/teachers/calendar": teacherCookie,
		"/api/learners/calendar": learnerCookie,
	} {
		response := request(t, server, http.MethodGet, path, "", cookie, false)
		if response.Code != http.StatusOK {
			t.Fatalf("%s after schema repair: %d %s", path, response.Code, response.Body.String())
		}
		if path == "/api/teachers/calendar" && (!strings.Contains(response.Body.String(), `"availability_rules":[]`) || !strings.Contains(response.Body.String(), `"availability_exceptions":[]`)) {
			t.Fatalf("%s must retain empty availability arrays: %s", path, response.Body.String())
		}
	}
}

func TestSchedulingAvailabilityReturnsUTCAndRejectsBlockConflict(t *testing.T) {
	app, server := newTestServerWithClock(t, func() time.Time { return time.Date(2026, time.January, 1, 9, 0, 0, 0, time.UTC) })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	createAvailabilityRule(t, server, teacherCookie, time.Date(2026, time.January, 5, 9, 0, 0, 0, time.UTC))
	exception := createAvailabilityException(t, server, teacherCookie, time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC), time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC), "available")
	if exception.Code != http.StatusOK || !strings.Contains(exception.Body.String(), `"start_at":"2026-01-05T09:00:00Z"`) {
		t.Fatalf("UTC exception: %d %s", exception.Code, exception.Body.String())
	}
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	seedScheduledLesson(t, app, assignment.Id, teacherID, learnerID, time.Date(2026, time.January, 5, 10, 15, 0, 0, time.UTC))
	blocked := request(t, server, http.MethodPost, "/api/teachers/availability/preview", `{"operation":"create","target":"exception","exception":{"start_at":"2026-01-05T10:30:00Z","end_at":"2026-01-05T11:30:00Z","kind":"unavailable"}}`, teacherCookie, true)
	if blocked.Code != http.StatusOK || !strings.Contains(blocked.Body.String(), `"near_term_conflicts":[{"lesson":"`+firstLesson(t, app).Id+`"`) {
		t.Fatalf("block conflict preview: %d %s", blocked.Code, blocked.Body.String())
	}
	bufferOnly := createAvailabilityException(t, server, teacherCookie, time.Date(2026, 1, 5, 11, 0, 0, 0, time.UTC), time.Date(2026, 1, 5, 11, 5, 0, 0, time.UTC), "unavailable")
	if bufferOnly.Code != http.StatusOK {
		t.Fatalf("buffer-only block must be allowed: %d %s", bufferOnly.Code, bufferOnly.Body.String())
	}
}

func TestSeedAssignmentRequiresAccountsAndRejectsDuplicate(t *testing.T) {
	t.Setenv("NUTKA_ENV", "development")
	app, _ := newTestServer(t)
	if err := seedAssignmentCommand(app, "missing-teacher", "missing-learner"); err == nil {
		t.Fatal("missing accounts must fail")
	}
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	if err := seedAssignmentCommand(app, "teacher@example.test", "learner@example.test"); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	if err := seedAssignmentCommand(app, "teacher@example.test", "learner@example.test"); err == nil {
		t.Fatal("duplicate assignment must fail")
	}
	rows, err := app.FindAllRecords(schedulingstore.TeacherLearnersCollectionName)
	if err != nil || len(rows) != 1 {
		t.Fatalf("assignment count=%d err=%v", len(rows), err)
	}
	if !rows[0].GetBool(schedulingstore.ActiveField) || rows[0].Collection().Fields.GetByName(schedulingstore.DefaultDurationMinutesField) != nil {
		t.Fatalf("unexpected seed defaults: %v", rows[0].Original())
	}
}

func TestSeedAssignmentCommandIsDevelopmentOnly(t *testing.T) {
	t.Setenv("NUTKA_ENV", "production")
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	registerSeedAssignmentCommand(app)
	for _, command := range app.RootCmd.Commands() {
		if command.Name() == "seed-assignment" {
			t.Fatal("assignment seed command must not exist in production")
		}
	}
}

func findID(t *testing.T, app core.App, collection, email string) string {
	row, err := app.FindFirstRecordByData(collection, "email", email)
	if err != nil {
		t.Fatal(err)
	}
	return row.Id
}
