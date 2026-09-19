// Verifies teacher transfer details and the learner payment-due read across realms, assignments, and settlement states.
package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
)

type paymentDueBody struct {
	TotalMinor int64 `json:"total_minor"`
	Items      []struct {
		Charge string `json:"charge"`
		Kind   string `json:"kind"`
	} `json:"items"`
	RecentPayments []struct {
		Kind string `json:"kind"`
	} `json:"recent_payments"`
	Instructions *struct {
		IBAN string `json:"iban"`
	} `json:"instructions"`
}

func readPaymentDue(t *testing.T, server http.Handler, path string, cookie *http.Cookie) paymentDueBody {
	t.Helper()
	response := request(t, server, http.MethodGet, path, "", cookie, false)
	var body paymentDueBody
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil {
		t.Fatalf("payment due: %d %s", response.Code, response.Body.String())
	}
	return body
}

func TestTeacherPaymentDetailsStayPrivateAndValidated(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	teacher := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learner := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	const path = "/api/teachers/payment-details"
	if response := request(t, server, http.MethodGet, path, "", learner, false); response.Code != http.StatusForbidden {
		t.Fatalf("learner read teacher details: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPut, path, `{"account_holder":"Dominika","iban":"PL61109010140000071219812875"}`, teacher, true); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_payment_details") {
		t.Fatalf("wrong checksum: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodPut, path, `{"account_holder":"Dominika","iban":"61 1090 1014 0000 0712 1981 2874"}`, teacher, false); response.Code != http.StatusForbidden {
		t.Fatalf("missing intent: %d %s", response.Code, response.Body.String())
	}
	saved := request(t, server, http.MethodPut, path, `{"account_holder":"Dominika","iban":"61 1090 1014 0000 0712 1981 2874"}`, teacher, true)
	if saved.Code != http.StatusOK || !strings.Contains(saved.Body.String(), `"iban":"PL61109010140000071219812874"`) {
		t.Fatalf("save: %d %s", saved.Code, saved.Body.String())
	}
	if me := request(t, server, http.MethodGet, "/api/teachers/auth/me", "", teacher, false); me.Code != http.StatusOK || strings.Contains(me.Body.String(), "PL6110") || strings.Contains(me.Body.String(), "payment_") {
		t.Fatalf("auth response exposed payment details: %d %s", me.Code, me.Body.String())
	}
}

func TestLearnerPaymentDueListsOpenAmountsOfOwnAssignment(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, findID(t, app, authconfig.LearnersCollectionName, "learner@example.test"))
	teacher := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learner := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	other := loginCookie(t, server, "/api/collections/learners/auth-with-password", "other@example.test")
	path := "/api/learners/assignments/" + assignment.Id + "/payment-due"
	if response := request(t, server, http.MethodGet, path, "", other, false); response.Code != http.StatusForbidden {
		t.Fatalf("other learner read payment due: %d %s", response.Code, response.Body.String())
	}
	createAvailabilityException(t, server, teacher, time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 3, 12, 0, 0, 0, time.UTC), "available")
	ended := responseID(t, request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-02T09:00:00Z"}`, learner, true).Body.Bytes())
	request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-03T10:00:00Z"}`, learner, true)
	now = time.Date(2030, time.January, 2, 10, 0, 0, 0, time.UTC)

	due := readPaymentDue(t, server, path, learner)
	if due.TotalMinor != 8000 || len(due.Items) != 1 || due.Items[0].Kind != "lesson" || due.Instructions != nil {
		t.Fatalf("ended ad hoc lesson only: %+v", due)
	}
	request(t, server, http.MethodPost, "/api/teachers/lessons/"+ended+"/outcome", `{"outcome":"completed"}`, teacher, true)
	if response := request(t, server, http.MethodPost, "/api/teachers/lessons/"+ended+"/settlement", `{"settlement":"paid"}`, teacher, true); response.Code != http.StatusOK {
		t.Fatalf("settlement: %d %s", response.Code, response.Body.String())
	}
	request(t, server, http.MethodPut, "/api/teachers/payment-details", `{"account_holder":"Dominika","iban":"PL61109010140000071219812874"}`, teacher, true)
	due = readPaymentDue(t, server, path, learner)
	if due.TotalMinor != 0 || len(due.Items) != 0 || len(due.RecentPayments) != 1 || due.Instructions == nil || due.Instructions.IBAN != "PL61109010140000071219812874" {
		t.Fatalf("paid lesson and instructions: %+v", due)
	}
}
