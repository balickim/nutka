// This file defines pure command inputs for contract activation and occurrence changes.
package regularcontract

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

type ActivationRequest struct {
	ID, AssignmentID, TeacherID, LearnerID, TeacherTimezone string
	Actor                                                   Actor
	StartOn                                                 time.Time
	Weekday                                                 time.Weekday
	StartMinute                                             int
	Now                                                     time.Time
	Policy                                                  Policy
	Availability                                            []scheduling.Interval
	Unavailable                                             []scheduling.Interval
	ExistingLessons                                         []scheduling.Lesson
	ExistingCommercialObligation                            bool
	BackdateReason                                          string
	PastOutcomes                                            map[string]string
}

type RescheduleRequest struct {
	OccurrenceID, AssignmentID string
	ReplacementStart           time.Time
	Now                        time.Time
	Actor                      Actor
	Available                  []scheduling.Interval
	ParticipantLessons         []scheduling.Lesson
	ParticipantLessonRecords   []ParticipantLesson
}

// ParticipantLesson pairs a stable lesson identity with its conflict interval.
type ParticipantLesson struct {
	ID        string
	TeacherID string
	LearnerID string
	Interval  scheduling.Interval
	Status    scheduling.LessonStatus
}

type CancellationRequest struct {
	OccurrenceID string
	Now          time.Time
	Actor        Actor
}

type ScheduleChangeRequest struct {
	Weekday      time.Weekday
	StartMinute  int
	EffectiveOn  time.Time
	Now          time.Time
	Actor        Actor
	Availability []scheduling.Interval
	Unavailable  []scheduling.Interval
	Lessons      []scheduling.Lesson
}

type PriceAmendmentRequest struct {
	EffectiveMonth string
	PriceMinor     int64
	Currency       string
	Now            time.Time
	Actor          Actor
}
