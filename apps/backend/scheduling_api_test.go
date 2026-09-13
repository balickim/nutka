// Verifies scheduling API authorization, DTO boundaries, defaults, and UTC exception behavior.
package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
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
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"default_duration_minutes":60`) {
		t.Fatalf("teacher update: %d %s", response.Code, response.Body.String())
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

func TestSchedulingAvailabilityReturnsUTCAndRejectsBlockConflict(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	rule := request(t, server, http.MethodPost, "/api/teachers/availability/rules", `{"weekday":1,"start_time":"09:00","end_time":"12:00"}`, teacherCookie, true)
	if rule.Code != http.StatusCreated || !strings.Contains(rule.Body.String(), `"start_time":"09:00"`) {
		t.Fatalf("create rule: %d %s", rule.Code, rule.Body.String())
	}
	exception := request(t, server, http.MethodPost, "/api/teachers/availability/exceptions", `{"start_at":"2026-01-05T10:00:00+01:00","end_at":"2026-01-05T11:00:00+01:00","kind":"available"}`, teacherCookie, true)
	if exception.Code != http.StatusCreated || !strings.Contains(exception.Body.String(), `"start_at":"2026-01-05T09:00:00Z"`) {
		t.Fatalf("UTC exception: %d %s", exception.Code, exception.Body.String())
	}
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	lessonCollection, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	lesson := core.NewRecord(lessonCollection)
	lesson.Set("teacher", teacherID)
	lesson.Set("learner", learnerID)
	lesson.Set("assignment", assignment.Id)
	lesson.Set("start_at", "2026-01-05T10:15:00Z")
	lesson.Set("end_at", "2026-01-05T11:00:00Z")
	lesson.Set("duration_minutes", 45)
	lesson.Set("status", "scheduled")
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	blocked := request(t, server, http.MethodPost, "/api/teachers/availability/exceptions", `{"start_at":"2026-01-05T10:30:00Z","end_at":"2026-01-05T11:30:00Z","kind":"unavailable"}`, teacherCookie, true)
	if blocked.Code != http.StatusConflict || !strings.Contains(blocked.Body.String(), `"code":"conflict"`) {
		t.Fatalf("block conflict: %d %s", blocked.Code, blocked.Body.String())
	}
	bufferOnly := request(t, server, http.MethodPost, "/api/teachers/availability/exceptions", `{"start_at":"2026-01-05T11:00:00Z","end_at":"2026-01-05T11:05:00Z","kind":"unavailable"}`, teacherCookie, true)
	if bufferOnly.Code != http.StatusCreated {
		t.Fatalf("buffer-only block must be allowed: %d %s", bufferOnly.Code, bufferOnly.Body.String())
	}
}

func TestSeedAssignmentRequiresAccountsAndRejectsDuplicate(t *testing.T) {
	t.Setenv("NUTKA_ENV", "development")
	app, _ := newTestServer(t)
	if err := seedAssignmentCommand(app, "missing-teacher", "missing-learner", 0); err == nil {
		t.Fatal("missing accounts must fail")
	}
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	if err := seedAssignmentCommand(app, "teacher@example.test", "learner@example.test", 0); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	if err := seedAssignmentCommand(app, "teacher@example.test", "learner@example.test", 0); err == nil {
		t.Fatal("duplicate assignment must fail")
	}
	rows, err := app.FindAllRecords(schedulingstore.TeacherLearnersCollectionName)
	if err != nil || len(rows) != 1 {
		t.Fatalf("assignment count=%d err=%v", len(rows), err)
	}
	if rows[0].GetInt(schedulingstore.DefaultDurationMinutesField) != 45 || !rows[0].GetBool(schedulingstore.ActiveField) {
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
