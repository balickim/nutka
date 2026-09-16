package commercialapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/database"
	_ "github.com/balickim/nutka/apps/backend/database/migrations"
	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestPurchaseCreatesFourTokensPaymentAndHistoryAtomically(t *testing.T) {
	app, teacher, assignment := packageTestApp(t)
	now := time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)
	event := packageRequest(t, app, teacher, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/packages", `{}`, true)
	if err := purchasePackage(event, func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}
	response := event.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	packages, err := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName)
	if err != nil || len(packages) != 1 {
		t.Fatalf("packages=%d err=%v", len(packages), err)
	}
	tokens, err := app.FindAllRecords(schedulingstore.PackageTokensCollectionName)
	if err != nil || len(tokens) != 4 {
		t.Fatalf("tokens=%d err=%v", len(tokens), err)
	}
	entries, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil || len(entries) != 1 || entries[0].GetString(schedulingstore.EntryTypeField) != "package_purchase" {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	events, err := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName)
	if err != nil || len(events) != 1 || events[0].GetString(schedulingstore.EventTypeField) != string(history.PackagePurchased) {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
}

func TestPurchaseRejectsUnknownFieldsWithoutCreatingState(t *testing.T) {
	app, teacher, assignment := packageTestApp(t)
	now := time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)
	event := packageRequest(t, app, teacher, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/packages", `{"unexpected":true}`, true)
	if err := purchasePackage(event, func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}
	response := event.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status=%d body=%s", response.Code, response.Body.String())
	}
	packages, _ := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName)
	entries, _ := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if len(packages) != 0 || len(entries) != 0 {
		t.Fatalf("strict decode left partial state packages=%d entries=%d", len(packages), len(entries))
	}
}

func TestPackageMutationRequiresAuthenticatedTeacher(t *testing.T) {
	app, teacher, assignment := packageTestApp(t)
	event := packageRequest(t, app, teacher, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/packages", `{}`, true)
	event.Auth = nil
	if err := purchasePackage(event, func() time.Time { return time.Now() }); err != nil {
		t.Fatal(err)
	}
	response := event.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestPurchaseConversionFailureRollsBackPackageAndPayment(t *testing.T) {
	app, teacher, assignment := packageTestApp(t)
	now := time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)
	event := packageRequest(t, app, teacher, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/packages", `{"convert_lesson_ids":["missing"]}`, true)
	if err := purchasePackage(event, func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}
	response := event.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusConflict {
		t.Fatalf("conversion failure status=%d body=%s", response.Code, response.Body.String())
	}
	packages, _ := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName)
	entries, _ := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if len(packages) != 0 || len(entries) != 0 {
		t.Fatalf("conversion failure left partial state packages=%d entries=%d", len(packages), len(entries))
	}
}

func TestReservePackageTokenCannotOverReserveConcurrentRequests(t *testing.T) {
	app, teacher, assignment := packageTestApp(t)
	now := time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)
	purchase := packageRequest(t, app, teacher, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/packages", `{}`, true)
	if err := purchasePackage(purchase, func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}
	packageRow, err := app.FindFirstRecordByData(schedulingstore.LessonPackagesCollectionName, schedulingstore.AssignmentField, assignment.Id)
	if err != nil {
		t.Fatal(err)
	}
	lessons := seedPackageLessons(t, app, assignment, now, 5)
	results := make(chan error, len(lessons))
	var wait sync.WaitGroup
	for _, lesson := range lessons {
		wait.Add(1)
		go func(lesson *core.Record) {
			defer wait.Done()
			_, reserveErr := ReservePackageToken(app, TokenReservationCommand{PackageID: packageRow.Id, LessonID: lesson.Id, AssignmentID: assignment.Id, Now: now, Actor: history.Actor{Role: history.TeacherActor, ID: teacher.Id}})
			results <- reserveErr
		}(lesson)
	}
	wait.Wait()
	close(results)
	successes := 0
	for reserveErr := range results {
		if reserveErr == nil {
			successes++
		}
	}
	if successes != 4 {
		t.Fatalf("successful reservations=%d", successes)
	}
	packageLessons := 0
	for _, lesson := range lessons {
		stored, findErr := app.FindRecordById(schedulingstore.LessonsCollectionName, lesson.Id)
		if findErr != nil {
			t.Fatal(findErr)
		}
		if stored.GetString(schedulingstore.PlanTypeField) == "package" {
			packageLessons++
			if stored.GetInt(schedulingstore.UnitPriceMinorField) != 0 {
				t.Fatalf("package lesson %s retained a per-lesson price", stored.Id)
			}
		}
	}
	if packageLessons != 4 {
		t.Fatalf("package lesson count=%d", packageLessons)
	}
	rows, err := app.FindAllRecords(schedulingstore.PackageTokensCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.GetString(schedulingstore.TokenStateField) == "available" {
			t.Fatal("a token remained available after four successful reservations")
		}
	}
}

func packageTestApp(t *testing.T) (*pocketbase.PocketBase, *core.Record, *core.Record) {
	t.Helper()
	_, source, _, _ := runtime.Caller(0)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(filepath.Join(filepath.Dir(source), "../..")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir(), DefaultDev: false})
	database.RegisterMigrate(app)
	schedulingstore.RegisterHooks(app)
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if err := app.RunAppMigrations(); err != nil {
		t.Fatal(err)
	}
	teachers, _ := app.FindCollectionByNameOrId(authconfig.TeachersCollectionName)
	teacher := core.NewRecord(teachers)
	teacher.SetEmail("teacher@example.test")
	teacher.SetPassword("local-password")
	teacher.SetVerified(true)
	teacher.Set("name", "Teacher")
	teacher.Set("timezone", "UTC")
	if err := app.Save(teacher); err != nil {
		t.Fatal(err)
	}
	learners, _ := app.FindCollectionByNameOrId(authconfig.LearnersCollectionName)
	learner := core.NewRecord(learners)
	learner.SetEmail("learner@example.test")
	learner.SetPassword("local-password")
	learner.SetVerified(true)
	learner.Set("name", "Learner")
	if err := app.Save(learner); err != nil {
		t.Fatal(err)
	}
	assignments, _ := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	assignment := core.NewRecord(assignments)
	assignment.Set("teacher", teacher.Id)
	assignment.Set("learner", learner.Id)
	if err := app.Save(assignment); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app, teacher, assignment
}

