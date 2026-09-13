// This file atomically reschedules and cancels authorized future lessons while retaining immutable event history.
package schedulingapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type rescheduleInput struct {
	StartAt         *string         `json:"start_at"`
	DurationMinutes json.RawMessage `json:"duration_minutes"`
}

func rescheduleLesson(e *core.RequestEvent, role string, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	account, err := caller(e, role)
	if err != nil {
		return handleError(e, err)
	}
	var input rescheduleInput
	if err := bindBody(e, &input); err != nil {
		return handleError(e, err)
	}
	if input.StartAt == nil && input.DurationMinutes == nil {
		return handleError(e, errInvalid)
	}
	now := clock().UTC()
	var result *core.Record
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		lesson, lookupErr := findAuthorizedLesson(tx, e.Request.PathValue("id"), role, account.Id)
		if lookupErr != nil {
			return lookupErr
		}
		assignment, assignmentErr := tx.FindRecordById(schedulingstore.TeacherLearnersCollectionName, lesson.GetString(schedulingstore.AssignmentField))
		if assignmentErr != nil || !assignment.GetBool(schedulingstore.ActiveField) {
			return errForbidden
		}
		if !lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC().After(now) || lesson.GetString(schedulingstore.StatusField) != string(scheduling.Scheduled) {
			return errConflict
		}
		start, duration, validationErr := rescheduleValues(lesson, role, input)
		if validationErr != nil {
			return validationErr
		}
		if err := validateLifecycleInterval(tx, now, start, duration, lesson.GetString("teacher"), lesson.GetString("learner"), lesson.Id); err != nil {
			return err
		}
		priorStart := lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()
		priorEnd := lesson.GetDateTime(schedulingstore.EndAtField).Time().UTC()
		priorDuration := lesson.GetInt(schedulingstore.DurationMinutesField)
		lesson.Set(schedulingstore.StartAtField, start.Format(time.RFC3339))
		lesson.Set(schedulingstore.EndAtField, start.Add(duration).Format(time.RFC3339))
		lesson.Set(schedulingstore.DurationMinutesField, int(duration/time.Minute))
		if err := tx.Save(lesson); err != nil {
			return errConflict
		}
		if err := saveLessonEvent(tx, lesson, "rescheduled", role, account.Id, now, priorStart, priorEnd, start, start.Add(duration), duration, priorDuration); err != nil {
			return err
		}
		result = lesson
		return nil
	})
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, lessonValue(result))
}

func rescheduleValues(lesson *core.Record, role string, input rescheduleInput) (time.Time, time.Duration, error) {
	start := lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()
	if input.StartAt != nil {
		var err error
		start, err = parseInstant(*input.StartAt)
		if err != nil {
			return time.Time{}, 0, err
		}
	}
	duration := time.Duration(lesson.GetInt(schedulingstore.DurationMinutesField)) * time.Minute
	if input.DurationMinutes != nil {
		if role != "teacher" {
			return time.Time{}, 0, errInvalid
		}
		var minutes int
		if err := json.Unmarshal(input.DurationMinutes, &minutes); err != nil {
			return time.Time{}, 0, errInvalid
		}
		duration = time.Duration(minutes) * time.Minute
	}
	if err := scheduling.ValidateLessonDuration(duration); err != nil {
		return time.Time{}, 0, errDuration
	}
	return start, duration, nil
}

func findAuthorizedLesson(app core.App, id, role, account string) (*core.Record, error) {
	if id == "" {
		return nil, errInvalid
	}
	row, err := app.FindRecordById(schedulingstore.LessonsCollectionName, id)
	if err != nil {
		return nil, errForbidden
	}
	field := "learner"
	if role == "teacher" {
		field = "teacher"
	}
	if row.GetString(field) != account {
		return nil, errForbidden
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, row.GetString(schedulingstore.AssignmentField))
	if err != nil || assignment.GetString("teacher") != row.GetString("teacher") || assignment.GetString("learner") != row.GetString("learner") {
		return nil, errForbidden
	}
	return row, nil
}

func cancelLesson(e *core.RequestEvent, role string, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	account, err := caller(e, role)
	if err != nil {
		return handleError(e, err)
	}
	if e.Request.Body != nil && e.Request.ContentLength != 0 {
		var input struct{}
		if err := bindBody(e, &input); err != nil {
			return handleError(e, err)
		}
	}
	now := clock().UTC()
	var result *core.Record
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		lesson, lookupErr := findAuthorizedLesson(tx, e.Request.PathValue("id"), role, account.Id)
		if lookupErr != nil {
			return lookupErr
		}
		if lesson.GetString(schedulingstore.StatusField) != string(scheduling.Scheduled) || !lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC().After(now) {
			return errConflict
		}
		start := lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()
		end := lesson.GetDateTime(schedulingstore.EndAtField).Time().UTC()
		duration := time.Duration(lesson.GetInt(schedulingstore.DurationMinutesField)) * time.Minute
		lesson.Set(schedulingstore.StatusField, string(scheduling.Cancelled))
		lesson.Set(schedulingstore.CancellationInitiatorRoleField, role)
		lesson.Set(schedulingstore.CancellationInitiatorIDField, account.Id)
		lesson.Set(schedulingstore.CancelledAtField, now.Format(time.RFC3339))
		if err := tx.Save(lesson); err != nil {
			return errInvalid
		}
		if err := saveLessonEvent(tx, lesson, "cancelled", role, account.Id, now, start, end, time.Time{}, time.Time{}, duration, int(duration/time.Minute)); err != nil {
			return err
		}
		result = lesson
		return nil
	})
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, lessonValue(result))
}
