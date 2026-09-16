// This file creates explicitly renewed contracts using current policy and revalidated availability.
package regularcontract

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

// Renew creates a distinct contract with current policy and fresh event-derived allowances.
func (c RegularContract) Renew(id string, startOn, now time.Time, policy Policy, availability []scheduling.Interval, lessons []scheduling.Lesson, actors ...Actor) (RegularContract, error) {
	location, locationErr := c.location()
	if locationErr != nil {
		return RegularContract{}, locationErr
	}
	if c.Status != Ended && !localDate(c.EndOn, location).Before(localDate(now, location)) {
		return RegularContract{}, ErrInvalidContract
	}
	if startOn.IsZero() {
		startOn = c.EndOn.AddDate(0, 0, 1)
	}
	if len(actors) != 1 || actors[0].Role != Teacher || actors[0].ID != c.TeacherID {
		return RegularContract{}, ErrUnauthorized
	}
	request := ActivationRequest{ID: id, AssignmentID: c.AssignmentID, TeacherID: c.TeacherID, LearnerID: c.LearnerID, TeacherTimezone: c.TeacherTimezone,
		Actor: actors[0], StartOn: startOn, Weekday: c.Weekday, StartMinute: c.StartMinute, Now: now, Policy: policy, Availability: availability, ExistingLessons: lessons}
	renewed, err := Activate(request)
	if err == nil && len(renewed.Events) > 0 {
		renewed.Events[0].Type = EventRenewed
	}
	return renewed, err
}
