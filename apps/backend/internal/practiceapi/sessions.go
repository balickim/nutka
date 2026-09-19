// This file records, deletes, and lists learner practice sessions.
package practiceapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/practice"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const sessionsPerPage = 20

type sessionInput struct {
	PracticedOn string   `json:"practiced_on"`
	Minutes     *int     `json:"minutes"`
	Tasks       []string `json:"tasks"`
	Comment     string   `json:"comment"`
}

func createSession(e *core.RequestEvent, now time.Time) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	assignment, err := ownedAssignment(e, personaroute.Learner)
	if err != nil {
		return respondError(e, err)
	}
	if err := personaroute.RequireActive(assignment); err != nil {
		return respondError(e, err)
	}
	var input sessionInput
	if err := decode(e, &input, practice.ErrInvalidSession); err != nil {
		return respondError(e, err)
	}
	session, err := sessionOf(e.App, assignment, input, now)
	if err != nil {
		return respondError(e, err)
	}
	collection, err := e.App.FindCachedCollectionByNameOrId(practice.SessionsCollectionName)
	if err != nil {
		return respondError(e, err)
	}
	record := core.NewRecord(collection)
	record.Load(map[string]any{practice.AssignmentField: assignment.Id, practice.PracticedOnField: session.PracticedOn, practice.MinutesField: session.Minutes, practice.TasksField: session.Tasks, practice.CommentField: session.Comment})
	if err := e.App.Save(record); err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusCreated, sessionDTOs(e.App, []*core.Record{record}, personaroute.Learner, now)[0])
}

// sessionOf validates the input in the teacher day and keeps unique tasks of the same assignment.
func sessionOf(app core.App, assignment *core.Record, input sessionInput, now time.Time) (practice.Session, error) {
	session := practice.Session{PracticedOn: input.PracticedOn, Comment: strings.TrimSpace(input.Comment), Tasks: []string{}}
	if input.Minutes != nil {
		session.Minutes = *input.Minutes
		if session.Minutes == 0 {
			return session, practice.ErrInvalidSession
		}
	}
	seen := map[string]bool{}
	for _, id := range input.Tasks {
		if !belongs(app, practice.TasksCollectionName, practice.AssignmentField, id, assignment.Id) {
			return session, practice.ErrInvalidSession
		}
		if !seen[id] {
			seen[id] = true
			session.Tasks = append(session.Tasks, id)
		}
	}
	location := teacherLocation(app, assignment.GetString("teacher"))
	return session, practice.ValidateSession(session, now, location)
}

func deleteSession(e *core.RequestEvent, now time.Time) error {
	record, assignment, err := ownedRecord(e, practice.SessionsCollectionName, personaroute.Learner)
	if err != nil {
		return respondError(e, err)
	}
	if err := personaroute.RequireActive(assignment); err != nil {
		return respondError(e, err)
	}
	if err := practice.LearnerMayDelete(record.GetDateTime("created").Time(), now); err != nil {
		return respondError(e, err)
	}
	if err := e.App.Delete(record); err != nil {
		return respondError(e, err)
	}
	return e.NoContent(http.StatusNoContent)
}

func listSessions(e *core.RequestEvent, who personaroute.Role, now time.Time) error {
	assignment, err := ownedAssignment(e, who)
	if err != nil {
		return respondError(e, err)
	}
	page, err := strconv.Atoi(e.Request.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	params := dbx.Params{"assignment": assignment.Id}
	rows, err := e.App.FindRecordsByFilter(practice.SessionsCollectionName, "assignment = {:assignment}", "-practiced_on,-created", sessionsPerPage, (page-1)*sessionsPerPage, params)
	if err != nil {
		return respondError(e, err)
	}
	total, err := e.App.CountRecords(practice.SessionsCollectionName, dbx.HashExp{practice.AssignmentField: assignment.Id})
	if err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, map[string]any{"items": sessionDTOs(e.App, rows, who, now), "page": page, "per_page": sessionsPerPage, "total": total})
}
