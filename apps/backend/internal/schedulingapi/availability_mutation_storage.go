// This file persists normalized recurring-rule and exception mutations after preview validation.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/availabilityimpact"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func saveRuleMutation(app core.App, teacherID string, mutation AvailabilityMutation, result *CommitResult) error {
	row, err := ruleMutationRecord(app, teacherID, mutation)
	if err != nil {
		return err
	}
	if err := app.Save(row); err != nil {
		return errInvalid
	}
	result.Rule = row
	return nil
}

func ruleMutationRecord(app core.App, teacherID string, mutation AvailabilityMutation) (*core.Record, error) {
	if mutation.Operation == availabilityimpact.Create {
		collection, err := app.FindCollectionByNameOrId(schedulingstore.AvailabilityRulesCollectionName)
		if err != nil {
			return nil, errInvalid
		}
		row := core.NewRecord(collection)
		row.Set("teacher", teacherID)
		return applyNewRuleValues(row, mutation.Rule)
	}
	row, err := ownedRecord(app, schedulingstore.AvailabilityRulesCollectionName, mutation.ID, "teacher", teacherID)
	if err != nil {
		return nil, err
	}
	day, start, end, enabled, err := ruleRecordValues(row)
	if err != nil {
		return nil, err
	}
	day, start, end, enabled, err = updatedRuleValues(day, start, end, enabled, mutation.Rule, mutation.Operation)
	if err != nil {
		return nil, err
	}
	setRuleValues(row, day, start, end, enabled)
	return row, nil
}

func applyNewRuleValues(row *core.Record, value *RuleMutation) (*core.Record, error) {
	day, start, end, enabled, err := newRuleValues(value)
	if err != nil {
		return nil, err
	}
	setRuleValues(row, day, start, end, enabled)
	return row, nil
}

func setRuleValues(row *core.Record, day, start, end int, enabled bool) {
	row.Set(schedulingstore.WeekdayField, day)
	row.Set(schedulingstore.StartTimeField, formatLocalMinute(start))
	row.Set(schedulingstore.EndTimeField, formatLocalMinute(end))
	row.Set(schedulingstore.EnabledField, enabled)
}

func saveExceptionMutation(app core.App, teacherID string, mutation AvailabilityMutation, result *CommitResult) error {
	row, err := exceptionMutationRecord(app, teacherID, mutation)
	if err != nil {
		return err
	}
	if err := app.Save(row); err != nil {
		return errInvalid
	}
	result.Exception = row
	return nil
}

func exceptionMutationRecord(app core.App, teacherID string, mutation AvailabilityMutation) (*core.Record, error) {
	if mutation.Operation == availabilityimpact.Create {
		collection, err := app.FindCollectionByNameOrId(schedulingstore.AvailabilityExceptionsCollectionName)
		if err != nil {
			return nil, errInvalid
		}
		row := core.NewRecord(collection)
		row.Set("teacher", teacherID)
		return applyNewExceptionValues(row, mutation.Exception)
	}
	row, err := ownedRecord(app, schedulingstore.AvailabilityExceptionsCollectionName, mutation.ID, "teacher", teacherID)
	if err != nil {
		return nil, err
	}
	start, end, kind, note, enabled, err := exceptionRecordValues(row)
	if err != nil {
		return nil, err
	}
	start, end, kind, note, enabled, err = updatedExceptionValuesForMutation(start, end, kind, note, enabled, mutation.Exception, mutation.Operation)
	if err != nil {
		return nil, err
	}
	setExceptionValues(row, start, end, kind, note, enabled)
	return row, nil
}

func applyNewExceptionValues(row *core.Record, value *ExceptionMutation) (*core.Record, error) {
	start, end, kind, note, enabled, err := newExceptionValues(value)
	if err != nil {
		return nil, err
	}
	setExceptionValues(row, start, end, kind, note, enabled)
	return row, nil
}

func setExceptionValues(row *core.Record, start, end time.Time, kind, note string, enabled bool) {
	row.Set(schedulingstore.StartAtField, start.Format(time.RFC3339))
	row.Set(schedulingstore.EndAtField, end.Format(time.RFC3339))
	row.Set(schedulingstore.KindField, kind)
	row.Set(schedulingstore.NoteField, note)
	row.Set(schedulingstore.EnabledField, enabled)
}
