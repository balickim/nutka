package schedulingapi

import (
	"testing"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func TestSelectActiveLessonsProtectsBothParticipantCalendars(t *testing.T) {
	collection := core.NewBaseCollection("lessons")
	teacherCollision := core.NewRecord(collection)
	teacherCollision.Set("teacher", "teacher-1")
	teacherCollision.Set("learner", "learner-other")
	teacherCollision.Set(schedulingstore.StatusField, "scheduled")
	learnerCollision := core.NewRecord(collection)
	learnerCollision.Set("teacher", "teacher-other")
	learnerCollision.Set("learner", "learner-1")
	learnerCollision.Set(schedulingstore.StatusField, "scheduled")
	unrelated := core.NewRecord(collection)
	unrelated.Set("teacher", "teacher-other")
	unrelated.Set("learner", "learner-other")
	unrelated.Set(schedulingstore.StatusField, "scheduled")
	cancelled := core.NewRecord(collection)
	cancelled.Set("teacher", "teacher-1")
	cancelled.Set("learner", "learner-1")
	cancelled.Set(schedulingstore.StatusField, "cancelled")
	rows := selectActiveLessons([]*core.Record{teacherCollision, learnerCollision, unrelated, cancelled}, "teacher-1", "learner-1")
	if len(rows) != 2 || rows[0] == unrelated || rows[1] == unrelated {
		t.Fatalf("participant union selected wrong lessons: %d", len(rows))
	}
	if rows = selectActiveLessons([]*core.Record{teacherCollision, learnerCollision, unrelated}, "teacher-1", ""); len(rows) != 1 || rows[0] != teacherCollision {
		t.Fatalf("teacher-only selection: %d", len(rows))
	}
	if rows = selectActiveLessons([]*core.Record{teacherCollision, learnerCollision, unrelated}, "", "learner-1"); len(rows) != 1 || rows[0] != learnerCollision {
		t.Fatalf("learner-only selection: %d", len(rows))
	}
	if rows = selectActiveLessons([]*core.Record{teacherCollision, learnerCollision, unrelated}, "", ""); len(rows) != 0 {
		t.Fatalf("empty participant selection: %d", len(rows))
	}
}

func TestLocalMinutesRejectsNegativeHoursAndMinutes(t *testing.T) {
	for _, input := range [][2]string{
		{"-1:00", "01:00"},
		{"-12:00", "01:00"},
		{"00:-15", "01:00"},
		{"00:00", "-0:15"},
		{"00:00", "00:-15"},
	} {
		if start, end, err := localMinutes(input[0], input[1]); err == nil {
			t.Fatalf("accepted negative rule %q-%q as %d-%d", input[0], input[1], start, end)
		}
	}
}
