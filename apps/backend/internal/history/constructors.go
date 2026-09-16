// This file constructs validated immutable events and copies caller-owned state.
// It keeps typed transition helpers on one shared constructor boundary.
package history

// NewEvent validates and copies one ordinary transition.
func NewEvent(eventType EventType, input EventInput) (Event, error) {
	event := Event{
		Type: eventType, AggregateType: input.AggregateType, AggregateID: input.AggregateID,
		AssignmentID: input.AssignmentID, Actor: input.Actor, EventAt: input.EventAt.UTC(),
		RelatedIDs: cloneStrings(input.RelatedIDs), PriorState: cloneMap(input.PriorState),
		NewState: cloneMap(input.NewState), Reason: input.Reason, InternalNote: input.InternalNote,
	}
	if err := event.Validate(); err != nil {
		return Event{}, err
	}
	return event, nil
}

func cloneStrings(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = cloneValue(item)
		}
		return result
	default:
		return value
	}
}

// constructor creates a typed event while keeping all shared validation in one place.
func constructor(eventType EventType, input EventInput) (Event, error) {
	return NewEvent(eventType, input)
}

func NewLessonCreated(input EventInput) (Event, error)   { return constructor(LessonCreated, input) }
func NewLessonConverted(input EventInput) (Event, error) { return constructor(LessonConverted, input) }
func NewLessonRescheduled(input EventInput) (Event, error) {
	return constructor(LessonRescheduled, input)
}
func NewLessonCancelled(input EventInput) (Event, error) { return constructor(LessonCancelled, input) }
func NewLessonOutcomeRecorded(input EventInput) (Event, error) {
	return constructor(LessonOutcomeRecorded, input)
}
func NewSettlementChanged(input EventInput) (Event, error) {
	return constructor(SettlementChanged, input)
}
func NewPackagePurchased(input EventInput) (Event, error) {
	return constructor(PackagePurchased, input)
}
func NewPackageTokenReserved(input EventInput) (Event, error) {
	return constructor(PackageTokenReserved, input)
}
func NewPackageTokenUsed(input EventInput) (Event, error) {
	return constructor(PackageTokenUsed, input)
}
func NewPackageTokenReturned(input EventInput) (Event, error) {
	return constructor(PackageTokenReturned, input)
}
func NewPackageTokenExpired(input EventInput) (Event, error) {
	return constructor(PackageTokenExpired, input)
}
func NewPackageTokenExtended(input EventInput) (Event, error) {
	return constructor(PackageTokenExtended, input)
}
func NewPackageTokenInvalidated(input EventInput) (Event, error) {
	return constructor(PackageTokenInvalidated, input)
}
func NewPackageValidityExtended(input EventInput) (Event, error) {
	return constructor(PackageValidityExtended, input)
}
func NewPackageClosed(input EventInput) (Event, error) { return constructor(PackageClosed, input) }
func NewContractActivated(input EventInput) (Event, error) {
	return constructor(ContractActivated, input)
}
func NewContractScheduleChanged(input EventInput) (Event, error) {
	return constructor(ContractScheduleChanged, input)
}
func NewContractOccurrenceRescheduled(input EventInput) (Event, error) {
	return constructor(ContractOccurrenceRescheduled, input)
}
func NewContractOccurrenceOmitted(input EventInput) (Event, error) {
	return constructor(ContractOccurrenceOmitted, input)
}
func NewContractOccurrenceRestored(input EventInput) (Event, error) {
	return constructor(ContractOccurrenceRestored, input)
}
func NewContractAmended(input EventInput) (Event, error) { return constructor(ContractAmended, input) }
func NewContractNoticeSubmitted(input EventInput) (Event, error) {
	return constructor(ContractNoticeSubmitted, input)
}
func NewContractRenewed(input EventInput) (Event, error) { return constructor(ContractRenewed, input) }
func NewContractEnded(input EventInput) (Event, error)   { return constructor(ContractEnded, input) }
func NewChargeCreated(input EventInput) (Event, error)   { return constructor(ChargeCreated, input) }
func NewChargeAdjusted(input EventInput) (Event, error)  { return constructor(ChargeAdjusted, input) }
func NewPaymentRecorded(input EventInput) (Event, error) { return constructor(PaymentRecorded, input) }
func NewCreditCreated(input EventInput) (Event, error)   { return constructor(CreditCreated, input) }
func NewCreditApplied(input EventInput) (Event, error)   { return constructor(CreditApplied, input) }
func NewRefundRecorded(input EventInput) (Event, error)  { return constructor(RefundRecorded, input) }
func NewAvailabilityConsequence(input EventInput) (Event, error) {
	return constructor(AvailabilityConsequence, input)
}
func NewAvailabilityChanged(input EventInput) (Event, error) {
	return constructor(AvailabilityChanged, input)
}
