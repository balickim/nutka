// Package migrations audits legacy scheduling records and preserves their history during commercial migration.
// Legacy concrete timestamps remain UTC and teacher-local dates are normalized to explicit date strings.
package migrations

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func auditAndBackfillScheduling(app core.App, assignments, lessons *core.Collection) error {
	assignmentRows, err := app.FindAllRecords(assignments)
	if err != nil {
		return fmt.Errorf("read assignments for duration audit: %w", err)
	}
	lessonRows, err := app.FindAllRecords(lessons)
	if err != nil {
		return fmt.Errorf("read lessons for duration audit: %w", err)
	}
	nonStandardAssignments, nonStandardLessons := countNonStandardDurations(assignmentRows, lessonRows)
	if nonStandardAssignments > 0 || nonStandardLessons > 0 {
		app.Logger().Warn("normalizing legacy non-45-minute scheduling records", "assignments", nonStandardAssignments, "lessons", nonStandardLessons)
	}
	for _, row := range assignmentRows {
		if row.GetInt(schedulingstore.DefaultDurationMinutesField) == commercialLessonMinutes {
			continue
		}
		row.Set(schedulingstore.DefaultDurationMinutesField, commercialLessonMinutes)
		if err := app.Save(row); err != nil {
			return fmt.Errorf("normalize assignment %s duration: %w", row.Id, err)
		}
	}
	now := time.Now().UTC()
	for _, row := range lessonRows {
		if err := backfillLesson(app, row, now); err != nil {
			return err
		}
	}
	return nil
}

func countNonStandardDurations(assignments, lessons []*core.Record) (int, int) {
	assignmentCount, lessonCount := 0, 0
	for _, row := range assignments {
		if value := row.GetInt(schedulingstore.DefaultDurationMinutesField); value != 0 && value != commercialLessonMinutes {
			assignmentCount++
		}
	}
	for _, row := range lessons {
		if value := row.GetInt(schedulingstore.DurationMinutesField); value != 0 && value != commercialLessonMinutes {
			lessonCount++
		}
	}
	return assignmentCount, lessonCount
}

func backfillLesson(app core.App, lesson *core.Record, now time.Time) error {
	start := lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()
	if start.IsZero() {
		return fmt.Errorf("lesson %s has no start time", lesson.Id)
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, lesson.GetString(schedulingstore.AssignmentField))
	if err != nil {
		return fmt.Errorf("find assignment for lesson %s: %w", lesson.Id, err)
	}
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return fmt.Errorf("find teacher for lesson %s: %w", lesson.Id, err)
	}
	location, err := time.LoadLocation(teacher.GetString(schedulingstore.TeacherTimezoneField))
	if err != nil {
		return fmt.Errorf("lesson %s has invalid teacher timezone %q: %w", lesson.Id, teacher.GetString(schedulingstore.TeacherTimezoneField), err)
	}
	end := start.Add(time.Duration(commercialLessonMinutes) * time.Minute)
	status := lesson.GetString(schedulingstore.StatusField)
	lesson.Set(schedulingstore.DurationMinutesField, commercialLessonMinutes)
	lesson.Set(schedulingstore.EndAtField, end.Format(time.RFC3339Nano))
	lesson.Set(schedulingstore.PlanTypeField, "ad_hoc")
	lesson.Set(schedulingstore.PackageTokenField, "")
	lesson.Set(schedulingstore.ContractField, "")
	lesson.Set(schedulingstore.OriginalLocalDateField, start.In(location).Format("2006-01-02"))
	lesson.Set(schedulingstore.OriginalStartAtField, start.Format(time.RFC3339Nano))
	lesson.Set(schedulingstore.PolicyVersionField, commercialPolicyVersion)
	lesson.Set(schedulingstore.PolicySnapshotField, policySnapshotJSON())
	lesson.Set(schedulingstore.UnitPriceMinorField, commercialLessonPrice)
	lesson.Set(schedulingstore.CurrencyField, commercialCurrency)
	lesson.Set(schedulingstore.ScheduleStateField, status)
	lesson.Set(schedulingstore.OutcomeField, "")
	lesson.Set(schedulingstore.OutcomeAtField, types.DateTime{})
	lesson.Set(schedulingstore.BillingOutcomeField, "")
	lesson.Set(schedulingstore.OmissionReasonField, "")
	if status != "cancelled" && !end.After(now) {
		lesson.Set(schedulingstore.OutcomeField, "awaiting_outcome")
	}
	lesson.Set(schedulingstore.IndividuallyRescheduledField, false)
	if err := app.Save(lesson); err != nil {
		return fmt.Errorf("backfill lesson %s: %w", lesson.Id, err)
	}
	return nil
}

