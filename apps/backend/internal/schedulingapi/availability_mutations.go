// This file validates availability mutation values and persists normalized rule or exception records.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/availabilityimpact"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func ruleRecordValues(row *core.Record) (int, int, int, bool, error) {
	start, end, err := localMinutes(row.GetString(schedulingstore.StartTimeField), row.GetString(schedulingstore.EndTimeField))
	if err != nil {
		return 0, 0, 0, false, err
	}
	day := row.GetInt(schedulingstore.WeekdayField)
	if day < 0 || day > 6 {
		return 0, 0, 0, false, errInvalid
	}
	return day, start, end, row.GetBool(schedulingstore.EnabledField), nil
}

func newRuleValues(value *RuleMutation) (int, int, int, bool, error) {
	if value == nil || value.Weekday == nil || value.StartTime == nil || value.EndTime == nil {
		return 0, 0, 0, false, errInvalid
	}
	if *value.Weekday < 0 || *value.Weekday > 6 {
		return 0, 0, 0, false, errInvalid
	}
	start, end, err := localMinutes(*value.StartTime, *value.EndTime)
	if err != nil {
		return 0, 0, 0, false, err
	}
	enabled := true
	if value.Enabled != nil {
		enabled = *value.Enabled
	}
	return *value.Weekday, start, end, enabled, nil
}

func updatedRuleValues(day, start, end int, enabled bool, value *RuleMutation, operation availabilityimpact.Operation) (int, int, int, bool, error) {
	if operation == availabilityimpact.Enable {
		return day, start, end, true, nil
	}
	if operation == availabilityimpact.Disable {
		return day, start, end, false, nil
	}
	if operation != availabilityimpact.Update || value == nil {
		return 0, 0, 0, false, errInvalid
	}
	day = updatedWeekday(day, value)
	if day < 0 || day > 6 {
		return 0, 0, 0, false, errInvalid
	}
	start, end, err := updatedRuleInterval(start, end, value)
	if err != nil {
		return 0, 0, 0, false, err
	}
	if start >= end {
		return 0, 0, 0, false, errInvalid
	}
	if value.Enabled != nil {
		enabled = *value.Enabled
	}
	return day, start, end, enabled, nil
}

func updatedWeekday(day int, value *RuleMutation) int {
	if value.Weekday != nil {
		return *value.Weekday
	}
	return day
}

func updatedRuleInterval(start, end int, value *RuleMutation) (int, int, error) {
	var err error
	if value.StartTime != nil {
		start, err = parseLocalMinute(*value.StartTime, false)
	}
	if err == nil && value.EndTime != nil {
		end, err = parseLocalMinute(*value.EndTime, true)
	}
	return start, end, err
}

func exceptionRecordValues(row *core.Record) (time.Time, time.Time, string, string, bool, error) {
	start := row.GetDateTime(schedulingstore.StartAtField).Time().UTC()
	end := row.GetDateTime(schedulingstore.EndAtField).Time().UTC()
	if start.IsZero() || !start.Before(end) {
		return time.Time{}, time.Time{}, "", "", false, errInvalid
	}
	kind := row.GetString(schedulingstore.KindField)
	if kind != "available" && kind != "unavailable" {
		return time.Time{}, time.Time{}, "", "", false, errInvalid
	}
	return start, end, kind, row.GetString(schedulingstore.NoteField), row.GetBool(schedulingstore.EnabledField), nil
}

func newExceptionValues(value *ExceptionMutation) (time.Time, time.Time, string, string, bool, error) {
	if value == nil || value.StartAt == nil || value.EndAt == nil || value.Kind == nil {
		return time.Time{}, time.Time{}, "", "", false, errInvalid
	}
	start, err := parseInstant(*value.StartAt)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", false, errInvalid
	}
	end, err := parseInstant(*value.EndAt)
	if err != nil || !validExceptionValues(start, end, *value.Kind) {
		return time.Time{}, time.Time{}, "", "", false, errInvalid
	}
	note, enabled := optionalExceptionValues(value)
	return start, end, *value.Kind, note, enabled, nil
}

func optionalExceptionValues(value *ExceptionMutation) (string, bool) {
	note, enabled := "", true
	if value.Note != nil {
		note = *value.Note
	}
	if value.Enabled != nil {
		enabled = *value.Enabled
	}
	return note, enabled
}

func validExceptionValues(start, end time.Time, kind string) bool {
	return start.Before(end) && (kind == "available" || kind == "unavailable")
}

func updatedExceptionValuesForMutation(start, end time.Time, kind, note string, enabled bool, value *ExceptionMutation, operation availabilityimpact.Operation) (time.Time, time.Time, string, string, bool, error) {
	if operation == availabilityimpact.Enable {
		return start, end, kind, note, true, nil
	}
	if operation == availabilityimpact.Disable {
		return start, end, kind, note, false, nil
	}
	if operation != availabilityimpact.Update || value == nil {
		return time.Time{}, time.Time{}, "", "", false, errInvalid
	}
	start, end, err := updatedExceptionInterval(start, end, value)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", false, err
	}
	kind, note, enabled = updatedExceptionOptions(kind, note, enabled, value)
	if !validExceptionValues(start, end, kind) {
		return time.Time{}, time.Time{}, "", "", false, errInvalid
	}
	return start, end, kind, note, enabled, nil
}

func updatedExceptionInterval(start, end time.Time, value *ExceptionMutation) (time.Time, time.Time, error) {
	if value.StartAt == nil && value.EndAt == nil {
		return start, end, nil
	}
	return newExceptionInterval(value.StartAt, value.EndAt)
}

func updatedExceptionOptions(kind, note string, enabled bool, value *ExceptionMutation) (string, string, bool) {
	if value.Kind != nil {
		kind = *value.Kind
	}
	if value.Note != nil {
		note = *value.Note
	}
	if value.Enabled != nil {
		enabled = *value.Enabled
	}
	return kind, note, enabled
}

func newExceptionInterval(start, end *string) (time.Time, time.Time, error) {
	if start == nil || end == nil {
		return time.Time{}, time.Time{}, errInvalid
	}
	left, err := parseInstant(*start)
	if err != nil {
		return time.Time{}, time.Time{}, errInvalid
	}
	right, err := parseInstant(*end)
	if err != nil || !left.Before(right) {
		return time.Time{}, time.Time{}, errInvalid
	}
	return left, right, nil
}

func formatLocalMinute(value int) string {
	if value == 1440 {
		return "24:00"
	}
	return time.Date(0, 1, 1, value/60, value%60, 0, 0, time.UTC).Format("15:04")
}
