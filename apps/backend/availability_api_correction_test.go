package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
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

	if response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"default_duration_minutes":50}`, teacherCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("invalid assignment duration: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, secondTeacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("cross-owner assignment update: %d %s", response.Code, response.Body.String())
	}

	rule := commitAvailabilityProposal(t, server, teacherCookie, `{"operation":"create","target":"recurring_rule","rule":{"weekday":1,"start_time":"09:00","end_time":"12:00"}}`, nil)
	ruleID := availabilityRecordID(t, rule, "availability_rule")
	for _, proposal := range []string{
		`{"operation":"update","target":"recurring_rule","id":"` + ruleID + `","rule":{}}`,
		`{"operation":"update","target":"recurring_rule","id":"` + ruleID + `","rule":{"start_time":"09:05"}}`,
		`{"operation":"update","target":"recurring_rule","id":"` + ruleID + `","rule":{"start_time":"12:00","end_time":"11:00"}}`,
	} {
		response := request(t, server, http.MethodPost, "/api/teachers/availability/preview", proposal, teacherCookie, true)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid preview accepted: %d %s", response.Code, response.Body.String())
		}
	}
	if response := request(t, server, http.MethodPost, "/api/teachers/availability/preview", `{"operation":"disable","target":"recurring_rule","id":"`+ruleID+`"}`, secondTeacherCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("cross-owner preview: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/teachers/availability/rules", `{}`, teacherCookie, true); response.Code != http.StatusNotFound {
		t.Fatalf("legacy direct write route remains: %d %s", response.Code, response.Body.String())
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
	disabled := commitAvailabilityProposal(t, server, teacherCookie, `{"operation":"create","target":"recurring_rule","rule":{"weekday":1,"start_time":"09:00","end_time":"10:00","enabled":false}}`, nil)
	if disabled.Code != http.StatusOK || !strings.Contains(disabled.Body.String(), `"enabled":false`) {
		t.Fatalf("explicit disabled rule changed: %d %s", disabled.Code, disabled.Body.String())
	}
	disabledID := availabilityRecordID(t, disabled, "availability_rule")
	storedDisabled, err := app.FindRecordById(schedulingstore.AvailabilityRulesCollectionName, disabledID)
	if err != nil || storedDisabled.GetBool(schedulingstore.EnabledField) {
		t.Fatalf("explicit disabled rule persisted enabled: %v", storedDisabled.Original())
	}
	enabled := commitAvailabilityProposal(t, server, teacherCookie, `{"operation":"create","target":"recurring_rule","rule":{"weekday":2,"start_time":"09:00","end_time":"10:00"}}`, nil)
	if enabled.Code != http.StatusOK || !strings.Contains(enabled.Body.String(), `"enabled":true`) {
		t.Fatalf("omitted enabled rule did not default true: %d %s", enabled.Code, enabled.Body.String())
	}
}

func TestCreateRuleRejectsNegativeClockHTTPWithoutSaving(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	for _, body := range []string{
		`{"operation":"create","target":"recurring_rule","rule":{"weekday":1,"start_time":"-1:00","end_time":"01:00"}}`,
		`{"operation":"create","target":"recurring_rule","rule":{"weekday":1,"start_time":"00:00","end_time":"00:-15"}}`,
	} {
		response := request(t, server, http.MethodPost, "/api/teachers/availability/preview", body, teacherCookie, true)
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

func TestAvailabilityExceptionSupportsPreviewedUpdateEnableDisableAndDelete(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	cookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	created := commitAvailabilityProposal(t, server, cookie, `{"operation":"create","target":"exception","exception":{"start_at":"2030-01-07T09:00:00Z","end_at":"2030-01-07T10:00:00Z","kind":"available"}}`, nil)
	id := availabilityRecordID(t, created, "availability_exception")
	for _, operation := range []string{"disable", "enable"} {
		response := commitAvailabilityProposal(t, server, cookie, `{"operation":"`+operation+`","target":"exception","id":"`+id+`"}`, nil)
		want := operation == "enable"
		row, err := app.FindRecordById(schedulingstore.AvailabilityExceptionsCollectionName, id)
		if response.Code != http.StatusOK || err != nil || row.GetBool(schedulingstore.EnabledField) != want {
			t.Fatalf("%s exception: status=%d enabled=%v err=%v body=%s", operation, response.Code, row.GetBool(schedulingstore.EnabledField), err, response.Body.String())
		}
	}
	updated := commitAvailabilityProposal(t, server, cookie, `{"operation":"update","target":"exception","id":"`+id+`","exception":{"note":"changed"}}`, nil)
	row, err := app.FindRecordById(schedulingstore.AvailabilityExceptionsCollectionName, id)
	if updated.Code != http.StatusOK || err != nil || row.GetString(schedulingstore.NoteField) != "changed" {
		t.Fatalf("update exception: %d %v %s", updated.Code, err, updated.Body.String())
	}
	deleted := commitAvailabilityProposal(t, server, cookie, `{"operation":"delete","target":"exception","id":"`+id+`"}`, nil)
	if _, err := app.FindRecordById(schedulingstore.AvailabilityExceptionsCollectionName, id); deleted.Code != http.StatusOK || err == nil {
		t.Fatalf("delete exception: %d err=%v body=%s", deleted.Code, err, deleted.Body.String())
	}
}

func TestAvailabilityExceptionConflictRollsBackPatch(t *testing.T) {
	now := time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	exception := commitAvailabilityProposal(t, server, teacherCookie, `{"operation":"create","target":"exception","exception":{"start_at":"2030-01-07T09:00:00Z","end_at":"2030-01-07T10:00:00Z","kind":"available","note":"keep"}}`, nil)
	exceptionID := availabilityRecordID(t, exception, "availability_exception")
	lessonCollection, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	lesson := core.NewRecord(lessonCollection)
	lesson.Set("teacher", teacherID)
	lesson.Set("learner", learnerID)
	lesson.Set("assignment", assignment.Id)
	lesson.Set(schedulingstore.StartAtField, "2030-01-07T09:15:00Z")
	lesson.Set(schedulingstore.EndAtField, "2030-01-07T10:00:00Z")
	lesson.Set(schedulingstore.DurationMinutesField, 45)
	lesson.Set(schedulingstore.StatusField, "scheduled")
	lesson.Set(schedulingstore.PlanTypeField, "ad_hoc")
	lesson.Set(schedulingstore.OriginalLocalDateField, "2030-01-07")
	lesson.Set(schedulingstore.OriginalStartAtField, "2030-01-07T09:15:00Z")
	lesson.Set(schedulingstore.PolicyVersionField, businesspolicy.CurrentVersion)
	lesson.Set(schedulingstore.PolicySnapshotField, businesspolicy.CurrentSnapshot())
	lesson.Set(schedulingstore.UnitPriceMinorField, businesspolicy.AdHocPriceMinor)
	lesson.Set(schedulingstore.CurrencyField, businesspolicy.CurrencyPLN)
	lesson.Set(schedulingstore.ScheduleStateField, "scheduled")
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	seedAdHocCharge(t, app, lesson, assignment.Id)
	proposal := `{"operation":"update","target":"exception","id":"` + exceptionID + `","exception":{"kind":"unavailable"}}`
	preview := request(t, server, http.MethodPost, "/api/teachers/availability/preview", proposal, teacherCookie, true)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), lesson.Id) {
		t.Fatalf("conflicting exception preview: %d %s", preview.Code, preview.Body.String())
	}
	stored, err := app.FindRecordById(schedulingstore.AvailabilityExceptionsCollectionName, exceptionID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString(schedulingstore.KindField) != "available" || stored.GetString(schedulingstore.NoteField) != "keep" || !stored.GetDateTime(schedulingstore.StartAtField).Time().Equal(time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("conflicting patch changed exception: %v", stored.Original())
	}
	version := availabilityPreviewVersion(t, preview)
	commitBody := fmt.Sprintf(`{"preview_version":%q,"proposal":%s,"resolutions":[]}`, version, proposal)
	incomplete := request(t, server, http.MethodPost, "/api/teachers/availability/commit", commitBody, teacherCookie, true)
	if incomplete.Code != http.StatusConflict || !strings.Contains(incomplete.Body.String(), `"code":"unresolved_obligations"`) {
		t.Fatalf("incomplete resolution: %d %s", incomplete.Code, incomplete.Body.String())
	}
	storedLesson, err := app.FindRecordById(schedulingstore.LessonsCollectionName, lesson.Id)
	if err != nil || storedLesson.GetString(schedulingstore.ScheduleStateField) != "scheduled" {
		t.Fatalf("incomplete commit changed lesson: %v", err)
	}
}

func TestAvailabilityCommitRejectsStalePreview(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	cookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	created := commitAvailabilityProposal(t, server, cookie, `{"operation":"create","target":"recurring_rule","rule":{"weekday":1,"start_time":"09:00","end_time":"12:00"}}`, nil)
	id := availabilityRecordID(t, created, "availability_rule")
	proposal := `{"operation":"update","target":"recurring_rule","id":"` + id + `","rule":{"end_time":"11:00"}}`
	preview := request(t, server, http.MethodPost, "/api/teachers/availability/preview", proposal, cookie, true)
	version := availabilityPreviewVersion(t, preview)
	commitAvailabilityProposal(t, server, cookie, `{"operation":"create","target":"exception","exception":{"start_at":"2030-01-07T09:00:00Z","end_at":"2030-01-07T10:00:00Z","kind":"available"}}`, nil)
	commitBody := fmt.Sprintf(`{"preview_version":%q,"proposal":%s,"resolutions":[]}`, version, proposal)
	stale := request(t, server, http.MethodPost, "/api/teachers/availability/commit", commitBody, cookie, true)
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), `"code":"stale_preview"`) {
		t.Fatalf("stale preview: %d %s", stale.Code, stale.Body.String())
	}
	rule, err := app.FindRecordById(schedulingstore.AvailabilityRulesCollectionName, id)
	if err != nil || rule.GetString(schedulingstore.EndTimeField) != "12:00" {
		t.Fatalf("stale commit changed rule: %v", err)
	}
}

func TestAvailabilityCommitRollsBackEarlierResolutionWhenLaterResolutionFails(t *testing.T) {
	now := time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	cookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	created := commitAvailabilityProposal(t, server, cookie, `{"operation":"create","target":"exception","exception":{"start_at":"2030-01-07T09:00:00Z","end_at":"2030-01-07T12:00:00Z","kind":"available"}}`, nil)
	exceptionID := availabilityRecordID(t, created, "availability_exception")
	first := seedScheduledLesson(t, app, assignment.Id, teacherID, learnerID, time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC))
	second := seedScheduledLesson(t, app, assignment.Id, teacherID, learnerID, time.Date(2030, 1, 7, 10, 0, 0, 0, time.UTC))
	seedAdHocCharge(t, app, first, assignment.Id)
	seedAdHocCharge(t, app, second, assignment.Id)
	proposal := `{"operation":"update","target":"exception","id":"` + exceptionID + `","exception":{"kind":"unavailable"}}`
	preview := request(t, server, http.MethodPost, "/api/teachers/availability/preview", proposal, cookie, true)
	version := availabilityPreviewVersion(t, preview)
	resolutions := fmt.Sprintf(`[{"lesson":%q,"action":"cancel"},{"lesson":%q,"action":"reschedule","replacement_start_at":"2030-01-07T11:00:00Z"}]`, first.Id, second.Id)
	commitBody := fmt.Sprintf(`{"preview_version":%q,"proposal":%s,"resolutions":%s}`, version, proposal, resolutions)
	failed := request(t, server, http.MethodPost, "/api/teachers/availability/commit", commitBody, cookie, true)
	if failed.Code != http.StatusBadRequest || !strings.Contains(failed.Body.String(), `"code":"invalid_resolution"`) {
		t.Fatalf("failed resolved commit: %d %s", failed.Code, failed.Body.String())
	}
	for _, lesson := range []*core.Record{first, second} {
		stored, err := app.FindRecordById(schedulingstore.LessonsCollectionName, lesson.Id)
		if err != nil || stored.GetString(schedulingstore.ScheduleStateField) != "scheduled" {
			t.Fatalf("partial lesson mutation survived rollback: %v", err)
		}
		charge, err := app.FindFirstRecordByData(schedulingstore.ChargesCollectionName, schedulingstore.SourceIDField, lesson.Id)
		if err != nil || charge.GetString(schedulingstore.SettlementStateField) != "pending_settlement" {
			t.Fatalf("partial charge mutation survived rollback: %v", err)
		}
	}
	storedException, err := app.FindRecordById(schedulingstore.AvailabilityExceptionsCollectionName, exceptionID)
	if err != nil || storedException.GetString(schedulingstore.KindField) != "available" {
		t.Fatalf("availability mutation survived rollback: %v", err)
	}
}

func TestAvailabilityDistantContractOmissionAndRestorationReconcileForecast(t *testing.T) {
	now := time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	cookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	start := time.Date(2030, 1, 21, 9, 0, 0, 0, time.UTC)
	created := commitAvailabilityProposal(t, server, cookie, fmt.Sprintf(`{"operation":"create","target":"recurring_rule","rule":{"weekday":%d,"start_time":"09:00","end_time":"12:00"}}`, start.In(mustLocation()).Weekday()), nil)
	ruleID := availabilityRecordID(t, created, "availability_rule")
	activation := request(t, server, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/contracts", fmt.Sprintf(`{"start_on":"2030-01-21","weekday":%d,"start_time":"10:00"}`, start.In(mustLocation()).Weekday()), cookie, true)
	if activation.Code != http.StatusCreated {
		t.Fatalf("activate distant contract: %d %s", activation.Code, activation.Body.String())
	}
	contractID := responseID(t, activation.Body.Bytes())
	disabled := commitAvailabilityProposal(t, server, cookie, `{"operation":"disable","target":"recurring_rule","id":"`+ruleID+`"}`, nil)
	if disabled.Code != http.StatusOK || !strings.Contains(disabled.Body.String(), `"effect":"omit"`) {
		t.Fatalf("distant omission: %d %s", disabled.Code, disabled.Body.String())
	}
	assertContractProjection(t, app, contractID, "omitted", "planned_omission", false)
	enabled := commitAvailabilityProposal(t, server, cookie, `{"operation":"enable","target":"recurring_rule","id":"`+ruleID+`"}`, nil)
	if enabled.Code != http.StatusOK || !strings.Contains(enabled.Body.String(), `"effect":"restore"`) {
		t.Fatalf("distant restoration: %d %s", enabled.Code, enabled.Body.String())
	}
	assertContractProjection(t, app, contractID, "scheduled", "billable", true)
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
	otherRule := commitAvailabilityProposal(t, server, otherTeacherCookie, `{"operation":"create","target":"recurring_rule","rule":{"weekday":1,"start_time":"09:00","end_time":"12:00"}}`, nil)
	if otherRule.Code != http.StatusOK {
		t.Fatalf("create inactive teacher rule: %d %s", otherRule.Code, otherRule.Body.String())
	}
	otherRuleID := availabilityRecordID(t, otherRule, "availability_rule")
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
	proposal := `{"operation":"create","target":"recurring_rule","rule":{"weekday":` + strconv.Itoa(int(start.In(mustLocation()).Weekday())) + `,"start_time":"09:00","end_time":"12:00"}}`
	response := commitAvailabilityProposal(t, server, cookie, proposal, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("create availability rule: %d %s", response.Code, response.Body.String())
	}
}

func commitAvailabilityProposal(t *testing.T, server http.Handler, cookie *http.Cookie, proposal string, resolutions []map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	preview := request(t, server, http.MethodPost, "/api/teachers/availability/preview", proposal, cookie, true)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview availability: %d %s", preview.Code, preview.Body.String())
	}
	var previewValue struct {
		PreviewVersion string `json:"preview_version"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &previewValue); err != nil || previewValue.PreviewVersion == "" {
		t.Fatalf("invalid availability preview: %v %s", err, preview.Body.String())
	}
	var proposalValue map[string]any
	if err := json.Unmarshal([]byte(proposal), &proposalValue); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"preview_version": previewValue.PreviewVersion, "proposal": proposalValue, "resolutions": resolutions})
	if err != nil {
		t.Fatal(err)
	}
	return request(t, server, http.MethodPost, "/api/teachers/availability/commit", string(body), cookie, true)
}

