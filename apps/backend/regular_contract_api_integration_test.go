// Verifies registered regular-contract routes, materialized series, ownership, notice, and price changes against PocketBase.
package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func TestRegularContractRoutesPersistAndPartitionACompleteSeries(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := time.Date(2030, time.January, 7, 9, 0, 0, 0, time.UTC)
	createAvailabilityRule(t, server, teacherCookie, start)

	body := fmt.Sprintf(`{"start_on":"2030-01-07","weekday":%d,"start_time":"10:00"}`, start.In(mustLocation()).Weekday())
	activation := request(t, server, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/contracts", body, teacherCookie, true)
	if activation.Code != http.StatusCreated || !strings.Contains(activation.Body.String(), `"end_on":"2030-06-30"`) {
		t.Fatalf("activate contract: %d %s", activation.Code, activation.Body.String())
	}
	contractID := responseID(t, activation.Body.Bytes())
	months, err := app.FindAllRecords(schedulingstore.ContractMonthsCollectionName)
	if err != nil || len(months) != 6 {
		t.Fatalf("contract month projections=%d err=%v", len(months), err)
	}
	charges, err := app.FindAllRecords(schedulingstore.ChargesCollectionName)
	if err != nil || len(charges) != 1 || charges[0].GetString(schedulingstore.SourceIDField) != contractID || charges[0].GetString(schedulingstore.PeriodField) != "2030-01" {
		t.Fatalf("immediate current charge=%d err=%v", len(charges), err)
	}
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil || len(lessons) < 20 {
		t.Fatalf("materialized occurrences=%d err=%v", len(lessons), err)
	}
	for _, lesson := range lessons {
		if lesson.GetString(schedulingstore.ContractField) != contractID || lesson.GetString(schedulingstore.PlanTypeField) != "regular_contract" || lesson.GetString(schedulingstore.BillingOutcomeField) != "billable" {
			t.Fatalf("invalid occurrence projection: %v", lesson.Original())
		}
	}
	paid := request(t, server, http.MethodPost, "/api/teachers/charges/"+charges[0].Id+"/payment", `{"settlement":"paid"}`, teacherCookie, true)
	if paid.Code != http.StatusOK || !strings.Contains(paid.Body.String(), `"settlement_state":"paid"`) {
		t.Fatalf("contract payment: %d %s", paid.Code, paid.Body.String())
	}
	firstOccurrence := contractLessonOn(t, lessons, "2030-01-07")
	cancelled := request(t, server, http.MethodPost, "/api/teachers/lessons/"+firstOccurrence.Id+"/cancel", `{}`, teacherCookie, true)
	if cancelled.Code != http.StatusOK {
		t.Fatalf("teacher contract cancellation: %d %s", cancelled.Code, cancelled.Body.String())
	}
	entries, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil || !hasFinancialEntry(entries, "credit_created", 5000) {
		t.Fatalf("paid cancellation credit missing: entries=%d err=%v", len(entries), err)
	}
	teacherSeries := request(t, server, http.MethodGet, "/api/teachers/contracts/"+contractID+"/series", "", teacherCookie, false)
	if teacherSeries.Code != http.StatusOK || !strings.Contains(teacherSeries.Body.String(), `"near_term"`) || !strings.Contains(teacherSeries.Body.String(), `"later"`) {
		t.Fatalf("teacher series: %d %s", teacherSeries.Code, teacherSeries.Body.String())
	}
	learnerContracts := request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/contracts", "", learnerCookie, false)
	if learnerContracts.Code != http.StatusOK || !strings.Contains(learnerContracts.Body.String(), contractID) {
		t.Fatalf("learner contract: %d %s", learnerContracts.Code, learnerContracts.Body.String())
	}
	blockedBooking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-08T10:00:00Z"}`, learnerCookie, true)
	if blockedBooking.Code != http.StatusBadRequest || !strings.Contains(blockedBooking.Body.String(), `"code":"plan_precedence"`) {
		t.Fatalf("contract precedence: %d %s", blockedBooking.Code, blockedBooking.Body.String())
	}
	amendment := request(t, server, http.MethodPost, "/api/teachers/contracts/"+contractID+"/amendments", `{"effective_on":"2030-03-01","price_minor":6000,"currency":"PLN"}`, teacherCookie, true)
	if amendment.Code != http.StatusOK {
		t.Fatalf("contract amendment: %d %s", amendment.Code, amendment.Body.String())
	}
	monthsRead := request(t, server, http.MethodGet, "/api/learners/contracts/"+contractID+"/months", "", learnerCookie, false)
	if monthsRead.Code != http.StatusOK || !strings.Contains(monthsRead.Body.String(), `"month":"2030-03"`) || !strings.Contains(monthsRead.Body.String(), `"forecast_amount_minor":24000`) || !strings.Contains(monthsRead.Body.String(), `"forecast":true`) {
		t.Fatalf("contract forecasts: %d %s", monthsRead.Code, monthsRead.Body.String())
	}
	notice := request(t, server, http.MethodPost, "/api/learners/contracts/"+contractID+"/notice", `{}`, learnerCookie, true)
	if notice.Code != http.StatusOK || !strings.Contains(notice.Body.String(), `"status":"notice_given"`) {
		t.Fatalf("learner notice: %d %s", notice.Code, notice.Body.String())
	}
}

func TestRegularContractManagementRoutesEnforceOwnershipAndRetainCorrections(t *testing.T) {
	now := time.Date(2030, time.May, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other-learner@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	otherLearnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "other-learner@example.test")
	start := time.Date(2030, time.May, 6, 9, 0, 0, 0, time.UTC)
	createAvailabilityRule(t, server, teacherCookie, start)
	body := fmt.Sprintf(`{"start_on":"2030-05-06","weekday":%d,"start_time":"10:00"}`, start.In(mustLocation()).Weekday())
	activation := request(t, server, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/contracts", body, teacherCookie, true)
	if activation.Code != http.StatusCreated {
		t.Fatalf("activate contract: %d %s", activation.Code, activation.Body.String())
	}
	contractID := responseID(t, activation.Body.Bytes())
	if forbidden := request(t, server, http.MethodPost, "/api/learners/contracts/"+contractID+"/notice", `{}`, otherLearnerCookie, true); forbidden.Code != http.StatusForbidden {
		t.Fatalf("unrelated learner notice: %d %s", forbidden.Code, forbidden.Body.String())
	}
	scheduleBody := fmt.Sprintf(`{"effective_on":"2030-06-01","weekday":%d,"start_time":"11:00"}`, start.In(mustLocation()).Weekday())
	changed := request(t, server, http.MethodPost, "/api/teachers/contracts/"+contractID+"/schedule", scheduleBody, teacherCookie, true)
	if changed.Code != http.StatusOK || !strings.Contains(changed.Body.String(), `"start_time":"11:00"`) {
		t.Fatalf("contract schedule: %d %s", changed.Code, changed.Body.String())
	}
	scheduleEvent := contractBusinessEvent(t, app, contractID, "contract_schedule_changed")
	corrected := request(t, server, http.MethodPost, "/api/teachers/contracts/"+contractID+"/correction", fmt.Sprintf(`{"event_id":%q,"reason":"audit correction"}`, scheduleEvent.Id), teacherCookie, true)
	if corrected.Code != http.StatusOK {
		t.Fatalf("contract correction: %d %s", corrected.Code, corrected.Body.String())
	}
	correction := contractBusinessEvent(t, app, contractID, "correction")
	if correction.GetString(schedulingstore.CorrectsEventField) != scheduleEvent.Id {
		t.Fatalf("contract correction relation=%q want=%q", correction.GetString(schedulingstore.CorrectsEventField), scheduleEvent.Id)
	}
	ended := request(t, server, http.MethodPost, "/api/teachers/contracts/"+contractID+"/early-end", `{"end_on":"2030-06-10","reason":"mutual agreement"}`, teacherCookie, true)
	if ended.Code != http.StatusOK || !strings.Contains(ended.Body.String(), `"status":"ended"`) {
		t.Fatalf("early end: %d %s", ended.Code, ended.Body.String())
	}
	renewed := request(t, server, http.MethodPost, "/api/teachers/contracts/"+contractID+"/renew", `{"start_on":"2030-07-01"}`, teacherCookie, true)
	if renewed.Code != http.StatusCreated || responseID(t, renewed.Body.Bytes()) == contractID {
		t.Fatalf("renew contract: %d %s", renewed.Code, renewed.Body.String())
	}
	learnerContracts := request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/contracts", "", learnerCookie, false)
	if learnerContracts.Code != http.StatusOK || strings.Count(learnerContracts.Body.String(), `"id"`) < 2 {
		t.Fatalf("learner contract list: %d %s", learnerContracts.Code, learnerContracts.Body.String())
	}
}

func TestContractLearnerAllowanceSurvivesReloadAndTeacherCorrection(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	start := time.Date(2030, time.January, 7, 9, 0, 0, 0, time.UTC)
	createAvailabilityRule(t, server, teacherCookie, start)
	for _, replacement := range []time.Time{time.Date(2030, 1, 8, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 9, 9, 0, 0, 0, time.UTC)} {
		createAvailabilityException(t, server, teacherCookie, replacement, replacement.Add(time.Hour), "available")
	}
	activation := request(t, server, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/contracts", fmt.Sprintf(`{"start_on":"2030-01-07","weekday":%d,"start_time":"10:00"}`, start.In(mustLocation()).Weekday()), teacherCookie, true)
	if activation.Code != http.StatusCreated {
		t.Fatalf("activate allowance contract: %d %s", activation.Code, activation.Body.String())
	}
	contractID := responseID(t, activation.Body.Bytes())
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	first := contractLessonOn(t, lessons, "2030-01-07")
	second := contractLessonOn(t, lessons, "2030-01-14")
	rescheduled := request(t, server, http.MethodPost, "/api/learners/lessons/"+first.Id+"/reschedule", `{"start_at":"2030-01-08T09:00:00Z"}`, learnerCookie, true)
	if rescheduled.Code != http.StatusOK {
		t.Fatalf("learner contract reschedule: %d %s", rescheduled.Code, rescheduled.Body.String())
	}
	contractRead := request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/contracts", "", learnerCookie, false)
	if contractRead.Code != http.StatusOK || !strings.Contains(contractRead.Body.String(), `"remaining_monthly_reschedules":0`) {
		t.Fatalf("persisted allowance: %d %s", contractRead.Code, contractRead.Body.String())
	}
	rejected := request(t, server, http.MethodPost, "/api/learners/lessons/"+second.Id+"/reschedule", `{"start_at":"2030-01-09T09:00:00Z"}`, learnerCookie, true)
	if rejected.Code != http.StatusConflict || !strings.Contains(rejected.Body.String(), `"code":"contract_allowance_exhausted"`) {
		t.Fatalf("exhausted contract reschedule: %d %s", rejected.Code, rejected.Body.String())
	}
	corrected := request(t, server, http.MethodPost, "/api/teachers/lessons/"+first.Id+"/correction", `{"correction":"restore_allowance","reason":"wrong learner request"}`, teacherCookie, true)
	if corrected.Code != http.StatusOK || !strings.Contains(corrected.Body.String(), `"start_at":"2030-01-07T09:00:00Z"`) {
		t.Fatalf("contract allowance correction: %d %s", corrected.Code, corrected.Body.String())
	}
	contractRead = request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/contracts", "", learnerCookie, false)
	if contractRead.Code != http.StatusOK || !strings.Contains(contractRead.Body.String(), contractID) || !strings.Contains(contractRead.Body.String(), `"remaining_monthly_reschedules":1`) {
		t.Fatalf("corrected allowance: %d %s", contractRead.Code, contractRead.Body.String())
	}
}

func contractLessonOn(t *testing.T, lessons []*core.Record, date string) *core.Record {
	t.Helper()
	for _, lesson := range lessons {
		if lesson.GetString(schedulingstore.OriginalLocalDateField) == date {
			return lesson
		}
	}
	t.Fatalf("contract occurrence %s missing", date)
	return nil
}

func hasFinancialEntry(rows []*core.Record, kind string, amount int) bool {
	for _, row := range rows {
		if row.GetString(schedulingstore.EntryTypeField) == kind && row.GetInt(schedulingstore.AmountMinorField) == amount {
			return true
		}
	}
	return false
}

func contractBusinessEvent(t *testing.T, app core.App, contractID, kind string) *core.Record {
	t.Helper()
	rows, err := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	for index := len(rows) - 1; index >= 0; index-- {
		if rows[index].GetString(schedulingstore.AggregateTypeField) == "contract" && rows[index].GetString(schedulingstore.AggregateIDField) == contractID && rows[index].GetString(schedulingstore.EventTypeField) == kind {
			return rows[index]
		}
	}
	t.Fatalf("contract event %s missing", kind)
	return nil
}
