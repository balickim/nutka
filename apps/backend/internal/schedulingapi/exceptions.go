// This file manages teacher-owned concrete UTC availability exceptions and block conflicts.
package schedulingapi

import (
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type exceptionInput struct {
	StartAt   *string `json:"start_at"`
	EndAt     *string `json:"end_at"`
	Kind      *string `json:"kind"`
	Note      *string `json:"note"`
	Teacher   string  `json:"teacher"`
	Learner   string  `json:"learner"`
	ActorID   string  `json:"actor_id"`
	ActorRole string  `json:"actor_role"`
}

func teacherExceptions(e *core.RequestEvent) error {
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	rows, err := recordsByField(e.App, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	items := make([]exceptionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, exceptionValue(row))
	}
	return e.JSON(http.StatusOK, map[string]any{"availability_exceptions": items})
}
func createException(e *core.RequestEvent) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	var input exceptionInput
	if bindErr := bindBody(e, &input); bindErr != nil {
		return handleError(e, bindErr)
	}
	if input.StartAt == nil || input.EndAt == nil || input.Kind == nil {
		return handleError(e, errInvalid)
	}
	start, end, parseErr := exceptionInterval(*input.StartAt, *input.EndAt, *input.Kind)
	if parseErr != nil {
		return handleError(e, parseErr)
	}
	row, saveErr := saveException(e.App, teacher.Id, start, end, *input.Kind, input.Note)
	if saveErr != nil {
		return handleError(e, saveErr)
	}
	return e.JSON(http.StatusCreated, exceptionValue(row))
}
func updateException(e *core.RequestEvent) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	var input exceptionInput
	if bindErr := bindBody(e, &input); bindErr != nil {
		return handleError(e, bindErr)
	}
	if input.StartAt == nil && input.EndAt == nil && input.Kind == nil && input.Note == nil {
		return handleError(e, errInvalid)
	}
	var result *core.Record
	err = e.App.RunInTransaction(func(tx core.App) error {
		var txErr error
		result, txErr = updateExceptionTx(tx, e.Request.PathValue("id"), teacher.Id, input)
		return txErr
	})
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, exceptionValue(result))
}

func updateExceptionTx(tx core.App, id, teacher string, input exceptionInput) (*core.Record, error) {
	row, err := ownedRecord(tx, schedulingstore.AvailabilityExceptionsCollectionName, id, "teacher", teacher)
	if err != nil {
		return nil, err
	}
	start, end, kind, err := updatedExceptionValues(row, input)
	if err != nil {
		return nil, err
	}
	if kind == "unavailable" {
		conflict, err := unavailableConflict(tx, teacher, start, end)
		if err != nil {
			return nil, err
		}
		if conflict {
			return nil, errConflict
		}
	}
	row.Set(schedulingstore.StartAtField, start.Format(time.RFC3339))
	row.Set(schedulingstore.EndAtField, end.Format(time.RFC3339))
	row.Set(schedulingstore.KindField, kind)
	if input.Note != nil {
		row.Set(schedulingstore.NoteField, *input.Note)
	}
	if err := tx.Save(row); err != nil {
		return nil, errInvalid
	}
	return row, nil
}

func updatedExceptionValues(row *core.Record, input exceptionInput) (time.Time, time.Time, string, error) {
	start := row.GetDateTime(schedulingstore.StartAtField).Time().UTC()
	end := row.GetDateTime(schedulingstore.EndAtField).Time().UTC()
	kind := row.GetString(schedulingstore.KindField)
	if input.StartAt != nil || input.EndAt != nil {
		var err error
		start, end, err = updatedExceptionInterval(input)
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}
	}
	if input.Kind != nil {
		kind = *input.Kind
	}
	if start.IsZero() || !start.Before(end) || (kind != "available" && kind != "unavailable") {
		return time.Time{}, time.Time{}, "", errInvalid
	}
	return start, end, kind, nil
}

func updatedExceptionInterval(input exceptionInput) (time.Time, time.Time, error) {
	if input.StartAt == nil || input.EndAt == nil {
		return time.Time{}, time.Time{}, errInvalid
	}
	start, err := parseInstant(*input.StartAt)
	if err != nil {
		return time.Time{}, time.Time{}, errInvalid
	}
	end, err := parseInstant(*input.EndAt)
	if err != nil {
		return time.Time{}, time.Time{}, errInvalid
	}
	return start, end, nil
}
func deleteException(e *core.RequestEvent) error {
	return deleteOwned(e, schedulingstore.AvailabilityExceptionsCollectionName)
}
func exceptionInterval(startValue, endValue, kind string) (time.Time, time.Time, error) {
	start, err := parseInstant(startValue)
	if err != nil {
		return time.Time{}, time.Time{}, errInvalid
	}
	end, err := parseInstant(endValue)
	if err != nil || !start.Before(end) {
		return time.Time{}, time.Time{}, errInvalid
	}
	if kind != "available" && kind != "unavailable" {
		return time.Time{}, time.Time{}, errInvalid
	}
	return start, end, nil
}
func saveException(app core.App, teacher string, start, end time.Time, kind string, note *string) (*core.Record, error) {
	var result *core.Record
	err := app.RunInTransaction(func(tx core.App) error { return saveExceptionTx(tx, teacher, start, end, kind, note, &result) })
	if err != nil {
		return nil, err
	}
	return result, nil
}

func saveExceptionTx(tx core.App, teacher string, start, end time.Time, kind string, note *string, result **core.Record) error {
	if kind == "unavailable" {
		conflict, err := unavailableConflict(tx, teacher, start, end)
		if err != nil {
			return err
		}
		if conflict {
			return errConflict
		}
	}
	collection, err := tx.FindCollectionByNameOrId(schedulingstore.AvailabilityExceptionsCollectionName)
	if err != nil {
		return errInvalid
	}
	row := core.NewRecord(collection)
	row.Set("teacher", teacher)
	row.Set(schedulingstore.StartAtField, start.Format(time.RFC3339))
	row.Set(schedulingstore.EndAtField, end.Format(time.RFC3339))
	row.Set(schedulingstore.KindField, kind)
	if note != nil {
		row.Set(schedulingstore.NoteField, *note)
	}
	if err := tx.Save(row); err != nil {
		return errInvalid
	}
	*result = row
	return nil
}
