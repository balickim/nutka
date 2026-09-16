// Package commercialapi owns the PocketBase adapter for commercial package records.
// Adapters load complete assignment-scoped state before pure aggregate commands run.
package commercialapi

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

var packageMutationMu sync.Mutex

func newRecordID() string {
	return core.GenerateDefaultRandomId()
}

func contractActive(app core.App, assignment, contract *core.Record, now time.Time) (bool, error) {
	status := contract.GetString(schedulingstore.StatusField)
	if status != "active" && status != "notice_given" {
		return false, nil
	}
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return false, err
	}
	location, err := time.LoadLocation(teacher.GetString("timezone"))
	if err != nil {
		return false, err
	}
	start, startErr := time.Parse("2006-01-02", contract.GetString(schedulingstore.StartOnField))
	endText := contract.GetString(schedulingstore.EffectiveEndOnField)
	if endText == "" {
		endText = contract.GetString(schedulingstore.EndOnField)
	}
	end, endErr := time.Parse("2006-01-02", endText)
	if startErr != nil || endErr != nil {
		return false, errInvalid
	}
	localDate := now.In(location).Format("2006-01-02")
	return localDate >= start.Format("2006-01-02") && localDate <= end.Format("2006-01-02"), nil
}

func ownedAssignment(app core.App, id, role, accountID string) (*core.Record, error) {
	if id == "" {
		return nil, errInvalid
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, id)
	field := "learner"
	if role == "teacher" {
		field = "teacher"
	}
	if (role != "teacher" && role != "learner") || err != nil || assignment.GetString(field) != accountID {
		return nil, errForbidden
	}
	return assignment, nil
}

func ownedPackage(app core.App, id, teacherID string) (*core.Record, error) {
	if id == "" {
		return nil, errInvalid
	}
	row, err := app.FindRecordById(schedulingstore.LessonPackagesCollectionName, id)
	if err != nil {
		return nil, errForbidden
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, row.GetString(schedulingstore.AssignmentField))
	if err != nil || assignment.GetString("teacher") != teacherID {
		return nil, errForbidden
	}
	return row, nil
}

func savePackage(app core.App, value commercial.Package) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.LessonPackagesCollectionName)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Id = value.ID
	encoded, err := json.Marshal(value.Policy)
	if err != nil {
		return nil, err
	}
	record.Set(schedulingstore.AssignmentField, value.Assignment.ID)
	record.Set(schedulingstore.PackageStatusField, string(value.Status))
	record.Set(schedulingstore.PurchasedOnField, value.PurchasedOn.Format("2006-01-02"))
	record.Set(schedulingstore.ValidThroughField, value.ValidThrough.Format("2006-01-02"))
	record.Set(schedulingstore.UnitPriceMinorField, value.Price.Minor)
	record.Set(schedulingstore.CurrencyField, value.Price.Currency)
	record.Set(schedulingstore.PolicyVersionField, value.Policy.Version)
	record.Set(schedulingstore.PolicySnapshotField, encoded)
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func saveTokens(app core.App, value commercial.Package, changedAt time.Time) (map[string]string, error) {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.PackageTokensCollectionName)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]string, len(value.Tokens))
	for _, token := range value.Tokens {
		record := core.NewRecord(collection)
		ids[token.ID] = record.Id
		record.Set(schedulingstore.PackageField, value.ID)
		record.Set(schedulingstore.OrdinalField, token.Ordinal)
		record.Set(schedulingstore.TokenStateField, string(token.State))
		record.Set(schedulingstore.ChangedAtField, changedAt.UTC().Format(time.RFC3339))
		if token.LessonID != "" {
			record.Set(schedulingstore.LessonField, token.LessonID)
		}
		if err := app.Save(record); err != nil {
			return nil, fmt.Errorf("save token %s package=%s: %w", record.Id, record.GetString(schedulingstore.PackageField), err)
		}
	}
	return ids, nil
}

func convertLesson(app core.App, lessonID, packageID string, value *commercial.Package) error {
	lesson, err := app.FindRecordById(schedulingstore.LessonsCollectionName, lessonID)
	if err != nil {
		return err
	}
	for _, token := range value.Tokens {
		if token.LessonID != lessonID {
			continue
		}
		lesson.Set(schedulingstore.PlanTypeField, string(commercial.PackagePlan))
		lesson.Set(schedulingstore.PackageTokenField, token.ID)
		lesson.Set(schedulingstore.PolicyVersionField, value.Policy.Version)
		encoded, encodeErr := json.Marshal(value.Policy)
		if encodeErr != nil {
			return encodeErr
		}
		lesson.Set(schedulingstore.PolicySnapshotField, encoded)
		lesson.Set(schedulingstore.UnitPriceMinorField, 0)
		lesson.Set(schedulingstore.CurrencyField, value.Price.Currency)
		if err := app.Save(lesson); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("lesson %s has no reserved package token", lessonID)
}
