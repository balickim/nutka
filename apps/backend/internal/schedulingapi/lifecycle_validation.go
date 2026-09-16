// This file applies shared booking and lifecycle validation against current PocketBase records.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func validateLifecycleInterval(app core.App, now, start time.Time, duration time.Duration, teacher, learner, exclude string) error {
	interval, err := validateInterval(now, start, duration)
	if err != nil {
		return err
	}
	if err := validateAvailability(app, now, interval, teacher); err != nil {
		return err
	}
	return validateParticipantConflicts(app, interval, teacher, learner, exclude)
}

func validateInterval(now, start time.Time, duration time.Duration) (scheduling.Interval, error) {
	interval, err := scheduling.ValidateBookingStart(start, now, duration)
	if err != nil {
		return scheduling.Interval{}, mapValidationError(err)
	}
	return interval, nil
}

func validateAvailability(app core.App, now time.Time, interval scheduling.Interval, teacher string) error {
	teacherRecord, err := app.FindRecordById(authconfig.TeachersCollectionName, teacher)
	if err != nil {
		return errUnauthorized
	}
	ruleRows, err := recordsByField(app, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacher)
	if err != nil {
		return err
	}
	exceptionRows, err := recordsByField(app, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacher)
	if err != nil {
		return err
	}
	rules, err := weeklyRules(ruleRows)
	if err != nil {
		return err
	}
	exceptions, err := domainExceptions(exceptionRows)
	if err != nil {
		return err
	}
	availabilityEnd := scheduling.HorizonEnd(now)
	if interval.End.After(availabilityEnd) {
		availabilityEnd = interval.End
	}
	available, err := scheduling.EffectiveAvailability(now, availabilityEnd, teacherRecord.GetString(schedulingstore.TeacherTimezoneField), rules, exceptions)
	if err != nil {
		return err
	}
	if containsInterval(available, interval) {
		return nil
	}
	return errConflict
}

func validateParticipantConflicts(app core.App, interval scheduling.Interval, teacher, learner, exclude string) error {
	rows, err := participantLessons(app, teacher, learner)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Id == exclude {
			continue
		}
		existingInterval, err := scheduling.NewInterval(row.GetDateTime(schedulingstore.StartAtField).Time().UTC(), row.GetDateTime(schedulingstore.EndAtField).Time().UTC())
		if err != nil {
			return errInvalid
		}
		lesson := scheduling.Lesson{TeacherID: row.GetString("teacher"), LearnerID: row.GetString("learner"), Interval: existingInterval, Status: scheduling.Scheduled}
		if len(scheduling.DetectConflicts(interval, teacher, learner, []scheduling.Lesson{lesson})) > 0 {
			return errConflict
		}
	}
	return nil
}

func mapValidationError(err error) error {
	switch err {
	case scheduling.ErrInvalidDuration:
		return errDuration
	case scheduling.ErrInvalidGrid:
		return errGrid
	case scheduling.ErrHorizon:
		return errHorizon
	default:
		return errInvalid
	}
}

func containsInterval(windows []scheduling.Interval, candidate scheduling.Interval) bool {
	for _, window := range windows {
		if window.Contains(candidate) {
			return true
		}
	}
	return false
}

func participantLessons(app core.App, teacher, learner string) ([]*core.Record, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		return nil, err
	}
	result := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if row.GetString(schedulingstore.StatusField) != string(scheduling.Scheduled) {
			continue
		}
		if row.GetString("teacher") == teacher || row.GetString("learner") == learner {
			result = append(result, row)
		}
	}
	return result, nil
}

func saveLessonEvent(app core.App, lesson *core.Record, kind, role, initiator string, eventAt, priorStart, priorEnd, newStart, newEnd time.Time, duration time.Duration, priorDuration int) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.LessonEventsCollectionName)
	if err != nil {
		return errInvalid
	}
	event := core.NewRecord(collection)
	event.Set(schedulingstore.LessonField, lesson.Id)
	event.Set(schedulingstore.KindField, kind)
	event.Set(schedulingstore.InitiatorRoleField, role)
	event.Set(schedulingstore.InitiatorIDField, initiator)
	event.Set(schedulingstore.EventAtField, eventAt.UTC().Format(time.RFC3339))
	event.Set(schedulingstore.DurationMinutesField, int(duration/time.Minute))
	if setEventInterval(event, schedulingstore.PriorStartAtField, schedulingstore.PriorEndAtField, priorStart, priorEnd) && priorDuration > 0 {
		event.Set(schedulingstore.PriorDurationMinutesField, priorDuration)
	}
	if setEventInterval(event, schedulingstore.NewStartAtField, schedulingstore.NewEndAtField, newStart, newEnd) {
		event.Set(schedulingstore.NewDurationMinutesField, int(duration/time.Minute))
	}
	if err := app.Save(event); err != nil {
		return errInvalid
	}
	return nil
}

func setEventInterval(event *core.Record, startField, endField string, start, end time.Time) bool {
	if start.IsZero() || end.IsZero() {
		return false
	}
	event.Set(startField, start.UTC().Format(time.RFC3339))
	event.Set(endField, end.UTC().Format(time.RFC3339))
	return true
}
