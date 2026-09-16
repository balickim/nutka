package schedulingapi

import (
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func TestFutureScheduledLessonUsesStrictStartBoundary(t *testing.T) {
	collection := core.NewBaseCollection("lessons")
	lesson := core.NewRecord(collection)
	lesson.Set(schedulingstore.StatusField, "scheduled")
	now := time.Date(2030, time.January, 2, 12, 0, 0, 0, time.UTC)
	lesson.Set(schedulingstore.StartAtField, now.Format(time.RFC3339Nano))
	if futureScheduledLesson(lesson, now) {
		t.Fatal("lesson starting exactly now must not block deactivation")
	}
	lesson.Set(schedulingstore.StartAtField, now.Add(time.Nanosecond).Format(time.RFC3339Nano))
	if !futureScheduledLesson(lesson, now) {
		t.Fatal("lesson starting after now must block deactivation")
	}
}
