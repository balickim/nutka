// Package lessonnotesapi serves teacher notes on lessons. The assigned teacher writes one note per lesson and both personas read the assignment list.
// Persona checks live in personaroute. Note rules live in lessonnotes.
package lessonnotesapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/lessonnotes"
	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Clock supplies the instant for the lesson start rule.
type Clock func() time.Time

type noteInput struct {
	Body      string   `json:"body"`
	Materials []string `json:"materials"`
}

// RegisterRoutes binds the lesson note routes.
func RegisterRoutes(app *pocketbase.PocketBase, clock Clock) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		r := e.Router
		r.PUT("/api/teachers/lessons/{id}/note", func(event *core.RequestEvent) error { return saveNote(event, clock()) })
		r.DELETE("/api/teachers/lessons/{id}/note", deleteNote)
		r.GET("/api/teachers/assignments/{id}/lesson-notes", func(event *core.RequestEvent) error { return listNotes(event, personaroute.Teacher) })
		r.GET("/api/learners/assignments/{id}/lesson-notes", func(event *core.RequestEvent) error { return listNotes(event, personaroute.Learner) })
		return e.Next()
	})
}

func saveNote(e *core.RequestEvent, now time.Time) error {
	lesson, err := ownedLesson(e)
	if err != nil {
		return respondError(e, err)
	}
	if err := lessonnotes.CheckLesson(lesson.GetDateTime(schedulingstore.StartAtField).Time(), lesson.GetString(schedulingstore.ScheduleStateField), now); err != nil {
		return respondError(e, err)
	}
	var input noteInput
	if err := json.NewDecoder(http.MaxBytesReader(e.Response, e.Request.Body, 2*lessonnotes.BodyMaxBytes)).Decode(&input); err != nil {
		return respondError(e, lessonnotes.ErrInvalid)
	}
	body := materials.SanitizeBody(input.Body)
	materialIDs, err := ownedMaterials(e.App, lesson.GetString(schedulingstore.AssignmentField), input.Materials)
	if err != nil {
		return respondError(e, err)
	}
	if err := lessonnotes.Validate(body, len(materialIDs)); err != nil {
		return respondError(e, err)
	}
	record, err := noteRecord(e.App, lesson)
	if err != nil {
		return respondError(e, err)
	}
	record.Set(lessonnotes.BodyField, body)
	record.Set(lessonnotes.MaterialsField, materialIDs)
	if err := e.App.Save(record); err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, toDTO(e.App, record, lesson))
}

func deleteNote(e *core.RequestEvent) error {
	lesson, err := ownedLesson(e)
	if err != nil {
		return respondError(e, err)
	}
	if note, err := e.App.FindFirstRecordByData(lessonnotes.CollectionName, lessonnotes.LessonField, lesson.Id); err == nil {
		if err := e.App.Delete(note); err != nil {
			return respondError(e, err)
		}
	}
	return e.NoContent(http.StatusNoContent)
}

func listNotes(e *core.RequestEvent, who personaroute.Role) error {
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return respondError(e, err)
	}
	assignment, err := personaroute.OwnedAssignment(e.App, e.Request.PathValue("id"), who, account.Id)
	if err != nil {
		return respondError(e, err)
	}
	rows, err := e.App.FindAllRecords(lessonnotes.CollectionName, dbx.HashExp{lessonnotes.AssignmentField: assignment.Id})
	if err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, map[string]any{"items": sortedDTOs(e.App, rows)})
}

// ownedLesson resolves a write request to a lesson of the calling teacher and hides foreign lessons.
func ownedLesson(e *core.RequestEvent) (*core.Record, error) {
	if err := personaroute.RequireIntent(e); err != nil {
		return nil, err
	}
	teacher, err := personaroute.Caller(e, personaroute.Teacher)
	if err != nil {
		return nil, err
	}
	lesson, err := e.App.FindRecordById(schedulingstore.LessonsCollectionName, e.Request.PathValue("id"))
	if err != nil {
		return nil, personaroute.ErrForbidden
	}
	if _, err := personaroute.OwnedAssignment(e.App, lesson.GetString(schedulingstore.AssignmentField), personaroute.Teacher, teacher.Id); err != nil {
		return nil, err
	}
	return lesson, nil
}

func noteRecord(app core.App, lesson *core.Record) (*core.Record, error) {
	if existing, err := app.FindFirstRecordByData(lessonnotes.CollectionName, lessonnotes.LessonField, lesson.Id); err == nil {
		return existing, nil
	}
	collection, err := app.FindCachedCollectionByNameOrId(lessonnotes.CollectionName)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set(lessonnotes.LessonField, lesson.Id)
	record.Set(lessonnotes.AssignmentField, lesson.GetString(schedulingstore.AssignmentField))
	return record, nil
}

func respondError(e *core.RequestEvent, err error) error {
	if handled, writeErr := personaroute.WriteError(e, err); handled {
		return writeErr
	}
	switch {
	case errors.Is(err, lessonnotes.ErrNotStarted):
		return e.JSON(http.StatusConflict, map[string]string{"code": "lesson_not_started", "message": "The lesson has not started."})
	case errors.Is(err, lessonnotes.ErrCancelled):
		return e.JSON(http.StatusConflict, map[string]string{"code": "lesson_cancelled", "message": "The lesson is cancelled."})
	case errors.Is(err, lessonnotes.ErrInvalid):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_lesson_note", "message": "The lesson note is invalid."})
	default:
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The lesson note request failed."})
	}
}
