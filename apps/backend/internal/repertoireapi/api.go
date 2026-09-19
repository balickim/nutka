// Package repertoireapi serves the pieces of one assignment. The assigned teacher manages pieces, and the assigned learner adds and removes own wishes.
// Persona checks live in personaroute. Piece rules live in repertoire.
package repertoireapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Clock supplies the instant of a status change.
type Clock func() time.Time

const maxRequestBytes = 8 << 10

// pieceInput uses pointers so a teacher update changes only the sent fields.
type pieceInput struct {
	Title  *string `json:"title"`
	Artist *string `json:"artist"`
	Status *string `json:"status"`
}

// RegisterRoutes binds the piece routes.
func RegisterRoutes(app *pocketbase.PocketBase, clock Clock) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		r := e.Router
		r.GET("/api/teachers/assignments/{id}/pieces", func(event *core.RequestEvent) error { return listPieces(event, personaroute.Teacher) })
		r.GET("/api/learners/assignments/{id}/pieces", func(event *core.RequestEvent) error { return listPieces(event, personaroute.Learner) })
		r.POST("/api/teachers/assignments/{id}/pieces", func(event *core.RequestEvent) error { return createPiece(event, personaroute.Teacher, clock()) })
		r.POST("/api/learners/assignments/{id}/pieces", func(event *core.RequestEvent) error { return createPiece(event, personaroute.Learner, clock()) })
		r.PATCH("/api/teachers/pieces/{id}", func(event *core.RequestEvent) error { return updatePiece(event, clock()) })
		r.DELETE("/api/teachers/pieces/{id}", func(event *core.RequestEvent) error { return deletePiece(event, personaroute.Teacher) })
		r.DELETE("/api/learners/pieces/{id}", func(event *core.RequestEvent) error { return deletePiece(event, personaroute.Learner) })
		return e.Next()
	})
}

func listPieces(e *core.RequestEvent, who personaroute.Role) error {
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return respondError(e, err)
	}
	assignment, err := personaroute.OwnedAssignment(e.App, e.Request.PathValue("id"), who, account.Id)
	if err != nil {
		return respondError(e, err)
	}
	rows, err := e.App.FindRecordsByFilter(repertoire.CollectionName, "assignment = {:assignment}", "-status_changed_at,-created", 0, 0, dbx.Params{"assignment": assignment.Id})
	if err != nil {
		return respondError(e, err)
	}
	items, err := listDTOs(e.App, assignment.Id, rows)
	if err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, map[string]any{"items": items})
}

// createPiece stores a learner piece as a wish. The author comes from the route, never from the request.
func createPiece(e *core.RequestEvent, who personaroute.Role, now time.Time) error {
	assignment, err := writableAssignment(e, who)
	if err != nil {
		return respondError(e, err)
	}
	input, err := decode(e)
	if err != nil {
		return respondError(e, err)
	}
	fields := repertoire.Input{Title: value(input.Title), Artist: value(input.Artist), Status: repertoire.StatusLearning}
	if input.Status != nil {
		fields.Status = *input.Status
	}
	if who == personaroute.Learner {
		fields.Status = repertoire.StatusWish
	}
	collection, err := e.App.FindCachedCollectionByNameOrId(repertoire.CollectionName)
	if err != nil {
		return respondError(e, err)
	}
	record := core.NewRecord(collection)
	record.Set(repertoire.AssignmentField, assignment.Id)
	record.Set(repertoire.ProposedByField, string(who))
	if err := apply(record, fields, now); err != nil {
		return respondError(e, err)
	}
	if err := e.App.Save(record); err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusCreated, toDTO(record, counters{}))
}

