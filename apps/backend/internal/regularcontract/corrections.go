// This file applies compensating contract events, including correction-of-correction chains.
package regularcontract

func (c *RegularContract) reverseEventEffect(event Event) {
	base, depth, found := c.correctionBase(event)
	if !found {
		return
	}
	// The new correction adds one link, so odd chain depth applies the inverse effect.
	c.applyBaseEffect(base, (depth+1)%2 == 1)
}

func (c *RegularContract) applyBaseEffect(event Event, inverse bool) {
	occurrence, err := c.occurrence(event.OccurrenceID)
	if err != nil {
		return
	}
	switch event.Type {
	case EventLearnerFreeCancel, EventTeacherCancel, EventLearnerLateCancel, EventLearnerBillableCancel:
		applyCancellationEffect(occurrence, event.Type, inverse)
	case EventLearnerRescheduled, EventTeacherRescheduled:
		applyRescheduleEffect(occurrence, event, inverse)
	}
}

func applyCancellationEffect(occurrence *Occurrence, eventType string, inverse bool) {
	if inverse {
		occurrence.ScheduleState = Scheduled
		occurrence.BillingOutcome = BillableOrdinary
		return
	}
	occurrence.ScheduleState = Cancelled
	outcomes := map[string]BillingOutcome{EventLearnerFreeCancel: FreeLearnerCancel, EventTeacherCancel: TeacherCancel, EventLearnerLateCancel: BillableLateCancel, EventLearnerBillableCancel: BillableExhausted}
	occurrence.BillingOutcome = outcomes[eventType]
}

func applyRescheduleEffect(occurrence *Occurrence, event Event, inverse bool) {
	if inverse {
		if event.PriorInterval.Valid() {
			occurrence.Interval = event.PriorInterval
		}
		occurrence.IndividuallyRescheduled = false
		return
	}
	if event.NewInterval.Valid() {
		occurrence.Interval = event.NewInterval
	}
	occurrence.IndividuallyRescheduled = true
}

func (c RegularContract) correctionBase(event Event) (Event, int, bool) {
	depth := 0
	seen := map[string]bool{}
	for event.Type == "correction" {
		if event.ID == "" || seen[event.ID] {
			return Event{}, 0, false
		}
		seen[event.ID] = true
		target, found := c.event(event.CorrectsEvent)
		if !found {
			return Event{}, 0, false
		}
		event = target
		depth++
	}
	return event, depth, true
}

func (c RegularContract) event(id string) (Event, bool) {
	for _, event := range c.Events {
		if event.ID == id {
			return event, true
		}
	}
	return Event{}, false
}
