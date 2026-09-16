// Verifies assignment duration removal and transactional deactivation guardrails at the HTTP boundary.
package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func TestAssignmentAPIUsesFixedDurationContract(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")

	response := request(t, server, http.MethodGet, "/api/teachers/assignments", "", teacherCookie, false)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "default_duration_minutes") {
		t.Fatalf("assignment response exposes duration: %d %s", response.Code, response.Body.String())
	}
	response = request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"default_duration_minutes":60}`, teacherCookie, true)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("duration override accepted: %d %s", response.Code, response.Body.String())
	}
}

func TestAssignmentDeactivationBlocksFutureLessonAndRetainsResolvedHistory(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")

	lesson := seedScheduledLesson(t, app, assignment.Id, teacherID, learnerID, time.Now().UTC().Add(24*time.Hour))
	response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, teacherCookie, true)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"unresolved_obligation"`) {
		t.Fatalf("future lesson did not block deactivation: %d %s", response.Code, response.Body.String())
	}
	stored, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, assignment.Id)
	if err != nil || !stored.GetBool(schedulingstore.ActiveField) {
		t.Fatalf("blocked deactivation changed assignment: %v", err)
	}

	lesson.Set(schedulingstore.StatusField, "cancelled")
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	response = request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, teacherCookie, true)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"active":false`) {
		t.Fatalf("resolved assignment did not deactivate: %d %s", response.Code, response.Body.String())
	}
}

func TestAssignmentDeactivationHonorsNoticeContractEffectiveEnd(t *testing.T) {
	now := time.Date(2030, time.January, 2, 12, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	contracts, err := app.FindCollectionByNameOrId(schedulingstore.RegularContractsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	contract := core.NewRecord(contracts)
	contract.Set(schedulingstore.AssignmentField, assignment.Id)
	contract.Set(schedulingstore.StatusField, "notice_given")
	contract.Set(schedulingstore.StartOnField, "2030-01-01")
	contract.Set(schedulingstore.EndOnField, "2030-06-30")
	contract.Set(schedulingstore.EffectiveEndOnField, "2030-01-10")
	contract.Set(schedulingstore.WeekdayField, 2)
	contract.Set(schedulingstore.StartTimeField, "09:00")
	contract.Set(schedulingstore.UnitPriceMinorField, businesspolicy.RegularLessonPriceMinor)
	contract.Set(schedulingstore.CurrencyField, businesspolicy.CurrencyPLN)
	contract.Set(schedulingstore.PolicyVersionField, businesspolicy.CurrentVersion)
	contract.Set(schedulingstore.PolicySnapshotField, businesspolicy.CurrentSnapshot())
	if err := app.Save(contract); err != nil {
		t.Fatal(err)
	}
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, teacherCookie, true)
	if response.Code != http.StatusConflict {
		t.Fatalf("notice contract before effective end did not block: %d %s", response.Code, response.Body.String())
	}
	contract.Set(schedulingstore.EffectiveEndOnField, "2030-01-01")
	if err := app.Save(contract); err != nil {
		t.Fatal(err)
	}
	response = request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, teacherCookie, true)
	if response.Code != http.StatusOK {
		t.Fatalf("notice contract after effective end blocked: %d %s", response.Code, response.Body.String())
	}
}

func TestTeacherTimezoneChangeIsBlockedUntilFutureLessonsAreResolved(t *testing.T) {
	app, _ := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedAssignment(t, app, teacherID, learnerID)
	lesson := seedScheduledLesson(t, app, assignment.Id, teacherID, learnerID, time.Now().UTC().Add(48*time.Hour))
	teacher, err := app.FindRecordById(authconfig.TeachersCollectionName, teacherID)
	if err != nil {
		t.Fatal(err)
	}
	teacher.Set(schedulingstore.TeacherTimezoneField, "UTC")
	if err := app.Save(teacher); !errors.Is(err, schedulingstore.ErrTimezoneChangeBlocked) {
		t.Fatalf("future lesson did not block timezone change: %v", err)
	}
	lesson.Set(schedulingstore.StatusField, "cancelled")
	lesson.Set(schedulingstore.ScheduleStateField, "cancelled")
	if err := app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	teacher, err = app.FindRecordById(authconfig.TeachersCollectionName, teacherID)
	if err != nil {
		t.Fatal(err)
	}
	teacher.Set(schedulingstore.TeacherTimezoneField, "UTC")
	if err := app.Save(teacher); err != nil {
		t.Fatalf("resolved lesson still blocked timezone change: %v", err)
	}
}
