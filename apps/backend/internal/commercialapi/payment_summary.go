// Package commercialapi exposes a compact assignment-owned payment summary.
// It derives overdue presentation from teacher-local dates without storing new payment state.
package commercialapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func paymentSummary(app core.App, assignmentID string, now time.Time) (paymentSummaryDTO, error) {
	result := paymentSummaryDTO{Currency: "PLN"}
	location, locationErr := assignmentLocation(app, assignmentID)
	if locationErr != nil {
		return paymentSummaryDTO{}, locationErr
	}
	rows, err := app.FindAllRecords(schedulingstore.ChargesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return paymentSummaryDTO{}, err
	}
	for _, row := range rows {
		addChargeSummary(&result, row, now, location)
	}
	entries, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return paymentSummaryDTO{}, err
	}
	for _, entry := range entries {
		addCreditSummary(&result, entry)
	}
	return result, nil
}

func addChargeSummary(result *paymentSummaryDTO, row *core.Record, now time.Time, location *time.Location) {
	state := row.GetString(schedulingstore.SettlementStateField)
	if state == "pending" || state == "pending_settlement" {
		result.Pending++
	}
	if state == "intentionally_unpaid" {
		result.IntentionallyUnpaid++
	}
	if state != "paid" && row.GetString(schedulingstore.DueOnField) != "" && overdue(row.GetString(schedulingstore.DueOnField), now, location) {
		result.Overdue++
	}
	if result.Currency == "PLN" && row.GetString(schedulingstore.CurrencyField) != "" {
		result.Currency = row.GetString(schedulingstore.CurrencyField)
	}
}

func addCreditSummary(result *paymentSummaryDTO, entry *core.Record) {
	switch entry.GetString(schedulingstore.EntryTypeField) {
	case "credit_created":
		result.CreditMinor += int64(entry.GetInt(schedulingstore.AmountMinorField))
	case "credit_applied", "refund":
		result.CreditMinor -= int64(entry.GetInt(schedulingstore.AmountMinorField))
	}
}

func assignmentLocation(app core.App, assignmentID string) (*time.Location, error) {
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, assignmentID)
	if err != nil {
		return nil, err
	}
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(teacher.GetString("timezone"))
	if err != nil {
		return nil, err
	}
	return location, nil
}

func overdue(due string, now time.Time, location *time.Location) bool {
	date, err := time.ParseInLocation("2006-01-02", due, location)
	return err == nil && !now.In(location).Before(date.AddDate(0, 0, 1))
}
