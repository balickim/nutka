// This file contains shared contract lookup, event, allowance, and replacement helpers.
package regularcontract

import (
	"strconv"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func (c *RegularContract) nextEventID() string {
	return c.ID + ":event:" + strconv.Itoa(len(c.Events)+1)
}

func (c *RegularContract) addEvent(eventType, occurrenceID, month string, actor Actor, at time.Time, reason string) {
	c.Events = append(c.Events, Event{ID: c.nextEventID(), Type: eventType, ContractID: c.ID, OccurrenceID: occurrenceID, OriginalMonth: month, Actor: actor, At: at.UTC(), Reason: reason})
}

func (c RegularContract) location() (*time.Location, error) {
	return scheduling.LoadTimezone(c.TeacherTimezone)
}

func (c *RegularContract) occurrence(id string) (*Occurrence, error) {
	for index := range c.Occurrences {
		if c.Occurrences[index].ID == id {
			return &c.Occurrences[index], nil
		}
	}
	return nil, ErrInvalidContract
}

func (c RegularContract) uncompensatedEvents() map[string]bool {
	compensated := make(map[string]bool)
	for _, event := range c.Events {
		if event.CorrectsEvent != "" {
			compensated[event.CorrectsEvent] = true
		}
	}
	result := make(map[string]bool)
	for _, event := range c.Events {
		if !compensated[event.ID] {
			result[event.ID] = true
		}
	}
	return result
}

// activeBaseEvents resolves correction chains so projections retain the original event effect.
func (c RegularContract) activeBaseEvents() map[string]bool {
	corrections := make(map[string]Event)
	for _, event := range c.Events {
		if event.Type == "correction" && event.CorrectsEvent != "" {
			corrections[event.CorrectsEvent] = event
		}
	}
	active := make(map[string]bool)
	for _, base := range c.Events {
		if base.Type == "correction" {
			continue
		}
		depth := 0
		seen := map[string]bool{base.ID: true}
		for correction, found := corrections[base.ID]; found; correction, found = corrections[correction.ID] {
			if seen[correction.ID] {
				depth = -1
				break
			}
			seen[correction.ID] = true
			depth++
		}
		if depth >= 0 && depth%2 == 0 {
			active[base.ID] = true
		}
	}
	return active
}

// Allowances derives current balances from uncompensated learner events.
func (c RegularContract) Allowances(originalMonth string) AllowanceBalance {
	active := c.activeBaseEvents()
	reschedules, cancellations := 0, 0
	for _, event := range c.Events {
		if !active[event.ID] || event.Actor.Role != Learner {
			continue
		}
		switch event.Type {
		case EventLearnerRescheduled:
			if event.OriginalMonth == originalMonth {
				reschedules++
			}
		case EventLearnerFreeCancel:
			cancellations++
		}
	}
	policy := c.lifecyclePolicy()
	return AllowanceBalance{MonthlyReschedulesRemaining: maxInt(0, policy.ContractMonthlyReschedules-reschedules), FreeCancellationsRemaining: maxInt(0, policy.ContractFreeCancellations-cancellations)}
}

func (c RegularContract) occurrenceWasLearnerRescheduled(id string) bool {
	active := c.activeBaseEvents()
	for _, event := range c.Events {
		if active[event.ID] && event.Type == EventLearnerRescheduled && event.OccurrenceID == id && event.Actor.Role == Learner {
			return true
		}
	}
	return false
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func authorized(c RegularContract, actor Actor) bool {
	return actor.ID != "" && ((actor.Role == Teacher && actor.ID == c.TeacherID) || (actor.Role == Learner && actor.ID == c.LearnerID))
}

func availableReplacement(request RescheduleRequest, duration time.Duration, teacherID, learnerID string, policy scheduling.IntervalPolicy) (scheduling.Interval, error) {
	if request.ReplacementStart.IsZero() {
		return scheduling.Interval{}, ErrInvalidContract
	}
	if !scheduling.IsOnGrid(request.ReplacementStart, policy.Grid) {
		return scheduling.Interval{}, scheduling.ErrInvalidGrid
	}
	replacement, err := scheduling.NewInterval(request.ReplacementStart.UTC(), request.ReplacementStart.UTC().Add(duration))
	if err != nil {
		return scheduling.Interval{}, err
	}
	if !containsInterval(request.Available, replacement) {
		return scheduling.Interval{}, ErrUnavailable
	}
	lessons := make([]scheduling.Lesson, 0, len(request.ParticipantLessons)+len(request.ParticipantLessonRecords))
	for _, record := range request.ParticipantLessonRecords {
		if record.ID == request.OccurrenceID {
			continue
		}
		lessons = append(lessons, scheduling.Lesson{TeacherID: record.TeacherID, LearnerID: record.LearnerID, Interval: record.Interval, Status: record.Status})
	}
	for _, lesson := range request.ParticipantLessons {
		lessons = append(lessons, lesson)
	}
	if err := scheduling.ValidateNoParticipantConflict(replacement, teacherID, learnerID, lessons, policy); err != nil {
		return scheduling.Interval{}, err
	}
	return replacement, nil
}
