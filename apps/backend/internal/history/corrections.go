// This file creates compensating events and resolves the active correction chain.
// It derives current effects without mutating the append-only event history.
package history

import (
	"sort"
	"strings"
)

// NewCorrection creates a compensating event and requires a non-empty reason.
func NewCorrection(input EventInput, correctedEventID string) (Event, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" {
		return Event{}, ErrCorrectionReason
	}
	if strings.TrimSpace(correctedEventID) == "" {
		return Event{}, ErrCorrectionTarget
	}
	event, err := NewEvent(AdministrativeCorrection, input)
	if err != nil {
		return Event{}, err
	}
	event.CorrectsEventID = correctedEventID
	return event, nil
}

// Uncompensated removes transitions whose active correction chain compensates them.
func Uncompensated(events []Event) []Event {
	byID := make(map[string]Event, len(events))
	corrections := make(map[string][]string, len(events))
	for _, event := range events {
		if event.ID != "" {
			byID[event.ID] = event
		}
		if event.ID != "" && event.Type == AdministrativeCorrection && event.CorrectsEventID != "" {
			corrections[event.CorrectsEventID] = append(corrections[event.CorrectsEventID], event.ID)
		}
	}
	for target := range corrections {
		sort.Strings(corrections[target])
	}
	state := make(map[string]uint8, len(byID))
	active := make(map[string]bool, len(byID))
	var isActive func(string) bool
	isActive = func(id string) bool {
		if state[id] == 2 {
			return active[id]
		}
		if state[id] == 1 {
			return true
		}
		state[id] = 1
		active[id] = true
		for _, correctionID := range corrections[id] {
			if isActive(correctionID) {
				active[id] = false
				break
			}
		}
		state[id] = 2
		return active[id]
	}
	result := make([]Event, 0, len(events))
	for _, event := range events {
		if event.ID == "" || isActive(event.ID) {
			result = append(result, event)
		}
	}
	return result
}

// CurrentEffects returns transitions that currently contribute to a projection.
func CurrentEffects(events []Event) []Event { return Uncompensated(events) }
