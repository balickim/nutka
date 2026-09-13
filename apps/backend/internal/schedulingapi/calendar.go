// This file serves isolated teacher and learner calendars, slot reads, and retained lesson views.
package schedulingapi

import (
	"net/http"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func teacherCalendar(e *core.RequestEvent) error {
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
	value, valueErr := calendarValue(e.App, assignments, enabledRules(rules), exceptions, lessons, teacher.Id, "teacher")
	if valueErr != nil {
		return handleError(e, valueErr)
	}
	return e.JSON(http.StatusOK, value)
}

func learnerCalendar(e *core.RequestEvent) error {
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
		exceptions = append(exceptions, rows...)
	}
	value, valueErr := calendarValue(e.App, assignments, rules, exceptions, lessons, learner.Id, "learner")
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
	duration := time.Duration(assignment.GetInt(schedulingstore.DefaultDurationMinutesField)) * time.Minute
	slots, err := generateLearnerSlots(e.App, teacherID, learner.Id, duration, clock)
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
	return e.JSON(http.StatusOK, map[string]any{"teacher": teacherID, "assignment": assignment.Id, "slots": items})
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

func calendarValue(app core.App, assignments, rules, exceptions, lessons []*core.Record, accountID, role string) (calendarDTO, error) {
	result := calendarDTO{Assignments: make([]assignmentDTO, 0, len(assignments)), Rules: make([]ruleDTO, 0, len(rules)), Exceptions: make([]exceptionDTO, 0, len(exceptions)), Lessons: make([]lessonDTO, 0, len(lessons))}
	for _, row := range assignments {
		item, err := assignmentValue(app, row)
		if err != nil {
			return calendarDTO{}, err
		}
		result.Assignments = append(result.Assignments, item)
	}
	for _, row := range rules {
		result.Rules = append(result.Rules, ruleValue(row))
	}
	for _, row := range exceptions {
		result.Exceptions = append(result.Exceptions, exceptionValue(row))
	}
	for _, row := range lessons {
		result.Lessons = append(result.Lessons, lessonValue(row))
	}
	result.Counters = cancellationCounters(lessons, accountID, role)
	sort.Slice(result.Lessons, func(i, j int) bool { return result.Lessons[i].StartAt < result.Lessons[j].StartAt })
	return result, nil
}
func enabledRules(rows []*core.Record) []*core.Record {
	result := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if row.GetBool(schedulingstore.EnabledField) {
			result = append(result, row)
		}
	}
	return result
}
func cancellationCounters(rows []*core.Record, accountID, role string) countersDTO {
	result := countersDTO{}
	for _, row := range rows {
		if row.GetString(schedulingstore.StatusField) != "cancelled" || row.GetString(schedulingstore.CancellationInitiatorIDField) != accountID {
			continue
		}
		if row.GetString(schedulingstore.CancellationInitiatorRoleField) == role {
			if role == "teacher" {
				result.Teacher++
			} else {
				result.Learner++
			}
		}
	}
	return result
}
