// This file builds bounded role-scoped history pages from assignment-owned events.
// It strips teacher-only notes before learner DTOs leave the backend.
package history

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Viewer identifies the assignment participant requesting a role-scoped history page.
type Viewer struct {
	Role ActorRole
	ID   string
}

// EventView is the API-safe representation of an immutable business event.
type EventView struct {
	ID              string            `json:"id"`
	EventType       EventType         `json:"event_type"`
	AggregateType   string            `json:"aggregate_type"`
	AggregateID     string            `json:"aggregate_id"`
	AssignmentID    string            `json:"assignment"`
	ActorRole       ActorRole         `json:"actor_role"`
	ActorID         string            `json:"actor_id,omitempty"`
	EventAt         string            `json:"event_at"`
	RelatedIDs      map[string]string `json:"related_ids,omitempty"`
	PriorState      map[string]any    `json:"prior_state,omitempty"`
	NewState        map[string]any    `json:"new_state,omitempty"`
	Reason          string            `json:"reason,omitempty"`
	InternalNote    string            `json:"internal_note,omitempty"`
	CorrectsEventID string            `json:"corrects_event,omitempty"`
}

// Page is a stable page of history ordered by event time and record identity.
type Page struct {
	Items      []EventView `json:"items"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalItems int         `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

var (
	ErrHistoryUnauthorized = errors.New("history authorization failed")
	ErrHistoryPagination   = errors.New("history pagination is invalid")
)

// QueryPage authorizes a participant and returns a bounded, deterministic history page.
func QueryPage(app core.App, assignmentID string, viewer Viewer, page, perPage int) (Page, error) {
	if page < 1 || perPage < 1 || perPage > 100 {
		return Page{}, ErrHistoryPagination
	}
	if err := authorizeAssignment(app, assignmentID, viewer); err != nil {
		return Page{}, err
	}
	filter := "assignment = {:assignment}"
	params := dbx.Params{"assignment": assignmentID}
	countFilter := []dbx.Expression{dbx.HashExp{AssignmentField: assignmentID}}
	if viewer.Role == LearnerActor {
		filter += " && event_type != {:hidden_event}"
		params["hidden_event"] = string(AvailabilityChanged)
		countFilter = append(countFilter, dbx.NotIn(EventTypeField, string(AvailabilityChanged)))
	}
	offset := (page - 1) * perPage
	rows, err := app.FindRecordsByFilter(BusinessEventsCollection, filter, "-event_at,-id", perPage, offset, params)
	if err != nil {
		return Page{}, fmt.Errorf("query business history: %w", err)
	}
	total, err := app.CountRecords(BusinessEventsCollection, countFilter...)
	if err != nil {
		return Page{}, fmt.Errorf("count business history: %w", err)
	}
	views := make([]EventView, 0, len(rows))
	for _, record := range rows {
		event, eventErr := EventFromRecord(record)
		if eventErr != nil {
			return Page{}, fmt.Errorf("decode business history: %w", eventErr)
		}
		views = append(views, viewFor(event, viewer.Role))
	}
	result := Page{Items: views, Page: page, PerPage: perPage, TotalItems: int(total)}
	result.TotalPages = int(math.Ceil(float64(result.TotalItems) / float64(perPage)))
	return result, nil
}

func authorizeAssignment(app core.App, assignmentID string, viewer Viewer) error {
	if assignmentID == "" || viewer.ID == "" || (viewer.Role != TeacherActor && viewer.Role != LearnerActor) {
		return ErrHistoryUnauthorized
	}
	assignment, err := app.FindRecordById("teacher_learners", assignmentID)
	if err != nil {
		return ErrHistoryUnauthorized
	}
	field := "learner"
	if viewer.Role == TeacherActor {
		field = "teacher"
	}
	if assignment.GetString(field) != viewer.ID {
		return ErrHistoryUnauthorized
	}
	return nil
}

func viewFor(event Event, role ActorRole) EventView {
	view := EventView{
		ID: event.ID, EventType: event.Type, AggregateType: event.AggregateType,
		AggregateID: event.AggregateID, AssignmentID: event.AssignmentID,
		ActorRole: event.Actor.Role, ActorID: event.Actor.ID,
		EventAt:    event.EventAt.UTC().Format(time.RFC3339Nano),
		RelatedIDs: cloneStrings(event.RelatedIDs), PriorState: cloneMap(event.PriorState),
		NewState: cloneMap(event.NewState), Reason: event.Reason,
		CorrectsEventID: event.CorrectsEventID,
	}
	if role == TeacherActor {
		view.InternalNote = event.InternalNote
	} else {
		view.PriorState = redactProtectedFields(event.PriorState)
		view.NewState = redactProtectedFields(event.NewState)
	}
	return view
}

func redactProtectedFields(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		if strings.EqualFold(key, "internal_note") || strings.EqualFold(key, "teacher_note") {
			continue
		}
		switch nested := value.(type) {
		case map[string]any:
			result[key] = redactProtectedFields(nested)
		case []any:
			items := make([]any, len(nested))
			for index, item := range nested {
				if child, ok := item.(map[string]any); ok {
					items[index] = redactProtectedFields(child)
				} else {
					items[index] = item
				}
			}
			result[key] = items
		default:
			result[key] = value
		}
	}
	return result
}
