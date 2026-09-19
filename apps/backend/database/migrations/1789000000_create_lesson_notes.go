// Package migrations creates the closed lesson notes collection: one sanitized teacher note per lesson with optional material links.
// Only superusers can use its native record routes.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/lessonnotes"
	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(createLessonNotes, func(core.App) error { return nil })
}

func createLessonNotes(app core.App) error {
	related := map[string]*core.Collection{}
	for _, name := range []string{schedulingstore.LessonsCollectionName, teacherLearnersCollection, materials.CollectionName} {
		collection, err := findCollection(app, name)
		if err != nil {
			return fmt.Errorf("find %s collection: %w", name, err)
		}
		related[name] = collection
	}
	collection, err := ensureCollection(app, lessonnotes.CollectionName, []core.Field{
		&core.RelationField{Name: lessonnotes.LessonField, CollectionId: related[schedulingstore.LessonsCollectionName].Id, Required: true, CascadeDelete: true, MaxSelect: 1},
		&core.RelationField{Name: lessonnotes.AssignmentField, CollectionId: related[teacherLearnersCollection].Id, Required: true, CascadeDelete: true, MaxSelect: 1},
		&core.EditorField{Name: lessonnotes.BodyField, Required: true, MaxSize: lessonnotes.BodyMaxBytes},
		&core.RelationField{Name: lessonnotes.MaterialsField, CollectionId: related[materials.CollectionName].Id, MaxSelect: lessonnotes.MaxMaterials},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	})
	if err != nil {
		return err
	}
	collection.AddIndex("idx_lesson_notes_lesson", true, "`lesson`", "")
	collection.AddIndex("idx_lesson_notes_assignment", false, "`assignment`", "")
	if err := app.Save(collection); err != nil {
		return fmt.Errorf("save %s collection: %w", lessonnotes.CollectionName, err)
	}
	return nil
}
