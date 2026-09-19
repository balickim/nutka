// This file links a material to at most one piece of the same assignment.
package materialsapi

import (
	"encoding/json"
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/pocketbase/pocketbase/core"
)

type pinInput struct {
	Piece *string `json:"piece"`
}

// pinMaterial changes only the piece link. A null or empty piece removes the link.
func pinMaterial(e *core.RequestEvent) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	record, err := ownedMaterial(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	var input pinInput
	if err := json.NewDecoder(http.MaxBytesReader(e.Response, e.Request.Body, 4<<10)).Decode(&input); err != nil {
		return respondError(e, errInvalid)
	}
	piece, err := ownedPiece(e.App, record.GetString(materials.AssignmentField), value(input.Piece))
	if err != nil {
		return respondError(e, err)
	}
	record.Set(repertoire.MaterialPieceField, piece)
	if err := e.App.Save(record); err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, toDTO(record, personaroute.Teacher))
}

// ownedPiece accepts an empty piece and rejects a piece of another assignment.
func ownedPiece(app core.App, assignmentID, pieceID string) (string, error) {
	if pieceID == "" {
		return "", nil
	}
	piece, err := app.FindRecordById(repertoire.CollectionName, pieceID)
	if err != nil || piece.GetString(repertoire.AssignmentField) != assignmentID {
		return "", errInvalid
	}
	return piece.Id, nil
}

func value(field *string) string {
	if field == nil {
		return ""
	}
	return *field
}

func optional(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
