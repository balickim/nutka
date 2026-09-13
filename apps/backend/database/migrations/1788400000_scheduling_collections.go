// Package migrations defines the scheduling collections and their constraints.
// The helpers preserve existing collection identifiers and add only missing fields.
package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

func createSchedulingCollections(app core.App, teachers, learners *core.Collection) error {
	assignment, err := ensureCollection(app, teacherLearnersCollection, []core.Field{
		&core.RelationField{Name: "teacher", CollectionId: teachers.Id, Required: true, CascadeDelete: true},
		&core.RelationField{Name: "learner", CollectionId: learners.Id, Required: true, CascadeDelete: true},
		&core.BoolField{Name: "active"},
		&core.NumberField{Name: "default_duration_minutes", Required: true, OnlyInt: true, Min: floatPtr(15)},
	})
	if err != nil {
		return err
	}
	assignment.AddIndex("idx_teacher_learners_pair", true, "`teacher`, `learner`", "")
	assignment.AddIndex("idx_teacher_learners_teacher_active", false, "`teacher`, `active`", "`active` = TRUE")
	assignment.AddIndex("idx_teacher_learners_learner_active", false, "`learner`, `active`", "`active` = TRUE")
	if err := app.Save(assignment); err != nil {
		return fmt.Errorf("save %s collection: %w", teacherLearnersCollection, err)
	}

	rule, err := ensureCollection(app, availabilityRulesCollection, []core.Field{
		&core.RelationField{Name: "teacher", CollectionId: teachers.Id, Required: true, CascadeDelete: true},
		// PocketBase treats numeric zero as blank for required fields, but Sunday is weekday zero.
		&core.NumberField{Name: "weekday", OnlyInt: true, Min: floatPtr(0), Max: floatPtr(6)},
		&core.TextField{Name: "start_time", Required: true, Pattern: `^(?:[01][0-9]|2[0-3]):(?:00|15|30|45)$`},
		&core.TextField{Name: "end_time", Required: true, Pattern: `^(?:(?:[01][0-9]|2[0-3]):(?:00|15|30|45)|24:00)$`},
		&core.BoolField{Name: "enabled"},
	})
	if err != nil {
		return err
	}
	rule.AddIndex("idx_availability_rules_teacher_weekday", false, "`teacher`, `weekday`", "`enabled` = TRUE")
	if err := app.Save(rule); err != nil {
		return fmt.Errorf("save %s collection: %w", availabilityRulesCollection, err)
	}

	exception, err := ensureCollection(app, availabilityExceptions, []core.Field{
		&core.RelationField{Name: "teacher", CollectionId: teachers.Id, Required: true, CascadeDelete: true},
		&core.DateField{Name: "start_at", Required: true},
		&core.DateField{Name: "end_at", Required: true},
		&core.SelectField{Name: "kind", Values: []string{"available", "unavailable"}, Required: true},
		&core.TextField{Name: "note", Max: 2000},
	})
	if err != nil {
		return err
	}
	exception.AddIndex("idx_availability_exceptions_teacher_range", false, "`teacher`, `start_at`, `end_at`", "")
	if err := app.Save(exception); err != nil {
		return fmt.Errorf("save %s collection: %w", availabilityExceptions, err)
	}

	lesson, err := ensureCollection(app, lessonsCollection, []core.Field{
		&core.RelationField{Name: "teacher", CollectionId: teachers.Id, Required: true},
		&core.RelationField{Name: "learner", CollectionId: learners.Id, Required: true},
		&core.RelationField{Name: "assignment", CollectionId: assignment.Id, Required: true},
		&core.DateField{Name: "start_at", Required: true},
		&core.DateField{Name: "end_at", Required: true},
		&core.NumberField{Name: "duration_minutes", Required: true, OnlyInt: true, Min: floatPtr(15)},
		&core.SelectField{Name: "status", Values: []string{"scheduled", "cancelled"}, Required: true},
		&core.SelectField{Name: "cancellation_initiator_role", Values: []string{"teacher", "learner"}},
		&core.TextField{Name: "cancellation_initiator_id", Max: 15},
		&core.DateField{Name: "cancelled_at"},
	})
	if err != nil {
		return err
	}
	lesson.AddIndex("idx_lessons_teacher_active_interval", false, "`teacher`, `start_at`, `end_at`", "`status` = 'scheduled'")
	lesson.AddIndex("idx_lessons_learner_active_interval", false, "`learner`, `start_at`, `end_at`", "`status` = 'scheduled'")
	if err := app.Save(lesson); err != nil {
		return fmt.Errorf("save %s collection: %w", lessonsCollection, err)
	}
	return createLessonEventsCollection(app, lesson)
}

func createLessonEventsCollection(app core.App, lesson *core.Collection) error {
	events, err := ensureCollection(app, lessonEventsCollection, []core.Field{
		&core.RelationField{Name: "lesson", CollectionId: lesson.Id, Required: true, CascadeDelete: true},
		&core.SelectField{Name: "kind", Values: []string{"created", "rescheduled", "cancelled"}, Required: true},
		&core.SelectField{Name: "initiator_role", Values: []string{"teacher", "learner"}, Required: true},
		&core.TextField{Name: "initiator_id", Required: true, Max: 15},
		&core.DateField{Name: "event_at", Required: true},
		&core.DateField{Name: "prior_start_at"},
		&core.DateField{Name: "prior_end_at"},
		&core.DateField{Name: "new_start_at"},
		&core.DateField{Name: "new_end_at"},
		&core.NumberField{Name: "duration_minutes", Required: true, OnlyInt: true, Min: floatPtr(15)},
		&core.NumberField{Name: "prior_duration_minutes", OnlyInt: true, Min: floatPtr(15)},
		&core.NumberField{Name: "new_duration_minutes", OnlyInt: true, Min: floatPtr(15)},
	})
	if err != nil {
		return err
	}
	events.AddIndex("idx_lesson_events_lesson_event_at", false, "`lesson`, `event_at`", "")
	if err := app.Save(events); err != nil {
		return fmt.Errorf("save %s collection: %w", lessonEventsCollection, err)
	}
	return nil
}

func ensureCollection(app core.App, name string, fields []core.Field) (*core.Collection, error) {
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("inspect %s collection: %w", name, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		collection = core.NewBaseCollection(name)
	}
	if !collection.IsBase() {
		return nil, fmt.Errorf("%s collection must be a base collection", name)
	}
	for _, field := range fields {
		if collection.Fields.GetByName(field.GetName()) == nil {
			collection.Fields.Add(field)
		}
	}
	collection.ListRule = nil
	collection.ViewRule = nil
	collection.CreateRule = nil
	collection.UpdateRule = nil
	collection.DeleteRule = nil
	return collection, nil
}

func findCollection(app core.App, name string) (*core.Collection, error) {
	return app.FindCollectionByNameOrId(name)
}

func floatPtr(value float64) *float64 { return &value }