func packageRequest(t *testing.T, app *pocketbase.PocketBase, auth *core.Record, method, path, body string, intent bool) *core.RequestEvent {
	t.Helper()
	token, err := auth.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if intent {
		request.Header.Set(authconfig.AuthIntentHeader, authconfig.AuthIntentValue)
	}
	request.AddCookie(&http.Cookie{Name: sessioncookie.TeacherName, Value: token})
	parts := strings.Split(path, "/")
	if len(parts) > 4 {
		request.SetPathValue("id", parts[4])
	}
	return &core.RequestEvent{App: app, Auth: auth, Event: router.Event{Request: request, Response: httptest.NewRecorder()}}
}

func seedPackageLessons(t *testing.T, app *pocketbase.PocketBase, assignment *core.Record, now time.Time, count int) []*core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	result := make([]*core.Record, count)
	for index := range result {
		lesson := core.NewRecord(collection)
		start := now.Add(time.Duration(index+1) * time.Hour)
		lesson.Set("teacher", assignment.GetString("teacher"))
		lesson.Set("learner", assignment.GetString("learner"))
		lesson.Set(schedulingstore.AssignmentField, assignment.Id)
		lesson.Set(schedulingstore.StartAtField, start.Format(time.RFC3339))
		lesson.Set(schedulingstore.EndAtField, start.Add(45*time.Minute).Format(time.RFC3339))
		lesson.Set(schedulingstore.DurationMinutesField, 45)
		lesson.Set(schedulingstore.StatusField, "scheduled")
		lesson.Set(schedulingstore.PlanTypeField, "ad_hoc")
		lesson.Set(schedulingstore.OriginalLocalDateField, "2026-09-13")
		lesson.Set(schedulingstore.PolicyVersionField, "v1")
		lesson.Set(schedulingstore.PolicySnapshotField, businesspolicy.CurrentSnapshot())
		lesson.Set(schedulingstore.UnitPriceMinorField, 8000)
		lesson.Set(schedulingstore.CurrencyField, "PLN")
		lesson.Set(schedulingstore.ScheduleStateField, "scheduled")
		if err := app.Save(lesson); err != nil {
			t.Fatal(err)
		}
		result[index] = lesson
	}
	return result
}
