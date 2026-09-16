package main

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestLessonBookingLifecycleRetainsEventsAndAttribution(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	zone, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Fatal(err)
	}
	local := time.Now().In(zone).AddDate(0, 0, 1)
	for local.Weekday() != time.Monday {
		local = local.AddDate(0, 0, 1)
	}
	start := time.Date(local.Year(), local.Month(), local.Day(), 10, 0, 0, 0, zone).UTC()
	createAvailabilityRule(t, server, teacherCookie, start)

	booking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true)
	if booking.Code != http.StatusCreated || !strings.Contains(booking.Body.String(), `"duration_minutes":45`) {
		t.Fatalf("booking failed: %d %s", booking.Code, booking.Body.String())
	}
	lesson := firstLesson(t, app)
	if events := countEvents(t, app, lesson.Id); events != 1 {
		t.Fatalf("creation event count: %d", events)
	}
	created := lessonEvents(t, app, lesson.Id)[0]
	if created.GetString(schedulingstore.KindField) != "created" || created.GetString(schedulingstore.InitiatorRoleField) != "learner" || created.GetDateTime(schedulingstore.NewStartAtField).Time().UTC() != start {
		t.Fatalf("creation snapshot: %v", created.Original())
	}
	if empty := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lesson.Id+"/reschedule", `{}`, teacherCookie, true); empty.Code != http.StatusBadRequest || countEvents(t, app, lesson.Id) != 1 {
		t.Fatalf("empty teacher reschedule changed history: %d %s", empty.Code, empty.Body.String())
	}
	if inactive := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, teacherCookie, true); inactive.Code != http.StatusConflict || !strings.Contains(inactive.Body.String(), `"code":"unresolved_obligation"`) {
		t.Fatalf("future lesson did not block deactivation: %d %s", inactive.Code, inactive.Body.String())
	}

	newStart := start.Add(15 * time.Minute)
	reschedule := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lesson.Id+"/reschedule", `{"start_at":"`+newStart.Format(time.RFC3339)+`"}`, teacherCookie, true)
	if reschedule.Code != http.StatusOK || !strings.Contains(reschedule.Body.String(), `"duration_minutes":45`) {
		t.Fatalf("reschedule failed: %d %s", reschedule.Code, reschedule.Body.String())
	}
	if events := countEvents(t, app, lesson.Id); events != 2 {
		t.Fatalf("reschedule event count: %d", events)
	}
	rescheduled := lessonEvents(t, app, lesson.Id)[1]
	if rescheduled.GetDateTime(schedulingstore.PriorStartAtField).Time().UTC() != start || rescheduled.GetDateTime(schedulingstore.NewStartAtField).Time().UTC() != newStart || rescheduled.GetInt(schedulingstore.PriorDurationMinutesField) != 45 || rescheduled.GetInt(schedulingstore.NewDurationMinutesField) != 45 {
		t.Fatalf("reschedule snapshot: %v", rescheduled.Original())
	}

	cancel := request(t, server, http.MethodPost, "/api/learners/lessons/"+lesson.Id+"/cancel", "{}", learnerCookie, true)
	if cancel.Code != http.StatusOK || !strings.Contains(cancel.Body.String(), `"schedule_state":"cancelled"`) || !strings.Contains(cancel.Body.String(), `"cancellation_initiator_role":"learner"`) {
		t.Fatalf("cancellation failed: %d %s", cancel.Code, cancel.Body.String())
	}
	if events := countEvents(t, app, lesson.Id); events != 3 {
		t.Fatalf("cancellation event count: %d", events)
	}
	cancelled := lessonEvents(t, app, lesson.Id)[2]
	if cancelled.GetString(schedulingstore.KindField) != "cancelled" || cancelled.GetString(schedulingstore.InitiatorIDField) != learnerID || cancelled.GetDateTime(schedulingstore.PriorStartAtField).Time().UTC() != newStart || !cancelled.GetDateTime(schedulingstore.NewStartAtField).Time().IsZero() {
		t.Fatalf("cancellation snapshot: %v", cancelled.Original())
	}
}

