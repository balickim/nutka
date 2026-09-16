// Verifies package precedence, token lifecycle, explicit outcomes, settlement, and registered commercial routes end to end.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
)

func TestPackageBookingAndCancellationTransitionsAreAtomic(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	available := createAvailabilityException(t, server, teacherCookie, time.Date(2030, 1, 3, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 3, 12, 0, 0, 0, time.UTC), "available")
	if available.Code != http.StatusOK {
		t.Fatalf("availability: %d %s", available.Code, available.Body.String())
	}
	purchase := request(t, server, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/packages", `{}`, teacherCookie, true)
	if purchase.Code != http.StatusCreated || !strings.Contains(purchase.Body.String(), `"price_minor":26000`) {
		t.Fatalf("purchase: %d %s", purchase.Code, purchase.Body.String())
	}
	forcedPlan := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-03T09:00:00Z","plan_type":"ad_hoc"}`, learnerCookie, true)
	if forcedPlan.Code != http.StatusBadRequest || !strings.Contains(forcedPlan.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("client plan override accepted: %d %s", forcedPlan.Code, forcedPlan.Body.String())
	}
	booking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-03T09:00:00Z"}`, learnerCookie, true)
	if booking.Code != http.StatusCreated || !strings.Contains(booking.Body.String(), `"plan_type":"package"`) || !strings.Contains(booking.Body.String(), `"unit_price_minor":0`) {
		slots := request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/slots", "", learnerCookie, false)
		t.Fatalf("package booking: %d %s slots=%d %s", booking.Code, booking.Body.String(), slots.Code, slots.Body.String())
	}
	lessonID := responseID(t, booking.Body.Bytes())
	cancel := request(t, server, http.MethodPost, "/api/learners/lessons/"+lessonID+"/cancel", `{}`, learnerCookie, true)
	if cancel.Code != http.StatusOK || !strings.Contains(cancel.Body.String(), `"plan_effect":"package_token_returned"`) {
		t.Fatalf("package cancellation: %d %s", cancel.Code, cancel.Body.String())
	}
	tokens, err := app.FindAllRecords(schedulingstore.PackageTokensCollectionName)
	if err != nil || len(tokens) != 4 || tokens[0].GetString(schedulingstore.TokenStateField) != "available" {
		t.Fatalf("token projection: %v %#v", err, tokens)
	}
	packageRow, err := app.FindFirstRecordByData(schedulingstore.LessonPackagesCollectionName, schedulingstore.AssignmentField, assignment.Id)
	if err != nil {
		t.Fatal(err)
	}
	originalValidity, err := time.Parse("2006-01-02", packageRow.GetString(schedulingstore.ValidThroughField))
	if err != nil {
		t.Fatal(err)
	}
	createAvailabilityException(t, server, teacherCookie, time.Date(2030, 1, 4, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 4, 10, 0, 0, 0, time.UTC), "available")
	secondBooking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-04T09:00:00Z"}`, learnerCookie, true)
	secondLessonID := responseID(t, secondBooking.Body.Bytes())
	teacherCancel := request(t, server, http.MethodPost, "/api/teachers/lessons/"+secondLessonID+"/cancel", `{}`, teacherCookie, true)
	if teacherCancel.Code != http.StatusOK || !strings.Contains(teacherCancel.Body.String(), `"plan_effect":"package_token_returned_and_extended"`) {
		t.Fatalf("teacher package cancellation: %d %s", teacherCancel.Code, teacherCancel.Body.String())
	}
	packageRow, err = app.FindRecordById(schedulingstore.LessonPackagesCollectionName, packageRow.Id)
	if err != nil || packageRow.GetString(schedulingstore.ValidThroughField) != originalValidity.AddDate(0, 0, 7).Format("2006-01-02") {
		t.Fatalf("teacher cancellation validity: %v %s", err, packageRow.GetString(schedulingstore.ValidThroughField))
	}
	createAvailabilityException(t, server, teacherCookie, time.Date(2030, 1, 5, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 5, 10, 0, 0, 0, time.UTC), "available")
	completedBooking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-05T09:00:00Z"}`, learnerCookie, true)
	completedLessonID := responseID(t, completedBooking.Body.Bytes())
	now = time.Date(2030, 1, 5, 10, 0, 0, 0, time.UTC)
	completed := request(t, server, http.MethodPost, "/api/teachers/lessons/"+completedLessonID+"/outcome", `{"outcome":"completed"}`, teacherCookie, true)
	if completed.Code != http.StatusOK || !strings.Contains(completed.Body.String(), `"outcome":"completed"`) {
		t.Fatalf("package outcome: %d %s", completed.Code, completed.Body.String())
	}
	usedToken, err := app.FindFirstRecordByFilter(schedulingstore.PackageTokensCollectionName, "state = 'used'")
	if err != nil || usedToken.GetString(schedulingstore.LessonField) != completedLessonID {
		t.Fatalf("completed package token: %v", err)
	}
}

func TestFlexibleBookingDoesNotConsumeAnotherAssignmentPackage(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	otherLearner := seedNamedLearner(t, app, "other-learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	otherAssignment := seedLessonAssignment(t, app, teacherID, otherLearner.Id)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	purchase := request(t, server, http.MethodPost, "/api/teachers/assignments/"+otherAssignment.Id+"/packages", `{}`, teacherCookie, true)
	if purchase.Code != http.StatusCreated {
		t.Fatalf("other assignment purchase: %d %s", purchase.Code, purchase.Body.String())
	}
	createAvailabilityException(t, server, teacherCookie, time.Date(2030, 1, 3, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 3, 10, 0, 0, 0, time.UTC), "available")
	booking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-03T09:00:00Z"}`, learnerCookie, true)
	if booking.Code != http.StatusCreated || !strings.Contains(booking.Body.String(), `"plan_type":"ad_hoc"`) {
		t.Fatalf("cross-assignment package changed eligibility: %d %s", booking.Code, booking.Body.String())
	}
	tokens, err := app.FindAllRecords(schedulingstore.PackageTokensCollectionName)
	if err != nil || len(tokens) != 4 {
		t.Fatalf("other assignment tokens: %d %v", len(tokens), err)
	}
	for _, token := range tokens {
		if token.GetString(schedulingstore.TokenStateField) != "available" {
			t.Fatalf("other assignment token was consumed: %s", token.GetString(schedulingstore.TokenStateField))
		}
	}
}

func TestAdHocOutcomeAndSettlementRequireExplicitTeacherWrites(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	createAvailabilityException(t, server, teacherCookie, time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 2, 10, 0, 0, 0, time.UTC), "available")
	booking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-02T09:00:00Z"}`, learnerCookie, true)
	if booking.Code != http.StatusCreated || !strings.Contains(booking.Body.String(), `"plan_type":"ad_hoc"`) {
		t.Fatalf("ad hoc booking: %d %s", booking.Code, booking.Body.String())
	}
	lessonID := responseID(t, booking.Body.Bytes())
	now = time.Date(2030, time.January, 2, 10, 0, 0, 0, time.UTC)
	work := request(t, server, http.MethodGet, "/api/teachers/unresolved-work", "", teacherCookie, false)
	if work.Code != http.StatusOK || !strings.Contains(work.Body.String(), lessonID) {
		t.Fatalf("unresolved work: %d %s", work.Code, work.Body.String())
	}
	outcome := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lessonID+"/outcome", `{"outcome":"completed"}`, teacherCookie, true)
	if outcome.Code != http.StatusOK || !strings.Contains(outcome.Body.String(), `"outcome":"completed"`) {
		t.Fatalf("outcome: %d %s", outcome.Code, outcome.Body.String())
	}
	settlement := request(t, server, http.MethodPost, "/api/teachers/lessons/"+lessonID+"/settlement", `{"settlement":"paid"}`, teacherCookie, true)
	if settlement.Code != http.StatusOK || !strings.Contains(settlement.Body.String(), `"settlement_state":"paid"`) {
		t.Fatalf("settlement: %d %s", settlement.Code, settlement.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(settlement.Body.Bytes(), &payload); err != nil || payload["lesson"] != lessonID {
		t.Fatalf("settlement response: %v %#v", err, payload)
	}
	createAvailabilityException(t, server, teacherCookie, time.Date(2030, 1, 3, 10, 0, 0, 0, time.UTC), time.Date(2030, 1, 3, 11, 0, 0, 0, time.UTC), "available")
	missedBooking := request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-03T10:00:00Z"}`, learnerCookie, true)
	missedLessonID := responseID(t, missedBooking.Body.Bytes())
	now = time.Date(2030, 1, 3, 11, 0, 0, 0, time.UTC)
	missed := request(t, server, http.MethodPost, "/api/teachers/lessons/"+missedLessonID+"/outcome", `{"outcome":"learner_no_show"}`, teacherCookie, true)
	if missed.Code != http.StatusOK || !strings.Contains(missed.Body.String(), `"outcome":"learner_no_show"`) {
		t.Fatalf("ad hoc no-show: %d %s", missed.Code, missed.Body.String())
	}
	missedCharge, err := app.FindFirstRecordByData(schedulingstore.ChargesCollectionName, schedulingstore.SourceIDField, missedLessonID)
	if err != nil || missedCharge.GetString(schedulingstore.SettlementStateField) != "not_applicable" || missedCharge.GetInt(schedulingstore.CurrentAmountMinorField) != 0 {
		t.Fatalf("ad hoc no-show charge: %v", err)
	}
}

func TestPackageClosureRefundCorrectionAndOwnedReadsAreRegistered(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedNamedTeacher(t, app, "second-teacher@example.test")
	seedTestLearner(t, app, true)
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	learnerID := findID(t, app, authconfig.LearnersCollectionName, "learner@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, learnerID)
	teacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	secondTeacherCookie := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "second-teacher@example.test")
	learnerCookie := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	purchase := request(t, server, http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/packages", `{}`, teacherCookie, true)
	packageID := responseID(t, purchase.Body.Bytes())
	if invalid := request(t, server, http.MethodPost, "/api/teachers/packages/"+packageID+"/close", `{"reason":""}`, teacherCookie, true); invalid.Code != http.StatusBadRequest {
		t.Fatalf("reasonless close: %d %s", invalid.Code, invalid.Body.String())
	}
	if forbidden := request(t, server, http.MethodPost, "/api/teachers/packages/"+packageID+"/close", `{"reason":"x"}`, secondTeacherCookie, true); forbidden.Code != http.StatusForbidden {
		t.Fatalf("cross-owner close: %d %s", forbidden.Code, forbidden.Body.String())
	}
	closed := request(t, server, http.MethodPost, "/api/teachers/packages/"+packageID+"/close", `{"reason":"agreement","refund":{"amount_minor":6500,"currency":"PLN","note":"cash"}}`, teacherCookie, true)
	if closed.Code != http.StatusOK || !strings.Contains(closed.Body.String(), `"status":"closed"`) {
		t.Fatalf("close package: %d %s", closed.Code, closed.Body.String())
	}
	events, err := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	var tokenID, invalidationEventID string
	invalidations := 0
	for _, event := range events {
		if event.GetString(schedulingstore.EventTypeField) == "package_token_invalidated" {
			invalidations++
			tokenID, invalidationEventID = event.GetString(schedulingstore.AggregateIDField), event.Id
		}
	}
	if invalidations != 4 {
		t.Fatalf("token invalidation events=%d", invalidations)
	}
	entries, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil || !hasFinancialEntry(entries, "refund", 6500) {
		t.Fatalf("package refund entry missing: entries=%d err=%v", len(entries), err)
	}
	correctionBody := fmt.Sprintf(`{"token_id":%q,"target":"available","corrects_event":%q,"reason":"data correction"}`, tokenID, invalidationEventID)
	corrected := request(t, server, http.MethodPost, "/api/teachers/packages/"+packageID+"/correction", correctionBody, teacherCookie, true)
	if corrected.Code != http.StatusOK || !strings.Contains(corrected.Body.String(), `"state":"available"`) {
		t.Fatalf("package correction: %d %s", corrected.Code, corrected.Body.String())
	}
	learnerRead := request(t, server, http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/packages", "", learnerCookie, false)
	if learnerRead.Code != http.StatusOK || !strings.Contains(learnerRead.Body.String(), packageID) {
		t.Fatalf("learner package read: %d %s", learnerRead.Code, learnerRead.Body.String())
	}
}
