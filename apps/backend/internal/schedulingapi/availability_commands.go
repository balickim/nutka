// This file provides the default PocketBase adapter for explicit teacher lesson lifecycle commands.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/history"
	lessondomain "github.com/balickim/nutka/apps/backend/internal/lesson"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/pocketbase/pocketbase/core"
)

type pocketBaseLessonCommands struct{}

func (pocketBaseLessonCommands) Cancel(app core.App, lesson *core.Record, actorID string, now time.Time) error {
	command, err := lifecycleCommand(app, lesson, history.Actor{Role: history.TeacherActor, ID: actorID}, now.UTC(), time.Time{})
	if err != nil {
		return err
	}
	decision, err := lessondomain.CancelLesson(command)
	if err != nil {
		return err
	}
	return persistLifecycleDecision(app, lesson, decision, now.UTC())
}

func (pocketBaseLessonCommands) Reschedule(app core.App, lesson *core.Record, replacement time.Time, availability []scheduling.Interval, actorID string, now time.Time) error {
	command, err := lifecycleCommand(app, lesson, history.Actor{Role: history.TeacherActor, ID: actorID}, now.UTC(), replacement.UTC())
	if err != nil {
		return err
	}
	command.Availability = availability
	decision, err := lessondomain.RescheduleLesson(command)
	if err != nil {
		return err
	}
	return persistLifecycleDecision(app, lesson, decision, now.UTC())
}
