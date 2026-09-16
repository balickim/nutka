package historyapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestHistoryHandlerRejectsLearnerSessionOnTeacherRealm(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir(), DefaultDev: false})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	teachers := core.NewAuthCollection("teachers")
	learners := core.NewAuthCollection("learners")
	if err := app.Save(teachers); err != nil {
		t.Fatal(err)
	}
	if err := app.Save(learners); err != nil {
		t.Fatal(err)
	}
	learner := core.NewRecord(learners)
	learner.SetEmail("learner@example.test")
	learner.SetPassword("local-password")
	learner.SetVerified(true)
	if err := app.Save(learner); err != nil {
		t.Fatal(err)
	}
	token, err := learner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/teachers/assignments/assignment/history", nil)
	request.AddCookie(&http.Cookie{Name: sessioncookie.LearnerName, Value: token})
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Auth: learner, Event: router.Event{Request: request, Response: response}}
	if err := serveHistory(event, history.TeacherActor); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusForbidden || response.Body.String() == "" {
		t.Fatalf("learner session crossed teacher history realm: %d %s", response.Code, response.Body.String())
	}
	_ = app.ResetBootstrapState()
}