func updatePiece(e *core.RequestEvent, now time.Time) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	record, err := ownedPiece(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	input, err := decode(e)
	if err != nil {
		return respondError(e, err)
	}
	fields := repertoire.Input{Title: or(input.Title, record.GetString(repertoire.TitleField)), Artist: or(input.Artist, record.GetString(repertoire.ArtistField)), Status: or(input.Status, record.GetString(repertoire.StatusField))}
	if err := apply(record, fields, now); err != nil {
		return respondError(e, err)
	}
	if err := e.App.Save(record); err != nil {
		return respondError(e, err)
	}
	stats, err := materialCounters(e.App, record.GetString(repertoire.AssignmentField))
	if err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, toDTO(record, stats[record.Id]))
}

func deletePiece(e *core.RequestEvent, who personaroute.Role) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	record, err := ownedPiece(e, who)
	if err != nil {
		return respondError(e, err)
	}
	if who == personaroute.Learner {
		if err := learnerDeleteAllowed(e.App, record); err != nil {
			return respondError(e, err)
		}
	}
	if err := e.App.Delete(record); err != nil {
		return respondError(e, err)
	}
	return e.NoContent(http.StatusNoContent)
}

func learnerDeleteAllowed(app core.App, record *core.Record) error {
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, record.GetString(repertoire.AssignmentField))
	if err != nil {
		return err
	}
	if err := personaroute.RequireActive(assignment); err != nil {
		return err
	}
	return repertoire.LearnerMayDelete(record.GetString(repertoire.ProposedByField), record.GetString(repertoire.StatusField))
}

// writableAssignment resolves a create request. A learner writes only to an active assignment.
func writableAssignment(e *core.RequestEvent, who personaroute.Role) (*core.Record, error) {
	if err := personaroute.RequireIntent(e); err != nil {
		return nil, err
	}
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return nil, err
	}
	assignment, err := personaroute.OwnedAssignment(e.App, e.Request.PathValue("id"), who, account.Id)
	if err != nil {
		return nil, err
	}
	if who == personaroute.Learner {
		return assignment, personaroute.RequireActive(assignment)
	}
	return assignment, nil
}

// ownedPiece hides the pieces of other accounts.
func ownedPiece(e *core.RequestEvent, who personaroute.Role) (*core.Record, error) {
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return nil, err
	}
	record, err := e.App.FindRecordById(repertoire.CollectionName, e.Request.PathValue("id"))
	if err != nil {
		return nil, personaroute.ErrForbidden
	}
	if _, err := personaroute.OwnedAssignment(e.App, record.GetString(repertoire.AssignmentField), who, account.Id); err != nil {
		return nil, err
	}
	return record, nil
}

func decode(e *core.RequestEvent) (pieceInput, error) {
	var input pieceInput
	if err := json.NewDecoder(http.MaxBytesReader(e.Response, e.Request.Body, maxRequestBytes)).Decode(&input); err != nil {
		return input, repertoire.ErrInvalid
	}
	return input, nil
}

// apply validates the fields and sets status_changed_at only when the status changes.
func apply(record *core.Record, input repertoire.Input, now time.Time) error {
	fields := repertoire.Normalize(input)
	if err := repertoire.Validate(fields); err != nil {
		return err
	}
	if record.GetString(repertoire.StatusField) != fields.Status {
		record.Set(repertoire.StatusChangedAtField, now.UTC())
	}
	record.Set(repertoire.TitleField, fields.Title)
	record.Set(repertoire.ArtistField, fields.Artist)
	record.Set(repertoire.StatusField, fields.Status)
	return nil
}

func value(field *string) string {
	return or(field, "")
}

func or(field *string, fallback string) string {
	if field == nil {
		return fallback
	}
	return *field
}

func respondError(e *core.RequestEvent, err error) error {
	if handled, writeErr := personaroute.WriteError(e, err); handled {
		return writeErr
	}
	switch {
	case errors.Is(err, repertoire.ErrInvalid):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_piece", "message": "The piece is invalid."})
	case errors.Is(err, repertoire.ErrLocked):
		return e.JSON(http.StatusConflict, map[string]string{"code": "piece_locked", "message": "The learner can delete only an own wish."})
	default:
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The pieces request failed."})
	}
}
