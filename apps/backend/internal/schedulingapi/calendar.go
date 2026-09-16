// This file serves isolated teacher and learner calendars, slot reads, and retained lesson views.
package schedulingapi

import (
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func teacherCalendar(e *core.RequestEvent) error { return teacherCalendarAt(e, time.Now) }

func teacherCalendarAt(e *core.RequestEvent, clock Clock) error {
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	assignments, err := ownedAssignments(e.App, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	rules, err := recordsByField(e.App, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	exceptions, err := recordsByField(e.App, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	lessons, err := lessonsFor(e.App, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	value, valueErr := calendarValue(e.App, assignments, enabledRules(rules), exceptions, lessons, "teacher", clock().UTC())
	if valueErr != nil {
		return handleError(e, valueErr)
	}
	return e.JSON(http.StatusOK, value)
}

func learnerCalendar(e *core.RequestEvent) error { return learnerCalendarAt(e, time.Now) }

func learnerCalendarAt(e *core.RequestEvent, clock Clock) error {
	learner, err := caller(e, "learner")
	if err != nil {
		return handleError(e, err)
	}
	assignments, err := ownedAssignments(e.App, "learner", learner.Id)
	if err != nil {
		return handleError(e, err)
	}
	lessons, err := lessonsFor(e.App, "learner", learner.Id)
	if err != nil {
		return handleError(e, err)
	}
	rules := make([]*core.Record, 0)
	exceptions := make([]*core.Record, 0)
	seen := map[string]bool{}
	for _, assignment := range assignments {
		if !assignment.GetBool(schedulingstore.ActiveField) {
			continue
		}
		teacherID := assignment.GetString("teacher")
		if seen[teacherID] {
			continue
		}
		seen[teacherID] = true
		rows, queryErr := recordsByField(e.App, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacherID)
		if queryErr != nil {
			return handleError(e, queryErr)
		}
		rules = append(rules, enabledRules(rows)...)
		rows, queryErr = recordsByField(e.App, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacherID)
		if queryErr != nil {
			return handleError(e, queryErr)
		}
		exceptions = append(exceptions, enabledExceptions(rows)...)
	}
	value, valueErr := calendarValue(e.App, assignments, rules, exceptions, lessons, "learner", clock().UTC())
	if valueErr != nil {
		return handleError(e, valueErr)
	}
	return e.JSON(http.StatusOK, value)
}

func learnerSlotsAt(e *core.RequestEvent, clock Clock) error {
	learner, err := caller(e, "learner")
	if err != nil {
		return handleError(e, err)
	}
	id := e.Request.PathValue("id")
	assignment, err := ownedAssignment(e.App, id, "learner", learner.Id)
	if err != nil {
		return handleError(e, err)
	}
	if !assignment.GetBool(schedulingstore.ActiveField) {
		return handleError(e, errForbidden)
	}
	teacherID := assignment.GetString("teacher")
	now := clock().UTC()
	state, err := loadBookingState(e.App, assignment, now, now)
	if err != nil {
		return bookingError(e, err)
	}
	if state.contract != nil {
		return e.JSON(http.StatusOK, map[string]any{"teacher": teacherID, "assignment": assignment.Id, "policy_version": businesspolicy.CurrentVersion, "slots": []slotDTO{}})
	}
	duration := businesspolicy.Current().LessonDuration
	slots, err := generateLearnerSlots(e.App, teacherID, learner.Id, duration, func() time.Time { return now })
	if err != nil {
		if err == scheduling.ErrHorizon {
			return handleError(e, errHorizon)
		}
		return handleError(e, errInvalid)
	}
	items := make([]slotDTO, 0, len(slots))
	for _, slot := range slots {
		items = append(items, slotDTO{StartAt: utcString(slot.Interval.Start), EndAt: utcString(slot.Interval.End), Duration: int(duration / time.Minute), Protected: &protectedDTO{StartAt: utcString(slot.Protected.Start), EndAt: utcString(slot.Protected.End)}})
	}
	return e.JSON(http.StatusOK, map[string]any{"teacher": teacherID, "assignment": assignment.Id, "policy_version": businesspolicy.CurrentVersion, "slots": items})
}

func generateLearnerSlots(app core.App, teacherID, learnerID string, duration time.Duration, clock Clock) ([]scheduling.Slot, error) {
	teacher, err := app.FindRecordById(authconfig.TeachersCollectionName, teacherID)
	if err != nil {
		return nil, errUnauthorized
	}
	rulesRows, err := recordsByField(app, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacherID)
	if err != nil {
		return nil, err
	}
	exceptionRows, err := recordsByField(app, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacherID)
	if err != nil {
		return nil, err
	}
	lessonRows, err := activeLessons(app, teacherID, learnerID)
	if err != nil {
		return nil, err
	}
	rules, err := weeklyRules(rulesRows)
	if err != nil {
		return nil, err
	}
	exceptions, err := domainExceptions(exceptionRows)
	if err != nil {
		return nil, err
	}
	lessons, err := domainLessons(lessonRows)
	if err != nil {
		return nil, err
	}
	return scheduling.GenerateSlots(clock().UTC(), teacher.GetString(schedulingstore.TeacherTimezoneField), rules, exceptions, teacherID, learnerID, lessons, duration)
}
