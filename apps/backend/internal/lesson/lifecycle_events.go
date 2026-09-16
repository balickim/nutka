// This file constructs immutable lifecycle events and complete command decisions.
package lesson

import "github.com/balickim/nutka/apps/backend/internal/history"

func decision(c Command, effect string, events ...history.Event) Decision {
	return Decision{Lesson: c.Lesson, Package: c.Package, Contract: c.Contract, Charge: c.Charge, Credits: c.Credits, PlanEffect: effect, Events: events}
}

func lifecycleEvent(kind history.EventType, c Command, prior, next map[string]any) (history.Event, error) {
	return history.NewEvent(kind, history.EventInput{AggregateType: "lesson", AggregateID: c.Lesson.ID, AssignmentID: c.Lesson.AssignmentID, Actor: c.Actor, EventAt: c.Now.UTC(), RelatedIDs: map[string]string{"plan": string(c.Lesson.Plan)}, PriorState: prior, NewState: next})
}

func cutoffClass(timely bool) string {
	if timely {
		return "timely"
	}
	return "late"
}
