// This file reconciles persisted contract charges with forecast reductions and applies credits oldest first.
package ledgerapi

import (
	"fmt"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func reconcileExistingCharge(app core.App, charge *ledger.Charge, forecast int64, now time.Time) error {
	adjusted, err := chargeAdjustmentTotal(app, charge.ID)
	if err != nil {
		return err
	}
	target := forecast - charge.OriginalAmountMinor
	difference := target - adjusted
	if difference < 0 {
		return reducePersistedCharge(app, charge, -difference, now)
	}
	if difference > 0 {
		return restorePersistedCharge(app, charge, difference, now)
	}
	return nil
}

func chargeAdjustmentTotal(app core.App, chargeID string) (int64, error) {
	rows, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, row := range rows {
		if row.GetString(schedulingstore.ChargeField) == chargeID && row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryAdjustment) {
			total += int64(row.GetInt(schedulingstore.AmountMinorField))
		}
	}
	return total, nil
}

func reducePersistedCharge(app core.App, charge *ledger.Charge, amount int64, now time.Time) error {
	decision, err := ledger.ReduceCharge(*charge, amount, "contract occurrence became non-billable", ledger.Actor{Role: ledger.SystemActor}, now)
	if err != nil {
		return err
	}
	if err := saveChargeProjection(app, &decision.Charge); err != nil {
		return err
	}
	for _, entry := range decision.Entries {
		kind := history.ChargeAdjusted
		if entry.EntryType == ledger.EntryCreditCreated {
			entry.RelatedEntryID = ""
			kind = history.CreditCreated
		}
		row, saveErr := saveEntryRecord(app, entry, charge.ID)
		if saveErr != nil {
			return saveErr
		}
		if err := appendReconciliationEvent(app, row, charge, kind, entry.AmountMinor, now); err != nil {
			return err
		}
	}
	*charge = decision.Charge
	return nil
}

func restorePersistedCharge(app core.App, charge *ledger.Charge, amount int64, now time.Time) error {
	if charge.SettlementState == ledger.Paid {
		if err := consumeSourceCredits(app, charge, amount, now); err != nil {
			return err
		}
	} else {
		charge.CurrentAmountMinor += amount
		if charge.CurrentAmountMinor > charge.OriginalAmountMinor {
			charge.CurrentAmountMinor = charge.OriginalAmountMinor
		}
		if err := saveChargeProjection(app, charge); err != nil {
			return err
		}
	}
	entry := ledger.FinancialEntry{AssignmentID: charge.AssignmentID, ChargeID: charge.ID, EntryType: ledger.EntryAdjustment, AmountMinor: amount, Currency: charge.Currency, EffectiveOn: now.UTC().Format("2006-01-02"), Reason: "contract occurrence became billable", Actor: ledger.Actor{Role: ledger.SystemActor}, EventAt: now.UTC()}
	row, err := saveEntryRecord(app, entry, charge.ID)
	if err != nil {
		return err
	}
	return appendReconciliationEvent(app, row, charge, history.ChargeAdjusted, amount, now)
}

func applyAvailableCredits(app core.App, charge *ledger.Charge, now time.Time) error {
	credits, err := availableCredits(app, charge.AssignmentID, charge.Currency, "")
	if err != nil || len(credits) == 0 {
		return err
	}
	decision, err := ledger.ApplyCredits(*charge, credits, now)
	if err != nil {
		return err
	}
	if err := saveChargeProjection(app, &decision.Charge); err != nil {
		return err
	}
	for _, entry := range decision.Entries {
		row, saveErr := saveEntryRecord(app, entry, charge.ID)
		if saveErr != nil {
			return saveErr
		}
		if err := appendReconciliationEvent(app, row, charge, history.CreditApplied, entry.AmountMinor, now); err != nil {
			return err
		}
	}
	*charge = decision.Charge
	return nil
}

