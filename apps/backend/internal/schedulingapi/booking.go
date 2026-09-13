// This file validates and atomically creates learner bookings from active assignment defaults.
package schedulingapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type bookingInput struct {
	StartAt         *string         `json:"start_at"`
	DurationMinutes json.RawMessage `json:"duration_minutes"`
}

func learnerBookingAt(e *core.RequestEvent, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	learner, err := caller(e, "learner")
	if err != nil {
		return handleError(e, err)
	}
	var input bookingInput
	if err := bindBody(e, &input); err != nil {
		return handleError(e, err)
	}
	if input.StartAt == nil || input.DurationMinutes != nil {
		return handleError(e, errInvalid)
	}
	start, err := parseInstant(*input.StartAt)
	if err != nil {
		return handleError(e, err)
	}
	now := clock().UTC()
	var result *core.Record
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		assignment, lookupErr := ownedAssignment(tx, e.Request.PathValue("id"), "learner", learner.Id)
		if lookupErr != nil || !assignment.GetBool(schedulingstore.ActiveField) {
			return errForbidden
		}
		teacherID := assignment.GetString("teacher")
		duration := time.Duration(assignment.GetInt(schedulingstore.DefaultDurationMinutesField)) * time.Minute
		if err := validateLifecycleInterval(tx, now, start, duration, teacherID, learner.Id, ""); err != nil {
			return err
		}
		collection, collectionErr := tx.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
		if collectionErr != nil {
			return errInvalid
		}
		lesson := core.NewRecord(collection)
		lesson.Set("teacher", teacherID)
		lesson.Set("learner", learner.Id)
		lesson.Set(schedulingstore.AssignmentField, assignment.Id)
		lesson.Set(schedulingstore.StartAtField, start.Format(time.RFC3339))
		lesson.Set(schedulingstore.EndAtField, start.Add(duration).Format(time.RFC3339))
		lesson.Set(schedulingstore.DurationMinutesField, int(duration/time.Minute))
		lesson.Set(schedulingstore.StatusField, string(scheduling.Scheduled))
		if err := tx.Save(lesson); err != nil {
			return errConflict
		}
		if err := saveLessonEvent(tx, lesson, "created", "learner", learner.Id, now, time.Time{}, time.Time{}, start, start.Add(duration), duration, 0); err != nil {
			return err
		}
		result = lesson
		return nil
	})
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusCreated, lessonValue(result))
}
