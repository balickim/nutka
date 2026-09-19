// Package practiceapi serves practice tasks, learner practice sessions, and practice summaries of one assignment.
// Persona checks live in personaroute. Practice rules live in practice.
package practiceapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/practice"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Clock supplies the instant for the day rules and the summary window.
type Clock func() time.Time

const maxRequestBytes = 64 << 10

var (
	errInvalidPlan = errors.New("practice plan is invalid")
	errNotFound    = errors.New("practice record is unavailable")
)

// RegisterRoutes binds the practice routes.
func RegisterRoutes(app *pocketbase.PocketBase, clock Clock) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		r := e.Router
		for _, who := range []personaroute.Role{personaroute.Teacher, personaroute.Learner} {
			prefix := "/api/" + string(who) + "s/assignments/{id}/"
			r.GET(prefix+"practice-tasks", func(event *core.RequestEvent) error { return listTasks(event, who) })
			r.GET(prefix+"practice-sessions", func(event *core.RequestEvent) error { return listSessions(event, who, clock()) })
			r.GET(prefix+"practice-summary", func(event *core.RequestEvent) error { return assignmentSummary(event, who, clock()) })
		}
		r.PUT("/api/teachers/assignments/{id}/practice-plan", savePlan)
		r.PATCH("/api/teachers/practice-tasks/{id}", updateTask)
		r.POST("/api/learners/assignments/{id}/practice-sessions", func(event *core.RequestEvent) error { return createSession(event, clock()) })
		r.DELETE("/api/learners/practice-sessions/{id}", func(event *core.RequestEvent) error { return deleteSession(event, clock()) })
		r.GET("/api/teachers/practice-summaries", func(event *core.RequestEvent) error { return daySummaries(event, clock()) })
		return e.Next()
	})
}

// ownedAssignment resolves the caller and the assignment of the route.
func ownedAssignment(e *core.RequestEvent, who personaroute.Role) (*core.Record, error) {
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return nil, err
	}
	return personaroute.OwnedAssignment(e.App, e.Request.PathValue("id"), who, account.Id)
}

// ownedRecord loads a record by the route identifier and hides records of other accounts.
func ownedRecord(e *core.RequestEvent, collection string, who personaroute.Role) (*core.Record, *core.Record, error) {
	if err := personaroute.RequireIntent(e); err != nil {
		return nil, nil, err
	}
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return nil, nil, err
	}
	record, err := e.App.FindRecordById(collection, e.Request.PathValue("id"))
	if err != nil {
		return nil, nil, personaroute.ErrForbidden
	}
	assignment, err := personaroute.OwnedAssignment(e.App, record.GetString(practice.AssignmentField), who, account.Id)
	if err != nil {
		return nil, nil, err
	}
	return record, assignment, nil
}

// teacherLocation reads the teacher timezone that defines the practice day.
func teacherLocation(app core.App, teacherID string) *time.Location {
	teacher, err := app.FindRecordById("teachers", teacherID)
	if err == nil {
		if location, locationErr := time.LoadLocation(teacher.GetString(schedulingstore.TeacherTimezoneField)); locationErr == nil {
			return location
		}
	}
	return time.UTC
}

func decode(e *core.RequestEvent, target any, invalid error) error {
	if err := json.NewDecoder(http.MaxBytesReader(e.Response, e.Request.Body, maxRequestBytes)).Decode(target); err != nil {
		return invalid
	}
	return nil
}

func respondError(e *core.RequestEvent, err error) error {
	if handled, writeErr := personaroute.WriteError(e, err); handled {
		return writeErr
	}
	switch {
	case errors.Is(err, practice.ErrInvalidTask):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_practice_task", "message": "The practice task is invalid."})
	case errors.Is(err, errInvalidPlan):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_practice_plan", "message": "The practice plan is invalid."})
	case errors.Is(err, practice.ErrInvalidSession):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_practice_session", "message": "The practice session is invalid."})
	case errors.Is(err, practice.ErrLocked):
		return e.JSON(http.StatusConflict, map[string]string{"code": "practice_session_locked", "message": "The practice session can no longer be deleted."})
	case errors.Is(err, errNotFound):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_request", "message": "The request is invalid."})
	default:
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The practice request failed."})
	}
}
