// Package migrations creates the closed pieces collection and adds the optional piece link to learner materials.
// Only superusers can use the native record routes of pieces.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(createPieces, func(core.App) error { return nil })
}

func createPieces(app core.App) error {
	assignments, err := findCollection(app, teacherLearnersCollection)
	if err != nil {
		return fmt.Errorf("find %s collection: %w", teacherLearnersCollection, err)
	}
	pieces, err := ensureCollection(app, repertoire.CollectionName, []core.Field{
		&core.RelationField{Name: repertoire.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true, MaxSelect: 1},
		&core.TextField{Name: repertoire.TitleField, Required: true, Max: repertoire.TextMaxLength},
		&core.TextField{Name: repertoire.ArtistField, Max: repertoire.TextMaxLength},
		&core.SelectField{Name: repertoire.StatusField, Required: true, MaxSelect: 1, Values: repertoire.Statuses},
		&core.DateField{Name: repertoire.StatusChangedAtField, Required: true},
		&core.SelectField{Name: repertoire.ProposedByField, Required: true, MaxSelect: 1, Values: []string{repertoire.ProposedByTeacher, repertoire.ProposedByLearner}},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	})
	if err != nil {
		return err
	}
	pieces.AddIndex("idx_pieces_assignment", false, "`assignment`", "")
	if err := app.Save(pieces); err != nil {
		return fmt.Errorf("save %s collection: %w", repertoire.CollectionName, err)
	}
	return addMaterialPiece(app, pieces)
}

// addMaterialPiece links a material to at most one piece. Without cascade, deleting the piece clears the link and keeps the material.
func addMaterialPiece(app core.App, pieces *core.Collection) error {
	learnerMaterials, err := findCollection(app, materials.CollectionName)
	if err != nil {
		return fmt.Errorf("find %s collection: %w", materials.CollectionName, err)
	}
	if learnerMaterials.Fields.GetByName(repertoire.MaterialPieceField) == nil {
		learnerMaterials.Fields.Add(&core.RelationField{Name: repertoire.MaterialPieceField, CollectionId: pieces.Id, MaxSelect: 1})
	}
	if err := app.Save(learnerMaterials); err != nil {
		return fmt.Errorf("save %s collection: %w", materials.CollectionName, err)
	}
	return nil
}
