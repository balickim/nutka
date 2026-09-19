// Verifies that lesson notes follow the lesson start and cancellation rules, stay one per lesson, and stay inside one assignment.
package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
)

func TestLessonNotesFollowLessonRulesAndAssignmentScope(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedNamedTeacher(t, app, "second-teacher@example.test")
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, findID(t, app, authconfig.LearnersCollectionName, "learner@example.test"))
	otherAssignment := seedLessonAssignment(t, app, teacherID, findID(t, app, authconfig.LearnersCollectionName, "other@example.test"))
	teacher := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	stranger := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "second-teacher@example.test")
	learner := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	other := loginCookie(t, server, "/api/collections/learners/auth-with-password", "other@example.test")
	createAvailabilityException(t, server, teacher, time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 3, 12, 0, 0, 0, time.UTC), "available")
	book := func(start string) string {
		return responseID(t, request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"`+start+`"}`, learner, true).Body.Bytes())
	}
	started, future, cancelled := book("2030-01-02T09:00:00Z"), book("2030-01-03T11:00:00Z"), book("2030-01-02T11:00:00Z")
	if response := request(t, server, http.MethodPost, "/api/learners/lessons/"+cancelled+"/cancel", `{}`, learner, true); response.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", response.Code, response.Body.String())
	}
	foreignMaterial := material(t, server, otherAssignment.Id, teacher)
	ownMaterial := material(t, server, assignment.Id, teacher)
	now = time.Date(2030, time.January, 2, 12, 0, 0, 0, time.UTC)
	note := func(lesson, body string, cookie *http.Cookie) *http.Response {
		return request(t, server, http.MethodPut, "/api/teachers/lessons/"+lesson+"/note", body, cookie, true).Result()
	}

	expectStatus(t, "future lesson", note(future, `{"body":"<p>x</p>"}`, teacher), http.StatusConflict)
	expectStatus(t, "cancelled lesson", note(cancelled, `{"body":"<p>x</p>"}`, teacher), http.StatusConflict)
	expectStatus(t, "foreign material", note(started, `{"body":"<p>x</p>","materials":["`+foreignMaterial+`"]}`, teacher), http.StatusBadRequest)
	expectStatus(t, "empty body", note(started, `{"body":"<p> </p>"}`, teacher), http.StatusBadRequest)
	expectStatus(t, "learner write", note(started, `{"body":"<p>x</p>"}`, learner), http.StatusForbidden)
	expectStatus(t, "foreign teacher", note(started, `{"body":"<p>x</p>"}`, stranger), http.StatusForbidden)
	expectStatus(t, "first save", note(started, `{"body":"<p>Pierwsza</p>"}`, teacher), http.StatusOK)
	expectStatus(t, "replace", note(started, `{"body":"<p>Refren <strong>wolniej</strong></p><script>x</script>","materials":["`+ownMaterial+`"]}`, teacher), http.StatusOK)

	listPath := "/api/learners/assignments/" + assignment.Id + "/lesson-notes"
	list := request(t, server, http.MethodGet, listPath, "", learner, false)
	var body struct {
		Items []struct {
			Body      string `json:"body"`
			Materials []struct {
				Title string `json:"title"`
			} `json:"materials"`
		} `json:"items"`
	}
	if list.Code != http.StatusOK || json.Unmarshal(list.Body.Bytes(), &body) != nil || len(body.Items) != 1 || strings.Contains(body.Items[0].Body, "script") || !strings.Contains(body.Items[0].Body, "wolniej") || len(body.Items[0].Materials) != 1 {
		t.Fatalf("learner list: %d %s", list.Code, list.Body.String())
	}
	if response := request(t, server, http.MethodGet, listPath, "", other, false); response.Code != http.StatusForbidden {
		t.Fatalf("other learner read notes: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodDelete, "/api/teachers/lessons/"+started+"/note", "", teacher, true); response.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, listPath, "", learner, false); !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatalf("deleted note still listed: %s", response.Body.String())
	}
}

func material(t *testing.T, server http.Handler, assignmentID string, cookie *http.Cookie) string {
	t.Helper()
	response := materialRequest(t, server, "/api/teachers/assignments/"+assignmentID+"/materials", map[string]string{"title": "Gamy", "body": "<p>C-dur</p>"}, nil, cookie)
	if response.Code != http.StatusCreated {
		t.Fatalf("material: %d %s", response.Code, response.Body.String())
	}
	return responseID(t, response.Body.Bytes())
}

func expectStatus(t *testing.T, name string, response *http.Response, want int) {
	t.Helper()
	if response.StatusCode != want {
		t.Fatalf("%s: status %d, want %d", name, response.StatusCode, want)
	}
}
