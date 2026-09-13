package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func TestAvailabilityAPIBoundariesRejectInvalidAndCrossOwnerMutations(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedNamedTeacher(t, app, "second-teacher@example.test")
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	secondTeacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "second-teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")

	if response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"default_duration_minutes":50}`, teacherCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_duration"`) {
		t.Fatalf("invalid assignment duration: %d %s", response.Code, response.Body.String())
	}
	storedAssignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, assignment.Id)
	if err != nil || storedAssignment.GetInt(schedulingstore.DefaultDurationMinutesField) != 45 {
		t.Fatalf("invalid assignment duration changed value: %v", storedAssignment.Original())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, secondTeacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("cross-owner assignment update: %d %s", response.Code, response.Body.String())
	}

	rule := request(t, server, http.MethodPost, "/api/teachers/availability/rules", `{"weekday":1,"start_time":"09:00","end_time":"12:00"}`, teacherCookie, true)
	if rule.Code != http.StatusCreated {
		t.Fatalf("create rule: %d %s", rule.Code, rule.Body.String())
	}
	ruleID := responseID(t, rule.Body.Bytes())
	if response := request(t, server, http.MethodPatch, "/api/teachers/availability/rules/"+ruleID, `{}`, teacherCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("empty rule update: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/availability/rules/"+ruleID, `{"start_time":"09:05"}`, teacherCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_grid"`) {
		t.Fatalf("off-grid rule update: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/availability/rules/"+ruleID, `{"start_time":"12:00","end_time":"11:00"}`, teacherCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("invalid range rule update: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/availability/rules/"+ruleID, `{"teacher":"spoofed"}`, teacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("rule identity spoof: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/availability/rules/"+ruleID, `{"enabled":false}`, secondTeacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("cross-owner rule update: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodDelete, "/api/teachers/availability/rules/"+ruleID, "", secondTeacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("cross-owner rule delete: %d %s", response.Code, response.Body.String())
	}

	exception := request(t, server, http.MethodPost, "/api/teachers/availability/exceptions", `{"start_at":"2030-01-07T09:00:00Z","end_at":"2030-01-07T10:00:00Z","kind":"available"}`, teacherCookie, true)
	if exception.Code != http.StatusCreated {
		t.Fatalf("create exception: %d %s", exception.Code, exception.Body.String())
	}
	exceptionID := responseID(t, exception.Body.Bytes())
	if response := request(t, server, http.MethodPatch, "/api/teachers/availability/exceptions/"+exceptionID, `{"teacher":"spoofed"}`, teacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("exception identity spoof: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/availability/exceptions/"+exceptionID, `{"note":"not-owner"}`, secondTeacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("cross-owner exception update: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodDelete, "/api/teachers/availability/exceptions/"+exceptionID, "", secondTeacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("cross-owner exception delete: %d %s", response.Code, response.Body.String())
	}

	if response := request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/slots?teacher=spoofed", "", learnerCookie, false); response.Code != http.StatusOK {
		t.Fatalf("undocumented teacher query changed canonical slots: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, "/api/learners/assignments/not-an-assignment/slots?assignment="+assignment.Id, "", learnerCookie, false); response.Code != http.StatusForbidden {
		t.Fatalf("assignment query fallback accepted: %d %s", response.Code, response.Body.String())
	}
}

func TestCreateRulePreservesExplicitDisabledValueAndDefaultsOmission(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	disabled := request(t, server, http.MethodPost, "/api/teachers/availability/rules", `{"weekday":1,"start_time":"09:00","end_time":"10:00","enabled":false}`, teacherCookie, true)
	if disabled.Code != http.StatusCreated || !strings.Contains(disabled.Body.String(), `"enabled":false`) {
		t.Fatalf("explicit disabled rule changed: %d %s", disabled.Code, disabled.Body.String())
	}
	disabledID := responseID(t, disabled.Body.Bytes())
	storedDisabled, err := app.FindRecordById(schedulingstore.AvailabilityRulesCollectionName, disabledID)
	if err != nil || storedDisabled.GetBool(schedulingstore.EnabledField) {
		t.Fatalf("explicit disabled rule persisted enabled: %v", storedDisabled.Original())
	}
	enabled := request(t, server, http.MethodPost, "/api/teachers/availability/rules", `{"weekday":2,"start_time":"09:00","end_time":"10:00"}`, teacherCookie, true)
	if enabled.Code != http.StatusCreated || !strings.Contains(enabled.Body.String(), `"enabled":true`) {
		t.Fatalf("omitted enabled rule did not default true: %d %s", enabled.Code, enabled.Body.String())
	}
}

func TestCreateRuleRejectsNegativeClockHTTPWithoutSaving(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	for _, body := range []string{
		`{"weekday":1,"start_time":"-1:00","end_time":"01:00"}`,
		`{"weekday":1,"start_time":"00:00","end_time":"00:-15"}`,
	} {
		response := request(t, server, http.MethodPost, "/api/teachers/availability/rules", body, teacherCookie, true)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("negative clock accepted: %d %s", response.Code, response.Body.String())
		}
	}
	rows, err := app.FindAllRecords(schedulingstore.AvailabilityRulesCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("negative rule persisted: %d", len(rows))
	}
}

func TestAvailabilityExceptionConflictRollsBackPatch(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	exception := request(t, server, http.MethodPost, "/api/teachers/availability/exceptions", `{"start_at":"2030-01-07T09:00:00Z","end_at":"2030-01-07T10:00:00Z","kind":"available","note":"keep"}`, teacherCookie, true)
	if exception.Code != http.StatusCreated {
		t.Fatalf("create exception: %d %s", exception.Code, exception.Body.String())
	}
	exceptionID := responseID(t, exception.Body.Bytes())
	lessonCollection, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	lesson := core.NewRecord(lessonCollection)
	lesson.Set("teacher", teacherID)
	lesson.Set("learner", learnerID)
	lesson.Set("assignment", assignment.Id)
	lesson.Set(schedulingstore.StartAtField, "2030-01-07T09:15:00Z")
	lesson.Set(schedulingstore.EndAtField, "2030-01-07T09:45:00Z")
	lesson.Set(schedulingstore.DurationMinutesField, 30)
	lesson.Set(schedulingstore.StatusField, "scheduled")
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	patch := request(t, server, http.MethodPatch, "/api/teachers/availability/exceptions/"+exceptionID, `{"kind":"unavailable"}`, teacherCookie, true)
	if patch.Code != http.StatusConflict || !strings.Contains(patch.Body.String(), `"code":"conflict"`) {
		t.Fatalf("conflicting exception patch: %d %s", patch.Code, patch.Body.String())
	}
	stored, err := app.FindRecordById(schedulingstore.AvailabilityExceptionsCollectionName, exceptionID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString(schedulingstore.KindField) != "available" || stored.GetString(schedulingstore.NoteField) != "keep" || !stored.GetDateTime(schedulingstore.StartAtField).Time().Equal(time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("conflicting patch changed exception: %v", stored.Original())
	}
}

func TestLearnerSlotsAndBookingUseTeacherAndLearnerConflictUnion(t *testing.T) {
	t.Run("teacher conflict", func(t *testing.T) {
		app, server := newTestServer(t)
		seedTestTeacher(t, app, true)
		seedTestLearner(t, app, true)
		seedNamedLearner(t, app, "other-learner@example.test")
		teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
		learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
		otherLearnerID := findID(t, app, authconfig.LearnersCollectionName, "other-learner@example.test")
		assignment := seedAssignment(t, app, teacherID, learnerID)
		otherAssignment := seedAssignment(t, app, teacherID, otherLearnerID)
		teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
		learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
		start := futureRuleStart()
		createAvailabilityRule(t, server, teacherCookie, start)
		seedScheduledLesson(t, app, otherAssignment.Id, teacherID, otherLearnerID, start)
		assertSlotConflict(t, server, learnerCookie, assignment.Id, start)
	})

	t.Run("learner conflict", func(t *testing.T) {
		app, server := newTestServer(t)
		seedTestTeacher(t, app, true)
		seedNamedTeacher(t, app, "other-teacher@example.test")
		seedTestLearner(t, app, true)
		teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
		otherTeacherID := findID(t, app, authconfig.TeachersCollectionName, "other-teacher@example.test")
		learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
		assignment := seedAssignment(t, app, teacherID, learnerID)
		otherAssignment := seedAssignment(t, app, otherTeacherID, learnerID)
		teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
		learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
		start := futureRuleStart()
		createAvailabilityRule(t, server, teacherCookie, start)
		seedScheduledLesson(t, app, otherAssignment.Id, otherTeacherID, learnerID, start)
		assertSlotConflict(t, server, learnerCookie, assignment.Id, start)
	})
}

func TestLearnerCalendarIsolatesAssignmentsLessonsAndInactiveTeacherAvailability(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedNamedTeacher(t, app, "other-teacher@example.test")
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other-learner@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	otherTeacherID := findID(t, app, authconfig.TeachersCollectionName, "other-teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	otherLearnerID := findID(t, app, authconfig.LearnersCollectionName, "other-learner@example.test")
	activeAssignment := seedAssignment(t, app, teacherID, learnerID)
	otherLearnerAssignment := seedAssignment(t, app, teacherID, otherLearnerID)
	inactiveAssignment := seedAssignment(t, app, otherTeacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	otherTeacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "other-teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	if response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+inactiveAssignment.Id, `{"active":false}`, otherTeacherCookie, true); response.Code != http.StatusOK {
		t.Fatalf("deactivate assignment: %d %s", response.Code, response.Body.String())
	}
	start := futureRuleStart()
	createAvailabilityRule(t, server, teacherCookie, start)
	otherRule := request(t, server, http.MethodPost, "/api/teachers/availability/rules", `{"weekday":1,"start_time":"09:00","end_time":"12:00"}`, otherTeacherCookie, true)
	if otherRule.Code != http.StatusCreated {
		t.Fatalf("create inactive teacher rule: %d %s", otherRule.Code, otherRule.Body.String())
	}
	otherRuleID := responseID(t, otherRule.Body.Bytes())
	seedScheduledLesson(t, app, otherLearnerAssignment.Id, teacherID, otherLearnerID, start)
	calendar := request(t, server, http.MethodGet, "/api/learners/calendar", "", learnerCookie, false)
	if calendar.Code != http.StatusOK {
		t.Fatalf("learner calendar: %d %s", calendar.Code, calendar.Body.String())
	}
	body := calendar.Body.String()
	if !strings.Contains(body, activeAssignment.Id) {
		t.Fatalf("active assignment missing: %s", body)
	}
	if strings.Contains(body, otherLearnerAssignment.Id) || strings.Contains(body, otherRuleID) {
		t.Fatalf("learner calendar leaked unrelated data or inactive availability: %s", body)
	}
}

func TestAssignmentResponsesIncludeAssociatedCounterpartNamesAndRemainPrivate(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedNamedTeacher(t, app, "other-teacher@example.test")
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other-learner@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	otherTeacherID := findID(t, app, authconfig.TeachersCollectionName, "other-teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	otherLearnerID := findID(t, app, authconfig.LearnersCollectionName, "other-learner@example.test")
	teacherLearner := seedAssignment(t, app, teacherID, learnerID)
	teacherOtherLearner := seedAssignment(t, app, teacherID, otherLearnerID)
	otherTeacherLearner := seedAssignment(t, app, otherTeacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	otherTeacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "other-teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	otherLearnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "other-learner@example.test")

	teacherAssignments := assignmentResponse(t, server, teacherCookie, "/api/teachers/assignments")
	if len(teacherAssignments) != 2 {
		t.Fatalf("teacher assignment count: %d", len(teacherAssignments))
	}
	assertAssignmentNames(t, teacherAssignments, teacherLearner.Id, "Test Teacher", "Test Learner")
	assertAssignmentNames(t, teacherAssignments, teacherOtherLearner.Id, "Test Teacher", "Other Learner")
	if body := request(t, server, http.MethodGet, "/api/teachers/assignments", "", teacherCookie, false).Body.String(); strings.Contains(body, "@example.test") {
		t.Fatalf("assignment response exposed email: %s", body)
	}

	if assignments := assignmentResponse(t, server, otherTeacherCookie, "/api/teachers/assignments"); len(assignments) != 1 || assignments[0].ID != otherTeacherLearner.Id {
		t.Fatalf("other teacher assignment isolation: %#v", assignments)
	}
	learnerAssignments := assignmentResponse(t, server, learnerCookie, "/api/learners/assignments")
	if len(learnerAssignments) != 2 {
		t.Fatalf("learner assignment count: %d", len(learnerAssignments))
	}
	assertAssignmentNames(t, learnerAssignments, teacherLearner.Id, "Test Teacher", "Test Learner")
	assertAssignmentNames(t, learnerAssignments, otherTeacherLearner.Id, "Second Teacher", "Test Learner")
	if assignments := assignmentResponse(t, server, otherLearnerCookie, "/api/learners/assignments"); len(assignments) != 1 || assignments[0].ID != teacherOtherLearner.Id {
		t.Fatalf("other learner assignment isolation: %#v", assignments)
	}
}

func TestAssignmentReadFailsInsteadOfReturningAnUnverifiedCounterpartName(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	unverified := seedNamedLearner(t, app, "unverified-learner@example.test")
	unverified.SetVerified(false)
	if err := app.Save(unverified); err != nil {
		t.Fatal(err)
	}
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	seedAssignment(t, app, teacherID, unverified.Id)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	response := request(t, server, http.MethodGet, "/api/teachers/assignments", "", teacherCookie, false)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"internal_error"`) || strings.Contains(response.Body.String(), "unverified-learner") {
		t.Fatalf("unverified counterpart response: %d %s", response.Code, response.Body.String())
	}
}

type assignmentResponseDTO struct {
	ID          string `json:"id"`
	TeacherName string `json:"teacher_name"`
	LearnerName string `json:"learner_name"`
}

func assignmentResponse(t *testing.T, server http.Handler, cookie *http.Cookie, path string) []assignmentResponseDTO {
	t.Helper()
	response := request(t, server, http.MethodGet, path, "", cookie, false)
	if response.Code != http.StatusOK {
		t.Fatalf("assignment response %s: %d %s", path, response.Code, response.Body.String())
	}
	var payload struct {
		Assignments []assignmentResponseDTO `json:"assignments"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Assignments
}

func assertAssignmentNames(t *testing.T, assignments []assignmentResponseDTO, id, teacherName, learnerName string) {
	t.Helper()
	for _, assignment := range assignments {
		if assignment.ID == id {
			if assignment.TeacherName != teacherName || assignment.LearnerName != learnerName {
				t.Fatalf("assignment %s names: teacher=%q learner=%q", id, assignment.TeacherName, assignment.LearnerName)
			}
			return
		}
	}
	t.Fatalf("assignment %s missing", id)
}

func seedNamedLearner(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(authconfig.LearnersCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.Set(authconfig.LearnerNameField, "Other Learner")
	record.SetPassword("local-password")
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func futureRuleStart() time.Time {
	zone, _ := time.LoadLocation("Europe/Warsaw")
	local := time.Now().In(zone).AddDate(0, 0, 2)
	return time.Date(local.Year(), local.Month(), local.Day(), 10, 0, 0, 0, zone).UTC()
}

func createAvailabilityRule(t *testing.T, server http.Handler, cookie *http.Cookie, start time.Time) {
	t.Helper()
	body := `{"weekday":` + strconv.Itoa(int(start.In(mustLocation()).Weekday())) + `,"start_time":"09:00","end_time":"12:00"}`
	response := request(t, server, http.MethodPost, "/api/teachers/availability/rules", body, cookie, true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create availability rule: %d %s", response.Code, response.Body.String())
	}
}

func mustLocation() *time.Location {
	location, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		panic(err)
	}
	return location
}

func seedScheduledLesson(t *testing.T, app core.App, assignment, teacher, learner string, start time.Time) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	lesson := core.NewRecord(collection)
	lesson.Set("teacher", teacher)
	lesson.Set("learner", learner)
	lesson.Set("assignment", assignment)
	lesson.Set(schedulingstore.StartAtField, start.Format("2006-01-02T15:04:05Z"))
	lesson.Set(schedulingstore.EndAtField, start.Add(45*time.Minute).Format("2006-01-02T15:04:05Z"))
	lesson.Set(schedulingstore.DurationMinutesField, 45)
	lesson.Set(schedulingstore.StatusField, "scheduled")
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	return lesson
}

func assertSlotConflict(t *testing.T, server http.Handler, learnerCookie *http.Cookie, assignment string, start time.Time) {
	t.Helper()
	path := "/api/learners/assignments/" + assignment + "/slots"
	slots := request(t, server, http.MethodGet, path, "", learnerCookie, false)
	if slots.Code != http.StatusOK || strings.Contains(slots.Body.String(), start.Format("2006-01-02T15:04:05Z")) {
		t.Fatalf("conflicting slot returned: %d %s", slots.Code, slots.Body.String())
	}
	booking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment+"/book", `{"start_at":"`+start.Format("2006-01-02T15:04:05Z")+`"}`, learnerCookie, true)
	if booking.Code != http.StatusConflict || !strings.Contains(booking.Body.String(), `"code":"conflict"`) {
		t.Fatalf("conflicting booking: %d %s", booking.Code, booking.Body.String())
	}
}

func seedNamedTeacher(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(authconfig.TeachersCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.Set(authconfig.TeacherNameField, "Second Teacher")
	record.SetPassword("local-password")
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func responseID(t *testing.T, body []byte) string {
	t.Helper()
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.ID == "" {
		t.Fatalf("response has no id: %s", body)
	}
	return payload.ID
}
