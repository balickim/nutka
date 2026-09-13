// This file manages teacher-owned recurring rules and concrete UTC availability exceptions.
package schedulingapi

import (
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type ruleInput struct {
	Weekday   *int    `json:"weekday"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Enabled   *bool   `json:"enabled"`
	Teacher   string  `json:"teacher"`
	Learner   string  `json:"learner"`
	ActorID   string  `json:"actor_id"`
	ActorRole string  `json:"actor_role"`
}

func teacherRules(e *core.RequestEvent) error {
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	rows, err := recordsByField(e.App, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	items := make([]ruleDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, ruleValue(row))
	}
	return e.JSON(http.StatusOK, map[string]any{"availability_rules": items})
}

func createRule(e *core.RequestEvent) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	var input ruleInput
	if bindErr := bindBody(e, &input); bindErr != nil {
		return handleError(e, bindErr)
	}
	start, end, err := ruleValues(input)
	if err != nil {
		return handleError(e, err)
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	collection, err := e.App.FindCollectionByNameOrId(schedulingstore.AvailabilityRulesCollectionName)
	if err != nil {
		return handleError(e, err)
	}
	row := core.NewRecord(collection)
	row.Set("teacher", teacher.Id)
	row.Set(schedulingstore.WeekdayField, *input.Weekday)
	row.Set(schedulingstore.StartTimeField, formatLocalMinute(start))
	row.Set(schedulingstore.EndTimeField, formatLocalMinute(end))
	row.Set(schedulingstore.EnabledField, enabled)
	if err := e.App.Save(row); err != nil {
		return handleError(e, errInvalid)
	}
	return e.JSON(http.StatusCreated, ruleValue(row))
}

func updateRule(e *core.RequestEvent) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	var input ruleInput
	if bindErr := bindBody(e, &input); bindErr != nil {
		return handleError(e, bindErr)
	}
	if input.Weekday == nil && input.StartTime == nil && input.EndTime == nil && input.Enabled == nil {
		return handleError(e, errInvalid)
	}
	var result *core.Record
	err = e.App.RunInTransaction(func(tx core.App) error {
		row, lookupErr := ownedRecord(tx, schedulingstore.AvailabilityRulesCollectionName, e.Request.PathValue("id"), "teacher", teacher.Id)
		if lookupErr != nil {
			return lookupErr
		}
		if input.Weekday != nil {
			if *input.Weekday < 0 || *input.Weekday > 6 {
				return errInvalid
			}
			row.Set(schedulingstore.WeekdayField, *input.Weekday)
		}
		if input.StartTime != nil {
			row.Set(schedulingstore.StartTimeField, *input.StartTime)
		}
		if input.EndTime != nil {
			row.Set(schedulingstore.EndTimeField, *input.EndTime)
		}
		if input.Enabled != nil {
			row.Set(schedulingstore.EnabledField, *input.Enabled)
		}
		start, end, parseErr := localMinutes(row.GetString(schedulingstore.StartTimeField), row.GetString(schedulingstore.EndTimeField))
		if parseErr != nil {
			return parseErr
		}
		row.Set(schedulingstore.StartTimeField, formatLocalMinute(start))
		row.Set(schedulingstore.EndTimeField, formatLocalMinute(end))
		if saveErr := tx.Save(row); saveErr != nil {
			return errInvalid
		}
		result = row
		return nil
	})
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, ruleValue(result))
}

func deleteRule(e *core.RequestEvent) error {
	return deleteOwned(e, schedulingstore.AvailabilityRulesCollectionName)
}

func ruleValues(input ruleInput) (int, int, error) {
	if input.Weekday == nil || input.StartTime == nil || input.EndTime == nil {
		return 0, 0, errInvalid
	}
	if *input.Weekday < 0 || *input.Weekday > 6 {
		return 0, 0, errInvalid
	}
	return localMinutes(*input.StartTime, *input.EndTime)
}

func formatLocalMinute(value int) string {
	if value == 1440 {
		return "24:00"
	}
	return time.Date(0, 1, 1, value/60, value%60, 0, 0, time.UTC).Format("15:04")
}
