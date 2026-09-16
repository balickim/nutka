// This file reconciles contract month projections and builds forecast responses.
// Reconciliation uses ledger.ReconcileContractMonth inside one PocketBase transaction.
package ledgerapi

import (
	"context"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func (s *PocketBaseService) ContractMonths(_ context.Context, actor ledger.Actor, contractID string, page, perPage int) (ContractMonthPage, error) {
	if page < 1 || perPage < 1 || perPage > 100 {
		return ContractMonthPage{}, errPagination
	}
	if _, _, err := contractForActor(s.app, contractID, actor); err != nil {
		return ContractMonthPage{}, err
	}
	if err := transaction(context.Background(), s.app, func(tx core.App) error { return s.reconcileContractMonths(tx, contractID) }); err != nil {
		return ContractMonthPage{}, wrapRecordError("reconcile contract month", err)
	}
	rows, err := s.app.FindAllRecords(schedulingstore.ContractMonthsCollectionName, dbx.HashExp{schedulingstore.ContractField: contractID})
	if err != nil {
		return ContractMonthPage{}, err
	}
	return monthPage(s.app, rows, page, perPage, s.instant()), nil
}

// ReconcileContractMonthsInTransaction updates every materialized contract month inside the caller's transaction.
func ReconcileContractMonthsInTransaction(app core.App, contractID string, now time.Time) error {
	if app == nil || !app.IsTransactional() || now.IsZero() {
		return ledger.ErrInvalidDate
	}
	return (&PocketBaseService{app: app, now: func() time.Time { return now.UTC() }}).reconcileContractMonths(app, contractID)
}

func (s *PocketBaseService) reconcileContractMonths(app core.App, contractID string) error {
	contract, err := findRecord(app, schedulingstore.RegularContractsCollectionName, contractID)
	if err != nil {
		return err
	}
	assignment, err := relatedAssignment(app, contract)
	if err != nil {
		return err
	}
	months, err := occurrenceMonths(app, contractID)
	if err != nil {
		return err
	}
	for _, month := range months {
		if err := s.reconcileContractMonth(app, contract, assignment, month); err != nil {
			return err
		}
	}
	return nil
}

func (s *PocketBaseService) reconcileContractMonth(app core.App, contract, assignment *core.Record, month string) error {
	contractID := contract.Id
	location := teacherLocation(app, assignment)
	now := s.instant()
	snapshots, err := monthSnapshots(app, contractID, contract.GetString(schedulingstore.AssignmentField), month)
	if err != nil {
		return err
	}
	existing, monthRecord, err := existingMonth(app, contractID, contract.GetString(schedulingstore.AssignmentField), month)
	if err != nil {
		return err
	}
	charge, err := chargeForMonth(app, contractID, monthRecord)
	if err != nil {
		return err
	}
	decision, err := ledger.ReconcileContractMonth(ledger.MonthReconciliationInput{Existing: existing, ContractID: contractID, AssignmentID: contract.GetString(schedulingstore.AssignmentField), Month: month, ActivationDate: contract.GetString(schedulingstore.StartOnField), Now: now, Location: location, DueDay: businesspolicy.Current().MonthlyPaymentDueDay, Currency: contract.GetString(schedulingstore.CurrencyField), Occurrences: snapshots, Charge: charge})
	if err != nil {
		return err
	}
	return applyMonthDecision(app, contract, contractID, month, monthRecord, charge, decision, now)
}

func applyMonthDecision(app core.App, contract *core.Record, contractID, month string, monthRecord *core.Record, charge *ledger.Charge, decision ledger.MonthReconciliationDecision, now time.Time) error {
	if monthRecord == nil {
		var err error
		monthRecord, err = newMonthRecord(app, contractID, month)
		if err != nil {
			return err
		}
	}
	monthRecord.Set(schedulingstore.ForecastAmountMinorField, decision.Forecast.ForecastAmountMinor)
	monthRecord.Set(schedulingstore.GeneratedAtField, now)
	if decision.Charge != nil {
		if charge == nil {
			if err := saveChargeAndEvent(app, contract, decision.Charge, now); err != nil {
				return err
			}
			if err := applyAvailableCredits(app, decision.Charge, now); err != nil {
				return err
			}
		} else if err := reconcileExistingCharge(app, charge, decision.Forecast.ForecastAmountMinor, now); err != nil {
			return err
		}
		monthRecord.Set(schedulingstore.ChargeField, decision.Charge.ID)
	}
	return app.Save(monthRecord)
}

func occurrenceMonths(app core.App, contractID string) ([]string, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.ContractField: contractID})
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		date := row.GetString(schedulingstore.OriginalLocalDateField)
		if len(date) >= 7 {
			seen[date[:7]] = true
		}
	}
	result := make([]string, 0, len(seen))
	for month := range seen {
		result = append(result, month)
	}
	sort.Strings(result)
	return result, nil
}