func TestLearnerAdHocCancellationUpgradesLegacyZeroAmountSchema(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := now.Add(48 * time.Hour)
	if response := createAvailabilityException(t, server, teacherCookie, start, start.Add(time.Hour), "available"); response.Code != http.StatusOK {
		t.Fatalf("availability: %d %s", response.Code, response.Body.String())
	}
	booking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true)
	if booking.Code != http.StatusCreated {
		t.Fatalf("booking: %d %s", booking.Code, booking.Body.String())
	}
	lesson := firstLesson(t, app)

	for collectionName, fieldName := range map[string]string{
		schedulingstore.ChargesCollectionName:          schedulingstore.CurrentAmountMinorField,
		schedulingstore.FinancialEntriesCollectionName: schedulingstore.AmountMinorField,
	} {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			t.Fatal(err)
		}
		collection.Fields.GetByName(fieldName).(*core.NumberField).Required = true
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := app.DB().NewQuery("DELETE FROM {{_migrations}} WHERE [[file]] = {:file}").Bind(dbx.Params{
		"file": "1788700000_allow_zero_commercial_amounts.go",
	}).Execute(); err != nil {
		t.Fatal(err)
	}
	if err := app.RunAppMigrations(); err != nil {
		t.Fatal(err)
	}

	cancellation := request(t, server, http.MethodPost, "/api/learners/lessons/"+lesson.Id+"/cancel", `{}`, learnerCookie, true)
	if cancellation.Code != http.StatusOK {
		t.Fatalf("cancellation: %d %s", cancellation.Code, cancellation.Body.String())
	}
	charge, err := app.FindFirstRecordByData(schedulingstore.ChargesCollectionName, schedulingstore.SourceIDField, lesson.Id)
	if err != nil || charge.GetInt(schedulingstore.CurrentAmountMinorField) != 0 || charge.GetString(schedulingstore.SettlementStateField) != "not_applicable" {
		t.Fatalf("cancelled charge: %v %v", charge, err)
	}
}

func TestLifecyclePersonaRescheduleContracts(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := futureRuleStart()
	createAvailabilityRule(t, server, teacherCookie, start)
	booking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true)
	if booking.Code != http.StatusCreated {
		t.Fatalf("booking: %d %s", booking.Code, booking.Body.String())
	}
	lesson := firstLesson(t, app)
	newStart := start.Add(15 * time.Minute)
	if response := request(t, server, http.MethodPost, "/api/learners/lessons/"+lesson.Id+"/reschedule", `{"start_at":"`+newStart.Format(time.RFC3339)+`"}`, learnerCookie, true); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"duration_minutes":45`) {
		t.Fatalf("learner reschedule: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/learners/lessons/"+lesson.Id+"/reschedule", `{"start_at":"`+start.Add(30*time.Minute).Format(time.RFC3339)+`","duration_minutes":60}`, learnerCookie, true); response.Code != http.StatusBadRequest || countEvents(t, app, lesson.Id) != 2 {
		t.Fatalf("learner duration override changed state: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lesson.Id+"/reschedule", `{"duration_minutes":60}`, teacherCookie, true); response.Code != http.StatusBadRequest || countEvents(t, app, lesson.Id) != 2 {
		t.Fatalf("teacher duration-only override changed state: %d %s", response.Code, response.Body.String())
	}
	stored, _ := app.FindRecordById(schedulingstore.LessonsCollectionName, lesson.Id)
	if response := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lesson.Id+"/reschedule", `{"duration_minutes":50}`, teacherCookie, true); response.Code != http.StatusBadRequest || countEvents(t, app, lesson.Id) != 2 || stored.GetInt(schedulingstore.DurationMinutesField) != 45 {
		t.Fatalf("invalid teacher duration changed state: %d %s", response.Code, response.Body.String())
	}
}

func TestConcurrentBookingCreatesOneLessonAndEvent(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := futureRuleStart()
	createAvailabilityRule(t, server, teacherCookie, start)
	responses := make(chan int, 2)
	var wait sync.WaitGroup
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			response := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true)
			responses <- response.Code
		}()
	}
	wait.Wait()
	close(responses)
	successes, conflicts := 0, 0
	for code := range responses {
		switch code {
		case http.StatusCreated:
			successes++
		case http.StatusConflict:
			conflicts++
		}
	}
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	if successes != 1 || conflicts != 1 || len(lessons) != 1 || countEvents(t, app, lessons[0].Id) != 1 {
		t.Fatalf("concurrent booking state: successes=%d conflicts=%d lessons=%d", successes, conflicts, len(lessons))
	}
}

func TestBookingUsesInjectedClockForGridHorizonAndAvailabilityFit(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	available := createAvailabilityException(t, server, teacherCookie, time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 2, 10, 0, 0, 0, time.UTC), "available")
	if available.Code != http.StatusOK {
		t.Fatalf("available exception: %d %s", available.Code, available.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-02T09:07:00Z"}`, learnerCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_grid"`) {
		t.Fatalf("off-grid booking: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-15T09:15:00Z"}`, learnerCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"horizon"`) {
		t.Fatalf("horizon booking: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-02T09:30:00Z"}`, learnerCookie, true); response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"lesson_conflict"`) {
		t.Fatalf("availability-fit booking: %d %s", response.Code, response.Body.String())
	}
}

