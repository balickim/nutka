// This file builds paginated contract month views from reconciled PocketBase projections.
package ledgerapi

import (
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func monthPage(app core.App, rows []*core.Record, page, perPage int, now time.Time) ContractMonthPage {
	sortMonths(rows)
	start := (page - 1) * perPage
	if start > len(rows) {
		start = len(rows)
	}
	end := start + perPage
	if end > len(rows) {
		end = len(rows)
	}
	items := make([]ContractMonthView, 0, end-start)
	for _, row := range rows[start:end] {
		items = append(items, monthView(app, row, now))
	}
	totalPages := (len(rows) + perPage - 1) / perPage
	return ContractMonthPage{Items: items, Page: page, PerPage: perPage, TotalItems: len(rows), TotalPages: totalPages}
}

func sortMonths(rows []*core.Record) {
	sort.SliceStable(rows, func(left, right int) bool {
		return rows[left].GetString(schedulingstore.MonthField) < rows[right].GetString(schedulingstore.MonthField)
	})
}

func monthView(app core.App, row *core.Record, now time.Time) ContractMonthView {
	view := ContractMonthView{ID: row.Id, ContractID: row.GetString(schedulingstore.ContractField), Month: row.GetString(schedulingstore.MonthField), ForecastAmountMinor: int64(row.GetInt(schedulingstore.ForecastAmountMinorField)), GeneratedAt: utc(recordTime(row, schedulingstore.GeneratedAtField)), Forecast: row.GetString(schedulingstore.ChargeField) == ""}
	appendMonthForecast(app, &view)
	chargeID := row.GetString(schedulingstore.ChargeField)
	if chargeID == "" {
		return view
	}
	view.ChargeID = &chargeID
	charge, err := findRecord(app, schedulingstore.ChargesCollectionName, chargeID)
	if err != nil {
		return view
	}
	appendMonthCharge(app, &view, charge, now)
	return view
}

func appendMonthForecast(app core.App, view *ContractMonthView) {
	contract, err := findRecord(app, schedulingstore.RegularContractsCollectionName, view.ContractID)
	if err != nil {
		return
	}
	view.AssignmentID = contract.GetString(schedulingstore.AssignmentField)
	view.Currency = contract.GetString(schedulingstore.CurrencyField)
	snapshots, err := monthSnapshots(app, view.ContractID, view.AssignmentID, view.Month)
	if err != nil {
		return
	}
	forecast, err := ledger.ForecastContractMonth(view.ContractID, view.AssignmentID, view.Month, snapshots)
	if err != nil {
		return
	}
	view.BillableCount = forecast.BillableCount
	view.OccurrenceIDs = forecast.OccurrenceIDs
}

func appendMonthCharge(app core.App, view *ContractMonthView, charge *core.Record, now time.Time) {
	view.CurrentAmountMinor = int64(charge.GetInt(schedulingstore.CurrentAmountMinorField))
	view.Currency = charge.GetString(schedulingstore.CurrencyField)
	view.SettlementState = charge.GetString(schedulingstore.SettlementStateField)
	view.DueOn = charge.GetString(schedulingstore.DueOnField)
	location := time.UTC
	if assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, view.AssignmentID); err == nil {
		location = teacherLocation(app, assignment)
	}
	domainCharge := ledger.Charge{ID: charge.Id, AssignmentID: view.AssignmentID, OriginalAmountMinor: int64(charge.GetInt(schedulingstore.OriginalAmountMinorField)), CurrentAmountMinor: view.CurrentAmountMinor, Currency: view.Currency, SettlementState: ledger.SettlementState(view.SettlementState), DueOn: view.DueOn}
	view.DerivedState = string(domainCharge.DerivedState(now, location))
}
