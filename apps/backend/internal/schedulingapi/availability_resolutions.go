// This file validates and applies explicit teacher resolutions for near-term availability conflicts.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/availabilityimpact"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func validateResolutions(conflicts []availabilityimpact.NearTermConflict, resolutions []Resolution) error {
	if len(conflicts) != len(resolutions) {
		return errIncompletePreview
	}
	allowed := make(map[string]availabilityimpact.NearTermConflict, len(conflicts))
	for _, conflict := range conflicts {
		allowed[conflict.LessonID] = conflict
	}
	seen := make(map[string]bool, len(resolutions))
	for _, resolution := range resolutions {
		_, allowedLesson := allowed[resolution.LessonID]
		validAction := resolution.Action == availabilityimpact.Cancel || resolution.Action == availabilityimpact.Reschedule
		if !allowedLesson || seen[resolution.LessonID] || !validAction {
			return errInvalidResolution
		}
		seen[resolution.LessonID] = true
		if resolution.Action == availabilityimpact.Reschedule && resolution.ReplacementStart.IsZero() {
			return errInvalidResolution
		}
	}
	return nil
}

func applyResolutions(app core.App, lessons map[string]*core.Record, resolutions []Resolution, commands LessonCommands, teacherID string, now time.Time, availability []scheduling.Interval, policy businesspolicy.Policy) error {
	for _, resolution := range resolutions {
		if err := applyResolution(app, lessons[resolution.LessonID], resolution, commands, teacherID, now, availability, policy); err != nil {
			return err
		}
	}
	return nil
}

func applyResolution(app core.App, lesson *core.Record, resolution Resolution, commands LessonCommands, teacherID string, now time.Time, availability []scheduling.Interval, policy businesspolicy.Policy) error {
	if lesson == nil {
		return errStalePreview
	}
	if resolution.Action == availabilityimpact.Cancel {
		return commands.Cancel(app, lesson, teacherID, now)
	}
	start := resolution.ReplacementStart.UTC()
	interval, err := scheduling.NewInterval(start, start.Add(policy.LessonDuration))
	if err != nil || !validReplacement(start, now, interval, availability, policy) {
		return errInvalidResolution
	}
	if err := replacementConflict(app, lesson, interval); err != nil {
		return err
	}
	return commands.Reschedule(app, lesson, start, availability, teacherID, now)
}

func validReplacement(start, now time.Time, interval scheduling.Interval, availability []scheduling.Interval, policy businesspolicy.Policy) bool {
	return start.After(now.UTC()) && !start.After(now.UTC().Add(policy.BookingHorizon)) && scheduling.IsOnGrid(start, policy.StartGrid) && containsInterval(availability, interval)
}

func replacementConflict(app core.App, current *core.Record, candidate scheduling.Interval) error {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		return err
	}
	teacher, learner := current.GetString("teacher"), current.GetString("learner")
	for _, row := range rows {
		if !potentialReplacementConflict(row, current.Id, teacher, learner) {
			continue
		}
		interval, err := scheduling.NewInterval(row.GetDateTime(schedulingstore.StartAtField).Time().UTC(), row.GetDateTime(schedulingstore.EndAtField).Time().UTC())
		if err != nil {
			return errInvalid
		}
		if candidate.Protected(businesspolicy.Current().ParticipantBuffer).Intersects(interval.Protected(businesspolicy.Current().ParticipantBuffer)) {
			return errConflict
		}
	}
	return nil
}

func potentialReplacementConflict(row *core.Record, currentID, teacherID, learnerID string) bool {
	participant := row.GetString("teacher") == teacherID || row.GetString("learner") == learnerID
	return row.Id != currentID && row.GetString(schedulingstore.StatusField) == string(scheduling.Scheduled) && participant
}