func TestFlexibleBookingHTTPAcceptsExactBoundsAndRequiresTeacherShortNoticeConfirmation(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	for _, start := range []time.Time{now.Add(time.Hour), now.Add(24 * time.Hour), now.Add(14 * 24 * time.Hour)} {
		createAvailabilityException(t, server, teacherCookie, start, start.Add(time.Hour), "available")
	}
	shortPath := "/api/teachers/assignments/" + assignment.Id + "/book"
	shortBody := `{"start_at":"` + now.Add(time.Hour).Format(time.RFC3339) + `"}`
	if response := request(t, server, http.MethodPost, shortPath, shortBody, teacherCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"short_notice_confirmation_required"`) {
		t.Fatalf("unconfirmed teacher short notice: %d %s", response.Code, response.Body.String())
	}
	confirmedBody := `{"start_at":"` + now.Add(time.Hour).Format(time.RFC3339) + `","confirm_short_notice":true}`
	if response := request(t, server, http.MethodPost, shortPath, confirmedBody, teacherCookie, true); response.Code != http.StatusCreated {
		t.Fatalf("confirmed teacher short notice: %d %s", response.Code, response.Body.String())
	}
	learnerPath := "/api/learners/assignments/" + assignment.Id + "/book"
	for _, start := range []time.Time{now.Add(24 * time.Hour), now.Add(14 * 24 * time.Hour)} {
		body := `{"start_at":"` + start.Format(time.RFC3339) + `"}`
		if response := request(t, server, http.MethodPost, learnerPath, body, learnerCookie, true); response.Code != http.StatusCreated {
			t.Fatalf("exact learner boundary %s: %d %s", start, response.Code, response.Body.String())
		}
	}
	beyondBody := `{"start_at":"` + now.Add(14*24*time.Hour+15*time.Minute).Format(time.RFC3339) + `","confirm_short_notice":true}`
	if response := request(t, server, http.MethodPost, shortPath, beyondBody, teacherCookie, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"horizon"`) {
		t.Fatalf("teacher horizon: %d %s", response.Code, response.Body.String())
	}
}

