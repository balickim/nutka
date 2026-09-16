// This file blocks ordinary teacher timezone changes while active contracts or future scheduled lessons exist.
package schedulingstore

import (
	"errors"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// ErrTimezoneChangeBlocked identifies an ordinary profile edit that would rewrite future schedule identities.
var ErrTimezoneChangeBlocked = errors.New("timezone change is blocked by future obligations")

// RegisterTimezoneGuard installs the ordinary profile-editing timezone integrity rule.
func RegisterTimezoneGuard(app core.App) {
	app.OnRecordValidate().Bind(&hook.Handler[*core.RecordEvent]{
		Id: "nutkaTeacherTimezoneObligationGuard",
		Func: func(e *core.RecordEvent) error {
			if e.Record.Collection().Name != authconfig.TeachersCollectionName || e.Record.Original().Id == "" || e.Record.GetString(TeacherTimezoneField) == e.Record.Original().GetString(TeacherTimezoneField) {
				return e.Next()
			}
			blocked, err := HasTimezoneChangeObligation(e.App, e.Record.Id, time.Now().UTC())
			if err != nil {
				return err
			}
			if blocked {
				return ErrTimezoneChangeBlocked
			}
			return e.Next()
		},
	})
}

// HasTimezoneChangeObligation reports whether an ordinary timezone edit would alter a live schedule.
func HasTimezoneChangeObligation(app core.App, teacherID string, now time.Time) (bool, error) {
	blocked, err := hasActiveTeacherContract(app, teacherID)
	if err != nil || blocked {
		return blocked, err
	}
	return hasFutureTeacherLesson(app, teacherID, now)
}

func hasActiveTeacherContract(app core.App, teacherID string) (bool, error) {
	contracts, err := app.FindCollectionByNameOrId(RegularContractsCollectionName)
	if err == nil {
		rows, findErr := app.FindAllRecords(contracts.Id)
		if findErr != nil {
			return false, findErr
		}
		for _, row := range rows {
			if row.GetString(AssignmentField) == "" || (row.GetString(StatusField) == "active" || row.GetString(StatusField) == "notice_given") {
				assignment, assignmentErr := app.FindRecordById(TeacherLearnersCollectionName, row.GetString(AssignmentField))
				if assignmentErr == nil && assignment.GetString("teacher") == teacherID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func hasFutureTeacherLesson(app core.App, teacherID string, now time.Time) (bool, error) {
	lessons, err := app.FindCollectionByNameOrId(LessonsCollectionName)
	if err != nil {
		return false, nil
	}
	rows, err := app.FindAllRecords(lessons.Id)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.GetString("teacher") != teacherID || row.GetString(StatusField) != "scheduled" || row.GetString(ScheduleStateField) == "omitted" {
			continue
		}
		if row.GetDateTime(StartAtField).Time().UTC().After(now.UTC()) {
			return true, nil
		}
	}
	return false, nil
}
