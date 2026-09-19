// Package migrations creates the closed practice tasks and practice sessions collections.
// Only superusers can use their native record routes.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/practice"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(createPractice, func(core.App) error { return nil })
}

func createPractice(app core.App) error {
	related := map[string]*core.Collection{}
	for _, name := range []string{teacherLearnersCollection, schedulingstore.LessonsCollectionName, repertoire.CollectionName, materials.CollectionName} {
		collection, err := findCollection(app, name)
		if err != nil {
			return fmt.Errorf("find %s collection: %w", name, err)
		}
		related[name] = collection
	}
	tasks, err := createPracticeTasks(app, related)
	if err != nil {
		return err
	}
	return createPracticeSessions(app, related[teacherLearnersCollection], tasks)
}

// Lesson, piece, and material links have no cascade, so deleting the target clears the link and keeps the task.
func createPracticeTasks(app core.App, related map[string]*core.Collection) (*core.Collection, error) {
	tasks, err := ensureCollection(app, practice.TasksCollectionName, []core.Field{
		&core.RelationField{Name: practice.AssignmentField, CollectionId: related[teacherLearnersCollection].Id, Required: true, CascadeDelete: true, MaxSelect: 1},
		&core.RelationField{Name: practice.LessonField, CollectionId: related[schedulingstore.LessonsCollectionName].Id, MaxSelect: 1},
		&core.TextField{Name: practice.TitleField, Required: true, Max: practice.TitleMaxLength},
		&core.TextField{Name: practice.DetailsField, Max: practice.DetailsMaxLength},
		&core.NumberField{Name: practice.SuggestedMinutesField, OnlyInt: true, Min: floatPtr(0), Max: floatPtr(practice.MaxSuggestedMinutes)},
		&core.RelationField{Name: practice.PieceField, CollectionId: related[repertoire.CollectionName].Id, MaxSelect: 1},
		&core.RelationField{Name: practice.MaterialField, CollectionId: related[materials.CollectionName].Id, MaxSelect: 1},
		&core.SelectField{Name: practice.StatusField, Required: true, MaxSelect: 1, Values: practice.Statuses},
		&core.NumberField{Name: practice.PositionField, OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	})
	if err != nil {
		return nil, err
	}
	tasks.AddIndex("idx_practice_tasks_assignment_status", false, "`assignment`, `status`", "")
	if err := app.Save(tasks); err != nil {
		return nil, fmt.Errorf("save %s collection: %w", practice.TasksCollectionName, err)
	}
	return tasks, nil
}

func createPracticeSessions(app core.App, assignments, tasks *core.Collection) error {
	sessions, err := ensureCollection(app, practice.SessionsCollectionName, []core.Field{
		&core.RelationField{Name: practice.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true, MaxSelect: 1},
		&core.TextField{Name: practice.PracticedOnField, Required: true, Pattern: `^\d{4}-\d{2}-\d{2}$`},
		&core.NumberField{Name: practice.MinutesField, OnlyInt: true, Min: floatPtr(0), Max: floatPtr(practice.MaxSessionMinutes)},
		&core.RelationField{Name: practice.TasksField, CollectionId: tasks.Id, MaxSelect: practice.MaxSessionTasks},
		&core.TextField{Name: practice.CommentField, Max: practice.CommentMaxLength},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	})
	if err != nil {
		return err
	}
	sessions.AddIndex("idx_practice_sessions_assignment_day", false, "`assignment`, `practiced_on`", "")
	if err := app.Save(sessions); err != nil {
		return fmt.Errorf("save %s collection: %w", practice.SessionsCollectionName, err)
	}
	return nil
}
