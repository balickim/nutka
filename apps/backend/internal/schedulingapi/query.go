// This file loads owned scheduling records and translates them to pure domain values.
package schedulingapi

import (
	"strconv"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func ownedAssignments(app core.App, role, id string) ([]*core.Record, error) {
	field := "learner"
	if role == "teacher" {
		field = "teacher"
	}
	return app.FindAllRecords(schedulingstore.TeacherLearnersCollectionName, dbx.HashExp{field: id})
}
func findAssignment(app core.App, id string) (*core.Record, error) {
	if id == "" {
		return nil, errInvalid
	}
	r, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, id)
	if err != nil {
		return nil, errForbidden
	}
	return r, nil
}
func ownedAssignment(app core.App, id, role, callerID string) (*core.Record, error) {
	r, err := findAssignment(app, id)
	if err != nil {
		return nil, err
	}
	field := "learner"
	if role == "teacher" {
		field = "teacher"
	}
	if r.GetString(field) != callerID {
		return nil, errForbidden
	}
	return r, nil
}
func recordsByField(app core.App, collection, field, value string) ([]*core.Record, error) {
	return app.FindAllRecords(collection, dbx.HashExp{field: value})
}
func activeLessons(app core.App, teacher, learner string) ([]*core.Record, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		return nil, err
	}
	return selectActiveLessons(rows, teacher, learner), nil
}
func selectActiveLessons(rows []*core.Record, teacher, learner string) []*core.Record {
	result := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if row.GetString(schedulingstore.StatusField) != "scheduled" {
			continue
		}
		teacherMatch := teacher != "" && row.GetString("teacher") == teacher
		learnerMatch := learner != "" && row.GetString("learner") == learner
		if !teacherMatch && !learnerMatch {
			continue
		}
		result = append(result, row)
	}
	return result
}
func lessonsFor(app core.App, role, id string) ([]*core.Record, error) {
	field := "learner"
	if role == "teacher" {
		field = "teacher"
	}
	return recordsByField(app, schedulingstore.LessonsCollectionName, field, id)
}
func unavailableConflict(app core.App, teacher string, start, end time.Time) (bool, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.GetString("teacher") != teacher || row.GetString(schedulingstore.StatusField) != "scheduled" {
			continue
		}
		lessonStart := row.GetDateTime(schedulingstore.StartAtField).Time().UTC()
		lessonEnd := row.GetDateTime(schedulingstore.EndAtField).Time().UTC()
		if lessonStart.Before(end) && start.Before(lessonEnd) {
			return true, nil
		}
	}
	return false, nil
}
func weeklyRules(rows []*core.Record) ([]scheduling.WeekdayRule, error) {
	rules := make([]scheduling.WeekdayRule, 0, len(rows))
	for _, row := range rows {
		start, end, err := localMinutes(row.GetString(schedulingstore.StartTimeField), row.GetString(schedulingstore.EndTimeField))
		if err != nil {
			return nil, errInvalid
		}
		rules = append(rules, scheduling.WeekdayRule{Weekday: time.Weekday(row.GetInt(schedulingstore.WeekdayField)), StartMinute: start, EndMinute: end, Enabled: row.GetBool(schedulingstore.EnabledField)})
	}
	return rules, nil
}
func localMinutes(start, end string) (int, int, error) {
	s, err := parseLocalMinute(start, false)
	if err != nil {
		return 0, 0, err
	}
	e, err := parseLocalMinute(end, true)
	if err != nil || s >= e {
		return 0, 0, errInvalid
	}
	return s, e, nil
}

func parseLocalMinute(value string, allow24 bool) (int, error) {
	if allow24 && value == "24:00" {
		return 1440, nil
	}
	if !validLocalMinuteShape(value) {
		return 0, errInvalid
	}
	hour, hourErr := strconv.Atoi(value[:2])
	minute, minuteErr := strconv.Atoi(value[3:])
	if hourErr != nil || minuteErr != nil || hour > 23 || minute > 59 {
		return 0, errInvalid
	}
	if minute%15 != 0 {
		return 0, errGrid
	}
	return hour*60 + minute, nil
}

func isDigit(value byte) bool { return value >= '0' && value <= '9' }

func validLocalMinuteShape(value string) bool {
	return len(value) == 5 && value[2] == ':' && isDigit(value[0]) && isDigit(value[1]) && isDigit(value[3]) && isDigit(value[4])
}
func domainExceptions(rows []*core.Record) ([]scheduling.AvailabilityException, error) {
	items := make([]scheduling.AvailabilityException, 0, len(rows))
	for _, row := range rows {
		if !row.GetBool(schedulingstore.EnabledField) {
			continue
		}
		start, end := row.GetDateTime(schedulingstore.StartAtField).Time().UTC(), row.GetDateTime(schedulingstore.EndAtField).Time().UTC()
		interval, err := scheduling.NewInterval(start, end)
		if err != nil {
			return nil, errInvalid
		}
		items = append(items, scheduling.AvailabilityException{Interval: interval, Kind: scheduling.ExceptionKind(row.GetString(schedulingstore.KindField)), Note: row.GetString(schedulingstore.NoteField)})
	}
	return items, nil
}
func domainLessons(rows []*core.Record) ([]scheduling.Lesson, error) {
	items := make([]scheduling.Lesson, 0, len(rows))
	for _, row := range rows {
		interval, err := scheduling.NewInterval(row.GetDateTime(schedulingstore.StartAtField).Time().UTC(), row.GetDateTime(schedulingstore.EndAtField).Time().UTC())
		if err != nil {
			return nil, errInvalid
		}
		items = append(items, scheduling.Lesson{TeacherID: row.GetString("teacher"), LearnerID: row.GetString("learner"), Interval: interval, Status: scheduling.LessonStatus(row.GetString(schedulingstore.StatusField))})
	}
	return items, nil
}
func ensureNoIdentityFields(data map[string]any) error {
	for key := range data {
		if strings.EqualFold(key, "teacher") || strings.EqualFold(key, "learner") || strings.EqualFold(key, "actor_id") || strings.EqualFold(key, "actor_role") {
			return errForbidden
		}
	}
	return nil
}
