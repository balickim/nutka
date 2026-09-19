package personaroute

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

type fixture struct {
	app      core.App
	teacher  *core.Record
	learner  *core.Record
	pending  *core.Record
	assigned string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir(), DefaultDev: false})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	teachers := save(t, app, core.NewAuthCollection(authconfig.TeachersCollectionName))
	learners := save(t, app, core.NewAuthCollection(authconfig.LearnersCollectionName))
	assignments := core.NewBaseCollection(schedulingstore.TeacherLearnersCollectionName)
	assignments.Fields.Add(&core.TextField{Name: string(Teacher)}, &core.TextField{Name: string(Learner)}, &core.BoolField{Name: schedulingstore.ActiveField})
	save(t, app, assignments)
	f := fixture{app: app, teacher: account(t, app, teachers, "teacher@example.test", true), learner: account(t, app, learners, "learner@example.test", true), pending: account(t, app, learners, "pending@example.test", false)}
	row := core.NewRecord(assignments)
	row.Set(string(Teacher), f.teacher.Id)
	row.Set(string(Learner), f.learner.Id)
	if err := app.Save(row); err != nil {
		t.Fatal(err)
	}
	f.assigned = row.Id
	return f
}

func save(t *testing.T, app core.App, collection *core.Collection) *core.Collection {
	t.Helper()
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	return collection
}

func account(t *testing.T, app core.App, collection *core.Collection, email string, verified bool) *core.Record {
	t.Helper()
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.SetPassword("local-password")
	record.SetVerified(verified)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func event(t *testing.T, app core.App, auth *core.Record, cookieName string) *core.RequestEvent {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if auth != nil {
		token, err := auth.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		request.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	}
	return &core.RequestEvent{App: app, Auth: auth, Event: router.Event{Request: request, Response: httptest.NewRecorder()}}
}

func TestCallerRejectsCrossRealmAndUnverifiedSessions(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		name   string
		auth   *core.Record
		cookie string
		who    Role
		want   error
	}{
		{"guest", nil, "", Learner, ErrUnauthenticated},
		{"learner on teacher realm", f.learner, sessioncookie.LearnerName, Teacher, ErrForbidden},
		{"learner token in teacher cookie", f.learner, sessioncookie.TeacherName, Teacher, ErrForbidden},
		{"unverified learner", f.pending, sessioncookie.LearnerName, Learner, ErrForbidden},
		{"verified learner", f.learner, sessioncookie.LearnerName, Learner, nil},
	}
	for _, tc := range cases {
		if _, err := Caller(event(t, f.app, tc.auth, tc.cookie), tc.who); !errors.Is(err, tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestOwnedAssignmentHidesForeignAssignments(t *testing.T) {
	f := newFixture(t)
	if _, err := OwnedAssignment(f.app, f.assigned, Learner, f.learner.Id); err != nil {
		t.Fatalf("owner was rejected: %v", err)
	}
	if _, err := OwnedAssignment(f.app, f.assigned, Learner, f.teacher.Id); !errors.Is(err, ErrForbidden) {
		t.Fatalf("account in the wrong role was accepted: %v", err)
	}
	if _, err := OwnedAssignment(f.app, f.assigned, Learner, f.pending.Id); !errors.Is(err, ErrForbidden) {
		t.Fatalf("foreign learner was accepted: %v", err)
	}
	if _, err := OwnedAssignment(f.app, "missing", Learner, f.learner.Id); !errors.Is(err, ErrForbidden) {
		t.Fatalf("missing assignment was not hidden: %v", err)
	}
}

func TestRequireIntent(t *testing.T) {
	f := newFixture(t)
	e := event(t, f.app, nil, "")
	if err := RequireIntent(e); !errors.Is(err, ErrIntent) {
		t.Fatalf("missing intent was accepted: %v", err)
	}
	e.Request.Header.Set(authconfig.AuthIntentHeader, authconfig.AuthIntentValue)
	if err := RequireIntent(e); err != nil {
		t.Fatalf("intent was rejected: %v", err)
	}
}

func TestRequireActive(t *testing.T) {
	f := newFixture(t)
	row, err := f.app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, f.assigned)
	if err != nil {
		t.Fatal(err)
	}
	if err := RequireActive(row); !errors.Is(err, ErrInactive) {
		t.Fatalf("inactive assignment was accepted: %v", err)
	}
	row.Set(schedulingstore.ActiveField, true)
	if err := RequireActive(row); err != nil {
		t.Fatalf("active assignment was rejected: %v", err)
	}
}