func monthSnapshots(app core.App, contractID, assignmentID, month string) ([]ledger.OccurrenceSnapshot, error) {
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.ContractField: contractID})
	if err != nil {
		return nil, err
	}
	result := make([]ledger.OccurrenceSnapshot, 0, len(lessons))
	for _, lesson := range lessons {
		date := lesson.GetString(schedulingstore.OriginalLocalDateField)
		if len(date) < 7 || date[:7] != month {
			continue
		}
		disposition := ledger.OccurrenceDisposition(lesson.GetString(schedulingstore.BillingOutcomeField))
		if disposition == "" {
			disposition = ledger.BillableOccurrence
		}
		result = append(result, ledger.OccurrenceSnapshot{ID: lesson.Id, ContractID: contractID, AssignmentID: assignmentID, Month: month, UnitPriceMinor: int64(lesson.GetInt(schedulingstore.UnitPriceMinorField)), Currency: lesson.GetString(schedulingstore.CurrencyField), Disposition: disposition, ScheduleState: lesson.GetString(schedulingstore.ScheduleStateField)})
	}
	return result, nil
}

func existingMonth(app core.App, contractID, assignmentID, month string) (*ledger.ContractMonth, *core.Record, error) {
	rows, err := app.FindAllRecords(schedulingstore.ContractMonthsCollectionName, dbx.HashExp{schedulingstore.ContractField: contractID, schedulingstore.MonthField: month})
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, nil
	}
	row := rows[0]
	return &ledger.ContractMonth{ID: row.Id, ContractID: contractID, AssignmentID: assignmentID, Month: month, ForecastAmountMinor: int64(row.GetInt(schedulingstore.ForecastAmountMinorField)), ChargeID: row.GetString(schedulingstore.ChargeField), GeneratedAt: recordTime(row, schedulingstore.GeneratedAtField)}, row, nil
}

func newMonthRecord(app core.App, contractID, month string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.ContractMonthsCollectionName)
	if err != nil {
		return nil, err
	}
	row := core.NewRecord(collection)
	row.Set(schedulingstore.ContractField, contractID)
	row.Set(schedulingstore.MonthField, month)
	return row, nil
}

func chargeForMonth(app core.App, contractID string, month *core.Record) (*ledger.Charge, error) {
	if month == nil || month.GetString(schedulingstore.ChargeField) == "" {
		return nil, nil
	}
	row, err := findRecord(app, schedulingstore.ChargesCollectionName, month.GetString(schedulingstore.ChargeField))
	if err != nil {
		return nil, err
	}
	if row.GetString(schedulingstore.SourceIDField) != contractID {
		return nil, errOwnership
	}
	return &ledger.Charge{ID: row.Id, AssignmentID: row.GetString(schedulingstore.AssignmentField), OriginalAmountMinor: int64(row.GetInt(schedulingstore.OriginalAmountMinorField)), CurrentAmountMinor: int64(row.GetInt(schedulingstore.CurrentAmountMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), SettlementState: ledger.SettlementState(row.GetString(schedulingstore.SettlementStateField)), DueOn: row.GetString(schedulingstore.DueOnField)}, nil
}

func saveChargeAndEvent(app core.App, contract *core.Record, charge *ledger.Charge, now time.Time) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.ChargesCollectionName)
	if err != nil {
		return err
	}
	row := core.NewRecord(collection)
	row.Set(schedulingstore.AssignmentField, contract.GetString(schedulingstore.AssignmentField))
	row.Set(schedulingstore.SourceTypeField, charge.SourceType)
	row.Set(schedulingstore.SourceIDField, charge.SourceID)
	row.Set(schedulingstore.PeriodField, charge.Period)
	row.Set(schedulingstore.OriginalAmountMinorField, charge.OriginalAmountMinor)
	row.Set(schedulingstore.CurrentAmountMinorField, charge.CurrentAmountMinor)
	row.Set(schedulingstore.CurrencyField, charge.Currency)
	row.Set(schedulingstore.SettlementStateField, string(charge.SettlementState))
	row.Set(schedulingstore.DueOnField, charge.DueOn)
	row.Set(schedulingstore.CreatedAtField, now)
	if err := app.Save(row); err != nil {
		return err
	}
	charge.ID = row.Id
	entry := ledger.FinancialEntry{AssignmentID: charge.AssignmentID, ChargeID: charge.ID, EntryType: ledger.EntryChargeCreated, AmountMinor: charge.OriginalAmountMinor, Currency: charge.Currency, EffectiveOn: now.UTC().Format("2006-01-02"), Actor: ledger.Actor{Role: ledger.SystemActor}, EventAt: now.UTC()}
	if err := saveEntry(app, entry, charge.ID); err != nil {
		return err
	}
	event, err := history.NewChargeCreated(history.EventInput{AggregateType: "charge", AggregateID: row.Id, AssignmentID: contract.GetString(schedulingstore.AssignmentField), Actor: history.Actor{Role: history.SystemActor}, EventAt: now, NewState: map[string]any{"amount_minor": charge.OriginalAmountMinor, "period": charge.Period}})
	if err != nil {
		return err
	}
	return (history.Storage{}).Append(app, event)
}
