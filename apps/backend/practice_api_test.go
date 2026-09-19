// Verifies that practice plans are atomic and assignment scoped, that sessions follow the teacher day rules, and that summaries start at the previous lesson.
package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
)

type practiceTask struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Lesson *string `json:"lesson"`
}

type practiceSummary struct {
	SinceOn  string `json:"since_on"`
	Days     int    `json:"days"`
	Minutes  int    `json:"minutes"`
	Sessions int    `json:"sessions"`
	Tasks    []struct {
		Title string `json:"title"`
		Count int    `json:"count"`
	} `json:"tasks"`
	Comments []struct {
		Comment string `json:"comment"`
	} `json:"comments"`
	Today      string   `json:"today"`
	RecentDays []string `json:"recent_days"`
}

func TestPracticePlanSessionsAndSummaries(t *testing.T) {
	now := time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC)
	app, server := newTestServerWithClock(t, func() time.Time { return now })
	seedTestTeacher(t, app, true)
	seedTestLearner(t, app, true)
	seedNamedLearner(t, app, "other@example.test")
	teacherID := findID(t, app, authconfig.TeachersCollectionName, "teacher@example.test")
	assignment := seedLessonAssignment(t, app, teacherID, findID(t, app, authconfig.LearnersCollectionName, "learner@example.test"))
	otherAssignment := seedLessonAssignment(t, app, teacherID, findID(t, app, authconfig.LearnersCollectionName, "other@example.test"))
	teacher := loginCookie(t, server, "/api/collections/teachers/auth-with-password", "teacher@example.test")
	learner := loginCookie(t, server, "/api/collections/learners/auth-with-password", "learner@example.test")
	other := loginCookie(t, server, "/api/collections/learners/auth-with-password", "other@example.test")
	createAvailabilityException(t, server, teacher, time.Date(2030, 1, 2, 9, 0, 0, 0, time.UTC), time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC), "available")
	lesson := responseID(t, request(t, server, http.MethodPost, "/api/learners/assignments/"+assignment.Id+"/book", `{"start_at":"2030-01-02T09:00:00Z"}`, learner, true).Body.Bytes())
	status := func(name, method, path, body string, cookie *http.Cookie, want int) []byte {
		t.Helper()
		response := request(t, server, method, path, body, cookie, true)
		if response.Code != want {
			t.Fatalf("%s: %d %s, want %d", name, response.Code, response.Body.String(), want)
		}
		return response.Body.Bytes()
	}
	planPath := "/api/teachers/assignments/" + assignment.Id + "/practice-plan"
	tasksOf := func(body []byte) []practiceTask {
		var payload struct {
			Items []practiceTask `json:"items"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Items
	}

	first := tasksOf(status("first plan", http.MethodPut, planPath, `{"create":[{"title":"Gamy"},{"title":"Refren, tempo 70","suggested_minutes":10}]}`, teacher, http.StatusOK))
	foreign := tasksOf(status("other plan", http.MethodPut, "/api/teachers/assignments/"+otherAssignment.Id+"/practice-plan", `{"create":[{"title":"Obce"}]}`, teacher, http.StatusOK))
	status("foreign task", http.MethodPut, planPath, `{"done":["`+foreign[0].ID+`"]}`, teacher, http.StatusBadRequest)
	status("repeated task", http.MethodPut, planPath, `{"keep":["`+first[0].ID+`"],"done":["`+first[0].ID+`"]}`, teacher, http.StatusBadRequest)
	status("invalid new task", http.MethodPut, planPath, `{"done":["`+first[0].ID+`"],"create":[{"title":" "}]}`, teacher, http.StatusBadRequest)
	status("learner plan", http.MethodPut, planPath, `{"create":[{"title":"x"}]}`, learner, http.StatusForbidden)
	if active := tasksOf(status("list after rejects", http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/practice-tasks", "", learner, http.StatusOK)); len(active) != 2 {
		t.Fatalf("rejected plan changed tasks: %+v", active)
	}

	now = time.Date(2030, time.January, 5, 12, 0, 0, 0, time.UTC)
	second := tasksOf(status("second plan", http.MethodPut, planPath, `{"lesson":"`+lesson+`","done":["`+first[0].ID+`"],"create":[{"title":"Akordy C, G, a, F"}]}`, teacher, http.StatusOK))
	if len(second) != 2 || second[0].ID != first[1].ID || second[1].Lesson == nil || *second[1].Lesson != lesson {
		t.Fatalf("second plan: %+v", second)
	}

	sessions := "/api/learners/assignments/" + assignment.Id + "/practice-sessions"
	session := func(day, extra string) string { return `{"practiced_on":"` + day + `"` + extra + `}` }
	created := responseID(t, status("today", http.MethodPost, sessions, session("2030-01-05", `,"minutes":20,"tasks":["`+second[0].ID+`","`+second[0].ID+`"],"comment":" takt 5 "`), learner, http.StatusCreated))
	status("before the lesson", http.MethodPost, sessions, session("2030-01-01", `,"minutes":30`), learner, http.StatusCreated)
	status("after the lesson", http.MethodPost, sessions, session("2030-01-03", `,"tasks":["`+second[1].ID+`"]`), learner, http.StatusCreated)
	status("future day", http.MethodPost, sessions, session("2030-01-06", ""), learner, http.StatusBadRequest)
	status("too old", http.MethodPost, sessions, session("2029-12-21", ""), learner, http.StatusBadRequest)
	status("foreign task in session", http.MethodPost, sessions, session("2030-01-05", `,"tasks":["`+foreign[0].ID+`"]`), learner, http.StatusBadRequest)
	status("zero minutes", http.MethodPost, sessions, session("2030-01-05", `,"minutes":0`), learner, http.StatusBadRequest)
	status("teacher session", http.MethodPost, "/api/teachers/assignments/"+assignment.Id+"/practice-sessions", session("2030-01-05", ""), teacher, http.StatusNotFound)

	var summary practiceSummary
	body := status("learner summary", http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/practice-summary", "", learner, http.StatusOK)
	if err := json.Unmarshal(body, &summary); err != nil || summary.SinceOn != "2030-01-02" || summary.Days != 2 || summary.Minutes != 20 || summary.Sessions != 2 || len(summary.Tasks) != 2 || summary.Comments[0].Comment != "takt 5" || summary.Today != "2030-01-05" || len(summary.RecentDays) != 3 {
		t.Fatalf("learner summary: %s", body)
	}
	status("other learner summary", http.MethodGet, "/api/learners/assignments/"+assignment.Id+"/practice-summary", "", other, http.StatusForbidden)

	var day struct {
		Items []struct {
			Lesson  string          `json:"lesson"`
			Summary practiceSummary `json:"summary"`
		} `json:"items"`
	}
	body = status("day summaries", http.MethodGet, "/api/teachers/practice-summaries?date=2030-01-02", "", teacher, http.StatusOK)
	if err := json.Unmarshal(body, &day); err != nil || len(day.Items) != 1 || day.Items[0].Lesson != lesson || day.Items[0].Summary.SinceOn != "2029-12-30" || day.Items[0].Summary.Days != 3 {
		t.Fatalf("day summaries: %s", body)
	}

	now = now.Add(8 * 24 * time.Hour)
	status("late delete", http.MethodDelete, "/api/learners/practice-sessions/"+created, "", learner, http.StatusConflict)
	assignment.Set(schedulingstore.ActiveField, false)
	if err := app.Save(assignment); err != nil {
		t.Fatal(err)
	}
	status("inactive assignment", http.MethodPost, sessions, session("2030-01-13", ""), learner, http.StatusConflict)
}
