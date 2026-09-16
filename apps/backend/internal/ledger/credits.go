// This file applies charge reductions, credits, refunds, and financial corrections.
// Each operation returns append-only entries plus independent updated projections.
package ledger

import (
	"fmt"
	"sort"
	"time"
)

type ChargeAdjustmentDecision struct {
	Charge  Charge
	Entry   FinancialEntry
	Entries []FinancialEntry
	Credit  *Credit
	Refund  *Refund
}

// ReduceCharge removes value before payment or creates a credit after payment.
func ReduceCharge(charge Charge, amount int64, reason string, actor Actor, at time.Time) (ChargeAdjustmentDecision, error) {
	if !charge.Valid() {
		return ChargeAdjustmentDecision{}, ErrInvalidState
	}
	if err := validateReduction(amount, reason, actor, at); err != nil {
		return ChargeAdjustmentDecision{}, err
	}
	if amount > charge.CurrentAmountMinor && charge.SettlementState != Paid {
		amount = charge.CurrentAmountMinor
	}
	entry := FinancialEntry{AssignmentID: charge.AssignmentID, ChargeID: charge.ID, EntryType: EntryAdjustment, AmountMinor: -amount, Currency: charge.Currency, EffectiveOn: at.UTC().Format("2006-01-02"), Reason: reason, Actor: actor, EventAt: at.UTC()}
	decision := ChargeAdjustmentDecision{Charge: charge, Entry: entry, Entries: []FinancialEntry{entry}}
	if charge.SettlementState == Paid {
		if amount > charge.CurrentAmountMinor {
			return ChargeAdjustmentDecision{}, ErrAdjustmentExceedsCharge
		}
		credit := &Credit{ID: fmt.Sprintf("%s:credit:%d", charge.ID, at.UTC().UnixNano()), AssignmentID: charge.AssignmentID, AmountMinor: amount, RemainingMinor: amount, Currency: charge.Currency, SourceChargeID: charge.ID, CreatedAt: at.UTC()}
		decision.Credit = credit
		decision.Entries = append(decision.Entries, FinancialEntry{AssignmentID: charge.AssignmentID, ChargeID: charge.ID, EntryType: EntryCreditCreated, AmountMinor: amount, Currency: charge.Currency, EffectiveOn: at.UTC().Format("2006-01-02"), RelatedEntryID: credit.ID, Reason: reason, Actor: actor, EventAt: at.UTC()})
		return decision, nil
	}
	decision.Charge.CurrentAmountMinor -= amount
	if decision.Charge.CurrentAmountMinor < 0 {
		decision.Charge.CurrentAmountMinor = 0
	}
	return decision, nil
}

func validateReduction(amount int64, reason string, actor Actor, at time.Time) error {
	if amount <= 0 || at.IsZero() {
		return ErrNegativeAmount
	}
	if actor.Role != TeacherActor && actor.Role != SystemActor {
		return ErrInvalidTransition
	}
	if actor.Role == TeacherActor && actor.ID == "" {
		return ErrInvalidTransition
	}
	return ValidateReason(reason)
}

type CreditApplicationDecision struct {
	Charge       Charge
	Credits      []Credit
	Entries      []FinancialEntry
	AppliedMinor int64
}

// ApplyCredits consumes same-assignment credits oldest first and never mutates its inputs.
func ApplyCredits(charge Charge, credits []Credit, at time.Time) (CreditApplicationDecision, error) {
	if !canApplyCredits(charge) {
		return CreditApplicationDecision{}, ErrInvalidState
	}
	if at.IsZero() {
		return CreditApplicationDecision{}, ErrInvalidDate
	}
	ordered := append([]Credit(nil), credits...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
	})
	entries := make([]FinancialEntry, 0)
	remaining := charge.CurrentAmountMinor
	initial := remaining
	for index := range ordered {
		credit := &ordered[index]
		if remaining == 0 {
			break
		}
		if credit.AssignmentID != charge.AssignmentID || credit.Currency != charge.Currency || !credit.Available() {
			continue
		}
		applied := credit.RemainingMinor
		if applied > remaining {
			applied = remaining
		}
		credit.RemainingMinor -= applied
		remaining -= applied
		entries = append(entries, FinancialEntry{AssignmentID: charge.AssignmentID, ChargeID: charge.ID, EntryType: EntryCreditApplied, AmountMinor: applied, Currency: charge.Currency, EffectiveOn: at.UTC().Format("2006-01-02"), RelatedEntryID: credit.ID, Actor: Actor{Role: SystemActor}, EventAt: at.UTC()})
	}
	charge.CurrentAmountMinor = remaining
	return CreditApplicationDecision{Charge: charge, Credits: ordered, Entries: entries, AppliedMinor: initial - remaining}, nil
}

func canApplyCredits(charge Charge) bool {
	return charge.Valid() && charge.SettlementState != Paid && charge.SettlementState != NotApplicable
}

// RefundCredit consumes a credit and records a teacher-selected refund alternative.
func RefundCredit(credit Credit, chargeID, reason string, actor Actor, at time.Time) (Credit, Refund, FinancialEntry, error) {
	if err := ValidateReason(reason); err != nil {
		return Credit{}, Refund{}, FinancialEntry{}, err
	}
	if !validRefundCredit(credit, chargeID, actor, at) {
		return Credit{}, Refund{}, FinancialEntry{}, ErrInvalidTransition
	}
	refunded := credit.RemainingMinor
	credit.RemainingMinor = 0
	credit.RefundedAt = at.UTC()
	refund := Refund{AssignmentID: credit.AssignmentID, CreditID: credit.ID, ChargeID: chargeID, AmountMinor: refunded, Currency: credit.Currency, Reason: reason, Actor: actor, RecordedAt: at.UTC()}
	entry := FinancialEntry{AssignmentID: credit.AssignmentID, ChargeID: chargeID, EntryType: EntryRefund, AmountMinor: refunded, Currency: credit.Currency, EffectiveOn: at.UTC().Format("2006-01-02"), RelatedEntryID: credit.ID, Reason: reason, Actor: actor, EventAt: at.UTC()}
	return credit, refund, entry, nil
}

func validRefundCredit(credit Credit, chargeID string, actor Actor, at time.Time) bool {
	return actor.Role == TeacherActor && actor.ID != "" && !at.IsZero() && chargeID != "" && credit.Available() && credit.AssignmentID != "" && credit.AmountMinor > 0 && validCurrency(credit.Currency)
}

func NewCorrection(correction Correction) (Correction, FinancialEntry, error) {
	if err := ValidateReason(correction.Reason); err != nil {
		return Correction{}, FinancialEntry{}, err
	}
	if correction.CorrectsEntryID == "" || correction.Actor.Role != TeacherActor || correction.AssignmentID == "" || !validCurrency(correction.Currency) {
		return Correction{}, FinancialEntry{}, ErrInvalidTransition
	}
	if correction.Actor.ID == "" || correction.RecordedAt.IsZero() {
		return Correction{}, FinancialEntry{}, ErrInvalidTransition
	}
	correction.RecordedAt = correction.RecordedAt.UTC()
	entry := FinancialEntry{ID: correction.ID, AssignmentID: correction.AssignmentID, EntryType: EntryCorrection, AmountMinor: correction.AmountMinor, Currency: correction.Currency, RelatedEntryID: correction.CorrectsEntryID, Reason: correction.Reason, Actor: correction.Actor, EventAt: correction.RecordedAt}
	return correction, entry, nil
}
