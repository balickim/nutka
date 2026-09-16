// This file provides the PocketBase-backed ledger application boundary.
// Reads enforce assignment ownership and mutations run inside caller-owned transactions.
package ledgerapi

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var (
	errRecordNotFound = errors.New("ledger record was not found")
	errOwnership      = errors.New("ledger ownership failed")
	errNotAdHoc       = errors.New("lesson is not an ad hoc lesson")
)

// PocketBaseService implements Service against the migrated ledger collections.
type PocketBaseService struct {
	app core.App
	now func() time.Time
}

// NewPocketBaseService creates a service using an injectable UTC clock.
func NewPocketBaseService(app core.App, now func() time.Time) *PocketBaseService {
	if now == nil {
		now = time.Now
	}
	return &PocketBaseService{app: app, now: now}
}

func (s *PocketBaseService) instant() time.Time {
	value := s.now()
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func assignmentForActor(app core.App, id string, actor ledger.Actor) (*core.Record, error) {
	if id == "" || (actor.Role != ledger.TeacherActor && actor.Role != ledger.LearnerActor) || actor.ID == "" {
		return nil, errOwnership
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, id)
	if err != nil {
		return nil, errOwnership
	}
	field := "learner"
	if actor.Role == ledger.TeacherActor {
		field = "teacher"
	}
	if assignment.GetString(field) != actor.ID {
		return nil, errOwnership
	}
	return assignment, nil
}

func contractForActor(app core.App, id string, actor ledger.Actor) (*core.Record, *core.Record, error) {
	contract, err := app.FindRecordById(schedulingstore.RegularContractsCollectionName, id)
	if err != nil {
		return nil, nil, errOwnership
	}
	assignment, err := assignmentForActor(app, contract.GetString(schedulingstore.AssignmentField), actor)
	if err != nil {
		return nil, nil, err
	}
	return contract, assignment, nil
}

func lessonForTeacher(app core.App, id string, actor ledger.Actor) (*core.Record, *core.Record, error) {
	lesson, err := app.FindRecordById(schedulingstore.LessonsCollectionName, id)
	if err != nil {
		return nil, nil, errOwnership
	}
	assignment, err := assignmentForActor(app, lesson.GetString(schedulingstore.AssignmentField), actor)
	if err != nil {
		return nil, nil, err
	}
	return lesson, assignment, nil
}

func chargesForAssignments(app core.App, assignments map[string]bool) ([]*core.Record, error) {
	rows, err := app.FindAllRecords(schedulingstore.ChargesCollectionName)
	if err != nil {
		return nil, err
	}
	result := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if assignments[row.GetString(schedulingstore.AssignmentField)] {
			result = append(result, row)
		}
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Id > result[right].Id })
	return result, nil
}

func assignmentsForActor(app core.App, actor ledger.Actor) ([]*core.Record, error) {
	if actor.ID == "" || (actor.Role != ledger.TeacherActor && actor.Role != ledger.LearnerActor) {
		return nil, errOwnership
	}
	field := "learner"
	if actor.Role == ledger.TeacherActor {
		field = "teacher"
	}
	return app.FindAllRecords(schedulingstore.TeacherLearnersCollectionName, dbx.HashExp{field: actor.ID})
}

func capped[T any](values []T, limit int) []T {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func findRecord(app core.App, collection, id string) (*core.Record, error) {
	if id == "" {
		return nil, errRecordNotFound
	}
	row, err := app.FindRecordById(collection, id)
	if err != nil {
		return nil, errRecordNotFound
	}
	return row, nil
}

func transaction(ctx context.Context, app core.App, operation func(core.App) error) error {
	if app == nil {
		return errors.New("ledger app is unavailable")
	}
	return app.RunInTransaction(func(tx core.App) error {
		return operation(tx)
	})
}

func recordTime(row *core.Record, field string) time.Time {
	return row.GetDateTime(field).Time().UTC()
}

func relatedAssignment(app core.App, row *core.Record) (*core.Record, error) {
	assignmentID := row.GetString(schedulingstore.AssignmentField)
	if assignmentID == "" {
		return nil, errOwnership
	}
	return findRecord(app, schedulingstore.TeacherLearnersCollectionName, assignmentID)
}

func wrapRecordError(operation string, err error) error {
	if errors.Is(err, errOwnership) || errors.Is(err, errRecordNotFound) {
		return errOwnership
	}
	return fmt.Errorf("%s: %w", operation, err)
}
