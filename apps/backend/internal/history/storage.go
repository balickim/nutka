// This file persists validated events inside a caller-owned PocketBase transaction.
// It exposes append only and does not provide event mutation or deletion operations.
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// Storage owns the append-only PocketBase boundary for business events.
// Callers must pass their active transaction so event writes share mutation atomicity.
type Storage struct{}

const (
	BusinessEventsCollection = "business_events"
	AssignmentField          = "assignment"
	AggregateTypeField       = "aggregate_type"
	AggregateIDField         = "aggregate_id"
	EventTypeField           = "event_type"
	ActorRoleField           = "actor_role"
	ActorIDField             = "actor_id"
	EventAtField             = "event_at"
	RelatedIDsField          = "related_ids"
	PriorStateField          = "prior_state"
	NewStateField            = "new_state"
	ReasonField              = "reason"
	InternalNoteField        = "internal_note"
	CorrectsEventField       = "corrects_event"
)

var ErrTransactionRequired = errors.New("business event append requires an active transaction")

// Append validates and inserts one event into the caller's transaction.
// Storage intentionally has no update or delete method.
func (Storage) Append(app core.App, event Event) error {
	_, err := (Storage{}).AppendReturningID(app, event)
	return err
}

// AppendReturningID appends one event and returns its generated immutable record ID.
func (Storage) AppendReturningID(app core.App, event Event) (string, error) {
	if app == nil || !app.IsTransactional() {
		return "", ErrTransactionRequired
	}
	if err := event.Validate(); err != nil {
		return "", err
	}
	collection, err := app.FindCollectionByNameOrId(BusinessEventsCollection)
	if err != nil {
		return "", fmt.Errorf("find %s collection: %w", BusinessEventsCollection, err)
	}
	record := core.NewRecord(collection)
	record.Set(AssignmentField, event.AssignmentID)
	record.Set(AggregateTypeField, event.AggregateType)
	record.Set(AggregateIDField, event.AggregateID)
	record.Set(EventTypeField, string(event.Type))
	record.Set(ActorRoleField, string(event.Actor.Role))
	record.Set(ActorIDField, event.Actor.ID)
	record.Set(EventAtField, event.EventAt.UTC().Format(time.RFC3339Nano))
	if err := setOptionalJSON(record, RelatedIDsField, event.RelatedIDs); err != nil {
		return "", err
	}
	if err := setJSON(record, PriorStateField, event.PriorState); err != nil {
		return "", err
	}
	if err := setJSON(record, NewStateField, event.NewState); err != nil {
		return "", err
	}
	record.Set(ReasonField, event.Reason)
	record.Set(InternalNoteField, event.InternalNote)
	record.Set(CorrectsEventField, event.CorrectsEventID)
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("append business event: %w", err)
	}
	return record.Id, nil
}

func setJSON(record *core.Record, field string, value any) error {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode %s: %w", field, err)
	}
	var normalized any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return fmt.Errorf("normalize %s: %w", field, err)
	}
	record.Set(field, normalized)
	return nil
}

func setOptionalJSON(record *core.Record, field string, value any) error {
	if record.Collection().Fields.GetByName(field) == nil {
		return nil
	}
	return setJSON(record, field, value)
}

// EventFromRecord converts a persisted event into the pure domain representation.
func EventFromRecord(record *core.Record) (Event, error) {
	if record == nil {
		return Event{}, ErrInvalidEvent
	}
	eventAt := record.GetDateTime(EventAtField).Time().UTC()
	event := Event{
		ID:              record.Id,
		Type:            EventType(record.GetString(EventTypeField)),
		AggregateType:   record.GetString(AggregateTypeField),
		AggregateID:     record.GetString(AggregateIDField),
		AssignmentID:    record.GetString(AssignmentField),
		Actor:           Actor{Role: ActorRole(record.GetString(ActorRoleField)), ID: record.GetString(ActorIDField)},
		EventAt:         eventAt,
		RelatedIDs:      mapStringString(record.Get(RelatedIDsField)),
		PriorState:      mapAny(record.Get(PriorStateField)),
		NewState:        mapAny(record.Get(NewStateField)),
		Reason:          record.GetString(ReasonField),
		InternalNote:    record.GetString(InternalNoteField),
		CorrectsEventID: record.GetString(CorrectsEventField),
	}
	if err := event.Validate(); err != nil {
		return Event{}, err
	}
	return event, nil
}

func mapAny(value any) map[string]any {
	var encoded []byte
	switch typed := value.(type) {
	case string:
		encoded = []byte(typed)
	case []byte:
		encoded = typed
	case nil:
		return nil
	default:
		encoded, _ = json.Marshal(typed)
	}
	var result map[string]any
	if json.Unmarshal(encoded, &result) != nil {
		return nil
	}
	return result
}

func mapStringString(value any) map[string]string {
	values := mapAny(value)
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		if text, ok := value.(string); ok {
			result[key] = text
		}
	}
	return result
}