func TestLifecycleRejectsImmutableAndCancelledLessons(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	for _, fixture := range []struct {
		name   string
		start  time.Time
		status string
	}{
		{name: "past", start: time.Now().UTC().Add(-2 * time.Hour), status: "scheduled"},
		{name: "started", start: time.Now().UTC().Add(-time.Minute), status: "scheduled"},
		{name: "cancelled", start: time.Now().UTC().Add(time.Hour), status: "cancelled"},
	} {
		lesson := seedScheduledLesson(t, app, assignment.Id, teacherID, learnerID, fixture.start)
		lesson.Set(schedulingstore.StatusField, fixture.status)
		lesson.Set(schedulingstore.ScheduleStateField, fixture.status)
		if err := app.Save(lesson); err != nil {
			t.Fatal(err)
		}
		beforeStart := lesson.GetDateTime(schedulingstore.StartAtField).Time()
		beforeEvents := countEvents(t, app, lesson.Id)
		for _, rolePath := range []string{"teachers", "learners"} {
			cookie := teacherCookie
			if rolePath == "learners" {
				cookie = learnerCookie
			}
			reschedule := request(t, server, http.MethodPost, "/api/"+rolePath+"/lessons/"+lesson.Id+"/reschedule", `{"start_at":"`+fixture.start.Add(time.Hour).Format(time.RFC3339)+`"}`, cookie, true)
			cancel := request(t, server, http.MethodPost, "/api/"+rolePath+"/lessons/"+lesson.Id+"/cancel", `{}`, cookie, true)
			if reschedule.Code != http.StatusConflict || cancel.Code != http.StatusConflict {
				t.Fatalf("%s %s mutation accepted: reschedule=%d cancel=%d", fixture.name, rolePath, reschedule.Code, cancel.Code)
			}
		}
		stored, err := app.FindRecordById(schedulingstore.LessonsCollectionName, lesson.Id)
		if err != nil || stored.GetDateTime(schedulingstore.StartAtField).Time() != beforeStart || countEvents(t, app, lesson.Id) != beforeEvents {
			t.Fatalf("%s mutation changed state: %v", fixture.name, stored.Original())
		}
	}
}