func policySnapshotJSON() string {
	raw, _ := json.Marshal(businesspolicy.CurrentSnapshot())
	return string(raw)
}

func backfillBusinessEvents(app core.App, lessons *core.Collection) error {
	events, err := app.FindCollectionByNameOrId(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		return fmt.Errorf("find business events collection: %w", err)
	}
	legacyEvents, err := app.FindCollectionByNameOrId(schedulingstore.LessonEventsCollectionName)
	if err != nil {
		return nil
	}
	rows, err := app.FindAllRecords(legacyEvents)
	if err != nil {
		return fmt.Errorf("read legacy lesson events: %w", err)
	}
	for _, old := range rows {
		if existing, _ := app.FindFirstRecordByData(events, schedulingstore.LegacyEventIDField, old.Id); existing != nil {
			continue
		}
		event, err := migratedBusinessEvent(app, events, lessons, old)
		if err != nil {
			return err
		}
		if err := app.Save(event); err != nil {
			return fmt.Errorf("backfill legacy lesson event %s: %w", old.Id, err)
		}
	}
	return nil
}

func migratedBusinessEvent(app core.App, events, lessons *core.Collection, old *core.Record) (*core.Record, error) {
	lesson, err := app.FindRecordById(lessons.Id, old.GetString(schedulingstore.LessonField))
	if err != nil {
		return nil, fmt.Errorf("find lesson for legacy event %s: %w", old.Id, err)
	}
	event := core.NewRecord(events)
	event.Set(schedulingstore.AssignmentField, lesson.GetString(schedulingstore.AssignmentField))
	event.Set(schedulingstore.AggregateTypeField, "lesson")
	event.Set(schedulingstore.AggregateIDField, lesson.Id)
	event.Set(schedulingstore.EventTypeField, migratedEventType(old.GetString(schedulingstore.KindField)))
	event.Set(schedulingstore.ActorRoleField, migratedActorRole(old.GetString(schedulingstore.InitiatorRoleField)))
	event.Set(schedulingstore.ActorIDField, old.GetString(schedulingstore.InitiatorIDField))
	event.Set(schedulingstore.EventAtField, old.GetDateTime(schedulingstore.EventAtField).Time().UTC().Format(time.RFC3339Nano))
	event.Set(schedulingstore.RelatedIDsField, map[string]string{"legacy_event_id": old.Id, "lesson_id": lesson.Id})
	event.Set(schedulingstore.LegacyEventIDField, old.Id)
	return event, nil
}

func migratedEventType(kind string) string {
	value := map[string]string{"created": "lesson_created", "rescheduled": "lesson_rescheduled", "cancelled": "lesson_cancelled"}[kind]
	if value == "" {
		return "lesson_legacy_event"
	}
	return value
}

func migratedActorRole(role string) string {
	if role == "teacher" || role == "learner" {
		return role
	}
	return "system"
}

func removeLegacyDurationField(app core.App, assignments *core.Collection) error {
	if assignments.Fields.GetByName(schedulingstore.DefaultDurationMinutesField) == nil {
		return nil
	}
	assignments.Fields.RemoveByName(schedulingstore.DefaultDurationMinutesField)
	if err := app.Save(assignments); err != nil {
		return fmt.Errorf("remove legacy assignment duration: %w", err)
	}
	return nil
}
