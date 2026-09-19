// Package materialsapi serves teacher-authored learner materials through persona routes.
// Teachers add, link to a piece, and delete materials of their assignments, learners read their own, and both stream protected files. Persona checks live in personaroute.
package materialsapi

import (
	"errors"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/pocketbase/dbx"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type attachmentDTO struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

type materialDTO struct {
	ID          string          `json:"id"`
	Assignment  string          `json:"assignment"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	Piece       *string         `json:"piece"`
	Attachments []attachmentDTO `json:"attachments"`
	CreatedAt   string          `json:"created_at"`
}

// RegisterRoutes binds the materials routes. Native collection routes stay closed to persona sessions.
func RegisterRoutes(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		r := e.Router
		r.GET("/api/teachers/assignments/{id}/materials", func(event *core.RequestEvent) error { return listMaterials(event, personaroute.Teacher) })
		r.GET("/api/learners/assignments/{id}/materials", func(event *core.RequestEvent) error { return listMaterials(event, personaroute.Learner) })
		r.POST("/api/teachers/assignments/{id}/materials", createMaterial).Bind(apis.BodyLimit(materials.MaxRequestBytes))
		r.PATCH("/api/teachers/materials/{id}", pinMaterial)
		r.DELETE("/api/teachers/materials/{id}", deleteMaterial)
		r.GET("/api/teachers/materials/{id}/files/{name}", func(event *core.RequestEvent) error { return serveFile(event, personaroute.Teacher) })
		r.GET("/api/learners/materials/{id}/files/{name}", func(event *core.RequestEvent) error { return serveFile(event, personaroute.Learner) })
		return e.Next()
	})
}

func listMaterials(e *core.RequestEvent, who personaroute.Role) error {
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return respondError(e, err)
	}
	assignment, err := personaroute.OwnedAssignment(e.App, e.Request.PathValue("id"), who, account.Id)
	if err != nil {
		return respondError(e, err)
	}
	rows, err := e.App.FindRecordsByFilter(materials.CollectionName, "assignment = {:assignment}", "-created", 0, 0, dbx.Params{"assignment": assignment.Id})
	if err != nil {
		return respondError(e, err)
	}
	items := make([]materialDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDTO(row, who))
	}
	return e.JSON(http.StatusOK, map[string]any{"items": items})
}

func createMaterial(e *core.RequestEvent) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	teacher, err := personaroute.Caller(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	assignment, err := personaroute.OwnedAssignment(e.App, e.Request.PathValue("id"), personaroute.Teacher, teacher.Id)
	if err != nil {
		return respondError(e, err)
	}
	record, err := newMaterial(e, assignment.Id)
	if err != nil {
		return respondError(e, err)
	}
	if err := e.App.Save(record); err != nil {
		return respondError(e, saveError(err))
	}
	return e.JSON(http.StatusCreated, toDTO(record, personaroute.Teacher))
}

func newMaterial(e *core.RequestEvent, assignmentID string) (*core.Record, error) {
	files, err := e.FindUploadedFiles(materials.AttachmentsField)
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		return nil, errInvalid
	}
	title := strings.TrimSpace(e.Request.FormValue(materials.TitleField))
	body := materials.SanitizeBody(e.Request.FormValue(materials.BodyField))
	if err := materials.Validate(title, body, len(files)); err != nil {
		return nil, errInvalid
	}
	collection, err := e.App.FindCachedCollectionByNameOrId(materials.CollectionName)
	if err != nil {
		return nil, err
	}
	piece, err := ownedPiece(e.App, assignmentID, e.Request.FormValue(repertoire.MaterialPieceField))
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set(materials.AssignmentField, assignmentID)
	record.Set(repertoire.MaterialPieceField, piece)
	record.Set(materials.TitleField, title)
	record.Set(materials.BodyField, body)
	record.Set(materials.AttachmentsField, files)
	return record, nil
}

// saveError reports field validation failures, such as a wrong file type or size, as an invalid material.
func saveError(err error) error {
	var fieldErrors validation.Errors
	if errors.As(err, &fieldErrors) {
		return errInvalid
	}
	return err
}

func deleteMaterial(e *core.RequestEvent) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	record, err := ownedMaterial(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	if err := e.App.Delete(record); err != nil {
		return respondError(e, err)
	}
	return e.NoContent(http.StatusNoContent)
}

func serveFile(e *core.RequestEvent, who personaroute.Role) error {
	record, err := ownedMaterial(e, who)
	if err != nil {
		return respondError(e, err)
	}
	name := e.Request.PathValue("name")
	if !slices.Contains(record.GetStringSlice(materials.AttachmentsField), name) {
		return respondError(e, errNotFound)
	}
	fsys, err := e.App.NewFilesystem()
	if err != nil {
		return respondError(e, err)
	}
	defer fsys.Close()
	// A cached file would outlive logout or an account switch in the same browser.
	e.Response.Header().Set("Cache-Control", "no-store")
	if err := fsys.Serve(e.Response, e.Request, path.Join(record.BaseFilesPath(), name), name); err != nil {
		return respondError(e, errNotFound)
	}
	return nil
}

func ownedMaterial(e *core.RequestEvent, who personaroute.Role) (*core.Record, error) {
	account, err := personaroute.Caller(e, who)
	if err != nil {
		return nil, err
	}
	record, err := e.App.FindRecordById(materials.CollectionName, e.Request.PathValue("id"))
	if err != nil {
		return nil, personaroute.ErrForbidden
	}
	if _, err := personaroute.OwnedAssignment(e.App, record.GetString(materials.AssignmentField), who, account.Id); err != nil {
		return nil, err
	}
	return record, nil
}

func toDTO(record *core.Record, who personaroute.Role) materialDTO {
	names := record.GetStringSlice(materials.AttachmentsField)
	attachments := make([]attachmentDTO, 0, len(names))
	for _, name := range names {
		attachments = append(attachments, attachmentDTO{
			Name: name,
			Kind: attachmentKind(name),
			URL:  "/api/" + string(who) + "s/materials/" + record.Id + "/files/" + name,
		})
	}
	return materialDTO{
		ID:          record.Id,
		Assignment:  record.GetString(materials.AssignmentField),
		Title:       record.GetString(materials.TitleField),
		Body:        record.GetString(materials.BodyField),
		Piece:       optional(record.GetString(repertoire.MaterialPieceField)),
		Attachments: attachments,
		CreatedAt:   record.GetDateTime("created").Time().UTC().Format(time.RFC3339),
	}
}

func attachmentKind(name string) string {
	if strings.EqualFold(path.Ext(name), ".pdf") {
		return "pdf"
	}
	return "image"
}

var (
	errNotFound = errors.New("material file is unavailable")
	errInvalid  = errors.New("material is invalid")
)

func respondError(e *core.RequestEvent, err error) error {
	if handled, writeErr := personaroute.WriteError(e, err); handled {
		return writeErr
	}
	switch {
	case errors.Is(err, errInvalid):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_material", "message": "The material is invalid."})
	case errors.Is(err, errNotFound), errors.Is(err, filesystem.ErrNotFound):
		return e.JSON(http.StatusNotFound, map[string]string{"code": "not_found", "message": "The file does not exist."})
	default:
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The materials request failed."})
	}
}