func TestCancelledIntervalCanBeReusedWithoutGenericCounters(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := futureRuleStart()
	createAvailabilityRule(t, server, teacherCookie, start)
	secondStart := start.Add(60 * time.Minute)
	firstBooking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true)
	secondBooking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+secondStart.Format(time.RFC3339)+`"}`, learnerCookie, true)
	if firstBooking.Code != http.StatusCreated || secondBooking.Code != http.StatusCreated {
		t.Fatalf("bookings: first=%d %s second=%d %s", firstBooking.Code, firstBooking.Body.String(), secondBooking.Code, secondBooking.Body.String())
	}
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil || len(lessons) != 2 {
		t.Fatalf("lessons: %v (%d)", err, len(lessons))
	}
	if response := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lessons[0].Id+"/cancel", `{}`, teacherCookie, true); response.Code != http.StatusOK {
		t.Fatalf("teacher cancellation: %d %s", response.Code, response.Body.String())
	}
	rebook := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true)
	if rebook.Code != http.StatusCreated {
		t.Fatalf("cancelled interval was not reusable: %d %s", rebook.Code, rebook.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lessons[1].Id+"/reschedule", `{"start_at":"`+secondStart.Add(15*time.Minute).Format(time.RFC3339)+`"}`, teacherCookie, true); response.Code != http.StatusOK {
		t.Fatalf("reschedule: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/learners/lessons/"+lessons[1].Id+"/cancel", `{}`, learnerCookie, true); response.Code != http.StatusOK {
		t.Fatalf("learner cancellation: %d %s", response.Code, response.Body.String())
	}
	teacherCalendar := request(t, server, http.MethodGet, "/api/teachers/calendar", "", teacherCookie, false)
	learnerCalendar := request(t, server, http.MethodGet, "/api/learners/calendar", "", learnerCookie, false)
	if teacherCalendar.Code != http.StatusOK || strings.Contains(teacherCalendar.Body.String(), `"cancellation_counters"`) {
		t.Fatalf("teacher calendar retained generic counters: %d %s", teacherCalendar.Code, teacherCalendar.Body.String())
	}
	if learnerCalendar.Code != http.StatusOK || strings.Contains(learnerCalendar.Body.String(), `"cancellation_counters"`) {
		t.Fatalf("learner calendar retained generic counters: %d %s", learnerCalendar.Code, learnerCalendar.Body.String())
	}
}

func TestLifecycleRejectsInactiveUnownedAndUnknownCancellationFields(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedNamedTeacher(t, app, "other-teacher@example.test")
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other-learner@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	otherTeacherID := findID(t, app, authconfig.TeachersCollectionName, "other-teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	otherLearnerID := findID(t, app, authconfig.LearnersCollectionName, "other-learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	otherAssignment := seedLessonAssignment(t, app, otherTeacherID, otherLearnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	otherTeacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "other-teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	otherLearnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "other-learner@example.test")
	start := futureRuleStart()
	createAvailabilityRule(t, server, teacherCookie, start)
	if response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":false}`, teacherCookie, true); response.Code != http.StatusOK {
		t.Fatalf("deactivate assignment: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("inactive assignment booking: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPost, "/api/learners/assignments/"+otherAssignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`"}`, learnerCookie, true); response.Code != http.StatusForbidden {
		t.Fatalf("unowned assignment booking: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPatch, "/api/teachers/assignments/"+assignment.Id, `{"active":true}`, teacherCookie, true); response.Code != http.StatusOK {
		t.Fatalf("reactivate assignment: %d %s", response.Code, response.Body.String())
	}
	lesson := seedScheduledLesson(t, app, assignment.Id, teacherID, learnerID, start)
	for _, fixture := range []struct {
		path   string
		cookie *http.Cookie
	}{
		{path: "teachers", cookie: otherTeacherCookie},
		{path: "learners", cookie: otherLearnerCookie},
	} {
		response := request(t, server, http.MethodPost, "/api/"+fixture.path+"/lessons/"+lesson.Id+"/reschedule", `{"start_at":"`+start.Add(15*time.Minute).Format(time.RFC3339)+`"}`, fixture.cookie, true)
		if response.Code != http.StatusForbidden {
			t.Fatalf("unrelated %s mutation: %d %s", fixture.path, response.Code, response.Body.String())
		}
	}
	if response := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lesson.Id+"/cancel", `{"foo":1}`, teacherCookie, true); response.Code != http.StatusBadRequest {
		t.Fatalf("unknown cancellation field: %d %s", response.Code, response.Body.String())
	}
}

func TestLearnerBookingRejectsDurationOverride(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := time.Now().UTC().Add(24 * time.Hour).Truncate(15 * time.Minute)
	response := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start.Format(time.RFC3339)+`","duration_minutes":60}`, learnerCookie, true)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"duration_override"`) {
		t.Fatalf("duration override accepted: %d %s", response.Code, response.Body.String())
	}
}

func TestLearnerBookingRejectsNullDurationAndIdentityFields(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := time.Now().UTC().Add(24 * time.Hour).Truncate(15 * time.Minute)
	for _, body := range []string{
		`{"start_at":"` + start.Format(time.RFC3339) + `","duration_minutes":null}`,
		`{"start_at":"` + start.Format(time.RFC3339) + `","learner":"other"}`,
	} {
		response := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", body, learnerCookie, true)
		if response.Code != http.StatusBadRequest && response.Code != http.StatusForbidden {
			t.Fatalf("booking accepted forbidden field: %d %s", response.Code, response.Body.String())
		}
	}
	if rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName); err != nil || len(rows) != 0 {
		t.Fatalf("forbidden booking wrote lesson: %v (%d)", err, len(rows))
	}
}

func firstLesson(t *testing.T, app core.App) *core.Record {
	t.Helper()
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil || len(rows) != 1 {
		t.Fatalf("lessons: %v (%d)", err, len(rows))
	}
	return rows[0]
}

func seedLessonAssignment(t *testing.T, app core.App, teacher, learner string) *core.Record {
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

func countEvents(t *testing.T, app core.App, lessonID string) int {
	t.Helper()
	rows, err := app.FindAllRecords(schedulingstore.LessonEventsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, row := range rows {
		if row.GetString(schedulingstore.LessonField) == lessonID {
			count++
		}
	}
	return count
}

func lessonEvents(t *testing.T, app core.App, lessonID string) []*core.Record {
	t.Helper()
	rows, err := app.FindAllRecords(schedulingstore.LessonEventsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	result := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if row.GetString(schedulingstore.LessonField) == lessonID {
			result = append(result, row)
		}
	}
	return result
}