func createAvailabilityException(t *testing.T, server http.Handler, cookie *http.Cookie, start, end time.Time, kind string) *httptest.ResponseRecorder {
	t.Helper()
	proposal := `{"operation":"create","target":"exception","exception":{"start_at":"` + start.UTC().Format(time.RFC3339) + `","end_at":"` + end.UTC().Format(time.RFC3339) + `","kind":"` + kind + `"}}`
	return commitAvailabilityProposal(t, server, cookie, proposal, nil)
}

func availabilityPreviewVersion(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var value struct {
		PreviewVersion string `json:"preview_version"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &value) != nil || value.PreviewVersion == "" {
		t.Fatalf("invalid preview: %d %s", response.Code, response.Body.String())
	}
	return value.PreviewVersion
}

func seedAdHocCharge(t *testing.T, app core.App, lesson *core.Record, assignmentID string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(schedulingstore.ChargesCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	charge := core.NewRecord(collection)
	charge.Set(schedulingstore.AssignmentField, assignmentID)
	charge.Set(schedulingstore.SourceTypeField, "ad_hoc")
	charge.Set(schedulingstore.SourceIDField, lesson.Id)
	charge.Set(schedulingstore.OriginalAmountMinorField, businesspolicy.AdHocPriceMinor)
	charge.Set(schedulingstore.CurrentAmountMinorField, businesspolicy.AdHocPriceMinor)
	charge.Set(schedulingstore.CurrencyField, businesspolicy.CurrencyPLN)
	charge.Set(schedulingstore.SettlementStateField, "pending_settlement")
	if err := app.Save(charge); err != nil {
		t.Fatal(err)
	}
}

func assertContractProjection(t *testing.T, app core.App, contractID, state, billing string, positiveForecast bool) {
	t.Helper()
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, lesson := range lessons {
		if lesson.GetString(schedulingstore.ContractField) != contractID {
			continue
		}
		count++
		if lesson.GetString(schedulingstore.ScheduleStateField) != state || lesson.GetString(schedulingstore.BillingOutcomeField) != billing {
			t.Fatalf("contract lesson projection: %v", lesson.Original())
		}
	}
	if count == 0 {
		t.Fatal("contract has no occurrences")
	}
	months, err := app.FindAllRecords(schedulingstore.ContractMonthsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	positive := false
	for _, month := range months {
		if month.GetString(schedulingstore.ContractField) == contractID && month.GetInt(schedulingstore.ForecastAmountMinorField) > 0 {
			positive = true
		}
	}
	if positive != positiveForecast {
		t.Fatalf("positive forecast=%v want=%v", positive, positiveForecast)
	}
}

func availabilityRecordID(t *testing.T, response *httptest.ResponseRecorder, key string) string {
	t.Helper()
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	var record struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload[key], &record); err != nil || record.ID == "" {
		t.Fatalf("availability response has no %s id: %v %s", key, err, response.Body.String())
	}
	return record.ID
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
	lesson.Set(schedulingstore.PlanTypeField, "ad_hoc")
	lesson.Set(schedulingstore.OriginalLocalDateField, start.In(mustLocation()).Format("2006-01-02"))
	lesson.Set(schedulingstore.OriginalStartAtField, start.UTC().Format(time.RFC3339))
	lesson.Set(schedulingstore.PolicyVersionField, businesspolicy.CurrentVersion)
	lesson.Set(schedulingstore.PolicySnapshotField, businesspolicy.CurrentSnapshot())
	lesson.Set(schedulingstore.UnitPriceMinorField, businesspolicy.AdHocPriceMinor)
	lesson.Set(schedulingstore.CurrencyField, businesspolicy.CurrencyPLN)
	lesson.Set(schedulingstore.ScheduleStateField, "scheduled")
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
	if booking.Code != http.StatusConflict || !strings.Contains(booking.Body.String(), `"code":"lesson_conflict"`) {
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