func consumeSourceCredits(app core.App, charge *ledger.Charge, amount int64, now time.Time) error {
	credits, err := availableCredits(app, charge.AssignmentID, charge.Currency, charge.ID)
	if err != nil {
		return err
	}
	remaining := amount
	for _, credit := range credits {
		if remaining == 0 {
			break
		}
		used := credit.RemainingMinor
		if used > remaining {
			used = remaining
		}
		entry := ledger.FinancialEntry{AssignmentID: charge.AssignmentID, ChargeID: charge.ID, EntryType: ledger.EntryCreditApplied, AmountMinor: used, Currency: charge.Currency, EffectiveOn: now.UTC().Format("2006-01-02"), RelatedEntryID: credit.ID, Actor: ledger.Actor{Role: ledger.SystemActor}, EventAt: now.UTC()}
		row, saveErr := saveEntryRecord(app, entry, charge.ID)
		if saveErr != nil {
			return saveErr
		}
		if err := appendReconciliationEvent(app, row, charge, history.CreditApplied, used, now); err != nil {
			return err
		}
		remaining -= used
	}
	if remaining != 0 {
		return ledger.ErrInsufficientCredit
	}
	return nil
}

func availableCredits(app core.App, assignmentID, currency, sourceChargeID string) ([]ledger.Credit, error) {
	rows, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return nil, err
	}
	credits := createdCredits(rows, assignmentID, currency, sourceChargeID)
	applyCreditConsumption(rows, credits)
	result := availableCreditValues(credits)
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func createdCredits(rows []*core.Record, assignmentID, currency, sourceChargeID string) map[string]*ledger.Credit {
	credits := make(map[string]*ledger.Credit)
	for _, row := range rows {
		if row.GetString(schedulingstore.AssignmentField) != assignmentID || currency != "" && row.GetString(schedulingstore.CurrencyField) != currency {
			continue
		}
		if row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryCreditCreated) && (sourceChargeID == "" || row.GetString(schedulingstore.ChargeField) == sourceChargeID) {
			amount := int64(row.GetInt(schedulingstore.AmountMinorField))
			credits[row.Id] = &ledger.Credit{ID: row.Id, AssignmentID: assignmentID, AmountMinor: amount, RemainingMinor: amount, Currency: row.GetString(schedulingstore.CurrencyField), SourceChargeID: row.GetString(schedulingstore.ChargeField), CreatedAt: recordTime(row, schedulingstore.EventAtField)}
		}
	}
	return credits
}

func applyCreditConsumption(rows []*core.Record, credits map[string]*ledger.Credit) {
	for _, row := range rows {
		credit := credits[row.GetString(schedulingstore.RelatedEntryField)]
		if credit != nil && (row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryCreditApplied) || row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryRefund)) {
			credit.RemainingMinor -= int64(row.GetInt(schedulingstore.AmountMinorField))
		}
	}
}

func availableCreditValues(credits map[string]*ledger.Credit) []ledger.Credit {
	result := make([]ledger.Credit, 0, len(credits))
	for _, credit := range credits {
		if credit.Available() {
			result = append(result, *credit)
		}
	}
	return result
}

func saveChargeProjection(app core.App, charge *ledger.Charge) error {
	row, err := app.FindRecordById(schedulingstore.ChargesCollectionName, charge.ID)
	if err != nil {
		return err
	}
	row.Set(schedulingstore.CurrentAmountMinorField, charge.CurrentAmountMinor)
	return app.Save(row)
}

func appendReconciliationEvent(app core.App, entry *core.Record, charge *ledger.Charge, kind history.EventType, amount int64, now time.Time) error {
	event, err := history.NewEvent(kind, history.EventInput{AggregateType: "financial_entry", AggregateID: entry.Id, AssignmentID: charge.AssignmentID, Actor: history.Actor{Role: history.SystemActor}, EventAt: now.UTC(), RelatedIDs: map[string]string{"charge": charge.ID}, NewState: map[string]any{"amount_minor": amount}, Reason: entry.GetString(schedulingstore.ReasonField)})
	if err != nil {
		return fmt.Errorf("construct reconciliation event: %w", err)
	}
	return (history.Storage{}).Append(app, event)
}
