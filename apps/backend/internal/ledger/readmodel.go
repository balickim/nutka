// This file builds bounded assignment-level financial work summaries.
// It exposes derived payment information without leaking persistence records.
package ledger

import "time"

// FinancialSummary is a bounded assignment-level payment read model.
type FinancialSummary struct {
	PendingSettlement int
	UnpaidAdHoc       int
	UnpaidAdHocIDs    []string
	UnpaidCharges     int
	UnpaidChargeIDs   []string
	OverdueCharges    int
	OverdueChargeIDs  []string
	OpenCreditMinor   int64
	Currency          string
}

// BuildFinancialSummary excludes paid, not-applicable, refunded, and unrelated records.
func BuildFinancialSummary(lessons []AdHocLesson, charges []Charge, credits []Credit, now time.Time, location *time.Location) FinancialSummary {
	summary := FinancialSummary{UnpaidAdHocIDs: []string{}, UnpaidChargeIDs: []string{}, OverdueChargeIDs: []string{}}
	for _, lesson := range lessons {
		addAdHocToSummary(&summary, lesson)
	}
	for _, charge := range charges {
		addChargeToSummary(&summary, charge, now, location)
	}
	for _, credit := range credits {
		addCreditToSummary(&summary, credit)
	}
	return summary
}

func addAdHocToSummary(summary *FinancialSummary, lesson AdHocLesson) {
	if lesson.SettlementState == PendingSettlement {
		summary.PendingSettlement++
	}
	if lesson.SettlementState == IntentionallyUnpaid {
		summary.UnpaidAdHoc++
		summary.UnpaidAdHocIDs = append(summary.UnpaidAdHocIDs, lesson.ID)
	}
}

func addChargeToSummary(summary *FinancialSummary, charge Charge, now time.Time, location *time.Location) {
	if charge.CurrentAmountMinor > 0 && (charge.SettlementState == PendingPayment || charge.SettlementState == IntentionallyUnpaid) {
		summary.UnpaidCharges++
		summary.UnpaidChargeIDs = append(summary.UnpaidChargeIDs, charge.ID)
	}
	if charge.IsOverdue(now, location) {
		summary.OverdueCharges++
		summary.OverdueChargeIDs = append(summary.OverdueChargeIDs, charge.ID)
	}
	if summary.Currency == "" {
		summary.Currency = charge.Currency
	}
}

func addCreditToSummary(summary *FinancialSummary, credit Credit) {
	if !credit.Available() {
		return
	}
	summary.OpenCreditMinor += credit.RemainingMinor
	if summary.Currency == "" {
		summary.Currency = credit.Currency
	}
}
