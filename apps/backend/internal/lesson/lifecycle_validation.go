// This file validates lifecycle timing, replacement intervals, and participant ownership.
package lesson

import (
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"time"
)

func (c Command) validate() error {
	if !validLessonIdentity(c.Lesson) || c.Now.IsZero() || !c.Lesson.Interval.Valid() {
		return ErrInvalidCommand
	}
	return c.validateActor()
}

func validLessonIdentity(value Lesson) bool {
	return value.ID != "" && value.AssignmentID != "" && value.TeacherID != "" && value.LearnerID != "" && value.Plan.Valid()
}

func (c Command) validateActor() error {
	if c.Actor.Role != history.TeacherActor && c.Actor.Role != history.LearnerActor {
		return ErrUnauthorized
	}
	if c.Actor.Role == history.TeacherActor && c.Actor.ID != c.Lesson.TeacherID {
		return ErrUnauthorized
	}
	if c.Actor.Role == history.LearnerActor && c.Actor.ID != c.Lesson.LearnerID {
		return ErrUnauthorized
	}
	return nil
}

func (c Command) futureScheduled() error {
	if c.Lesson.ScheduleState == domain.CancelledState {
		return ErrCancelled
	}
	if c.Lesson.ScheduleState != domain.ScheduledState || !c.Lesson.Interval.Start.After(c.Now) {
		return ErrStarted
	}
	return nil
}

func learnerCutoff(c Command) time.Duration {
	if c.Lesson.Plan == commercial.RegularContract && c.Contract != nil {
		return c.Contract.Policy.LearnerChangeCutoff
	}
	if c.Lesson.Plan == commercial.PackagePlan && c.Package != nil {
		return time.Duration(c.Package.Policy.LearnerChangeCutoffHours) * time.Hour
	}
	return 24 * time.Hour
}

func replacementInterval(c Command) (scheduling.Interval, error) {
	policy := scheduling.DefaultIntervalPolicy()
	if c.Lesson.Plan == commercial.RegularContract && c.Contract != nil {
		policy = scheduling.IntervalPolicyFromBusinessPolicy(c.Contract.Policy)
	}
	interval, err := scheduling.NewInterval(c.ReplacementStart.UTC(), c.ReplacementStart.UTC().Add(policy.Duration))
	if err != nil {
		return scheduling.Interval{}, err
	}
	if !scheduling.IsOnGrid(interval.Start, policy.Grid) || !contains(c.Availability, interval) {
		return scheduling.Interval{}, regularcontract.ErrUnavailable
	}
	if err := scheduling.ValidateNoParticipantConflict(interval, c.Lesson.TeacherID, c.Lesson.LearnerID, c.participantLessons(), policy); err != nil {
		return scheduling.Interval{}, err
	}
	return interval, nil
}

func validateReplacementTiming(c Command, start time.Time) error {
	cutoff := learnerCutoff(c)
	if c.Actor.Role == history.LearnerActor && start.Before(c.Now.Add(cutoff)) {
		return commercial.ErrLearnerChangeTooLate
	}
	if c.Lesson.Plan == commercial.RegularContract {
		return nil
	}
	horizon := 14 * 24 * time.Hour
	if c.Lesson.Plan == commercial.PackagePlan && c.Package != nil && c.Package.Policy.BookingHorizonDays > 0 {
		horizon = time.Duration(c.Package.Policy.BookingHorizonDays) * 24 * time.Hour
	}
	if start.After(c.Now.Add(horizon)) {
		return scheduling.ErrHorizon
	}
	return nil
}

func (c Command) participantLessons() []scheduling.Lesson {
	result := make([]scheduling.Lesson, 0, len(c.ParticipantLessons))
	for _, value := range c.ParticipantLessons {
		result = append(result, scheduling.Lesson{TeacherID: value.TeacherID, LearnerID: value.LearnerID, Interval: value.Interval, Status: value.Status})
	}
	return result
}

func contains(values []scheduling.Interval, candidate scheduling.Interval) bool {
	for _, value := range values {
		if value.Contains(candidate) {
			return true
		}
	}
	return false
}
