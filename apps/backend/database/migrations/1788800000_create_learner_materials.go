// Package migrations creates the closed learner materials collection with a sanitized editor body and protected image and PDF files.
// Only superusers can use its native record routes.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(createLearnerMaterials, func(core.App) error { return nil })
}

func createLearnerMaterials(app core.App) error {
	assignments, err := findCollection(app, teacherLearnersCollection)
	if err != nil {
		return fmt.Errorf("find %s collection: %w", teacherLearnersCollection, err)
	}
	collection, err := ensureCollection(app, materials.CollectionName, []core.Field{
		&core.RelationField{Name: materials.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true, MaxSelect: 1},
		&core.TextField{Name: materials.TitleField, Required: true, Max: materials.TitleMaxLength},
		&core.EditorField{Name: materials.BodyField, MaxSize: materials.BodyMaxBytes},
		&core.FileField{
			Name:      materials.AttachmentsField,
			MaxSelect: materials.MaxFiles,
			MaxSize:   materials.MaxFileBytes,
			MimeTypes: materials.AllowedMimeTypes,
			Protected: true,
		},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	})
	if err != nil {
		return err
	}
	collection.AddIndex("idx_learner_materials_assignment_created", false, "`assignment`, `created`", "")
	if err := app.Save(collection); err != nil {
		return fmt.Errorf("save %s collection: %w", materials.CollectionName, err)
	}
	return nil
}
