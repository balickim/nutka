// This file implements assignment-scoped ledger read models for PocketBase.
// It returns bounded DTOs and never exposes raw records or protected financial fields.
package ledgerapi

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func (s *PocketBaseService) FinancialWork(_ context.Context, actor ledger.Actor) (FinancialWork, error) {
	assignments, err := assignmentsForActor(s.app, actor)
	if err != nil {
		return FinancialWork{}, err
	}
	owned := assignmentSet(assignments)
	charges, err := chargesForAssignments(s.app, owned)
	if err != nil {
		return FinancialWork{}, err
	}
	entries, err := entriesForAssignments(s.app, owned)
	if err != nil {
		return FinancialWork{}, err
	}
	return FinancialWork{Charges: chargeViews(s.app, charges, s.instant(), assignments), Entries: entryViews(entries), Credits: creditViews(entries)}, nil
}

func (s *PocketBaseService) FinancialSummary(_ context.Context, actor ledger.Actor) (FinancialSummary, error) {
	assignments, err := assignmentsForActor(s.app, actor)
	if err != nil {
		return FinancialSummary{}, err
	}
	owned := assignmentSet(assignments)
	charges, err := chargesForAssignments(s.app, owned)
	if err != nil {
		return FinancialSummary{}, err
	}
	entries, err := entriesForAssignments(s.app, owned)
	if err != nil {
		return FinancialSummary{}, err
	}
	result := FinancialSummary{Charges: chargeViews(s.app, charges, s.instant(), assignments), Credits: creditViews(entries)}
	for _, charge := range charges {
		addFinancialSummaryCharge(&result, s.app, charge, assignments, s.instant())
	}
	for _, assignment := range assignments {
		credits, creditErr := availableCredits(s.app, assignment.Id, result.Currency, "")
		if creditErr != nil {
			return FinancialSummary{}, creditErr
		}
		for _, credit := range credits {
			result.OpenCreditMinor += credit.RemainingMinor
			if result.Currency == "" {
				result.Currency = credit.Currency
			}
		}
	}
	if result.Currency == "" {
		result.Currency = "PLN"
	}
	return result, nil
}

func addFinancialSummaryCharge(result *FinancialSummary, app core.App, charge *core.Record, assignments []*core.Record, now time.Time) {
	state := charge.GetString(schedulingstore.SettlementStateField)
	if charge.GetString(schedulingstore.SourceTypeField) == "ad_hoc" {
		if state == string(ledger.PendingSettlement) {
			result.PendingSettlement++
		}
		if state == string(ledger.IntentionallyUnpaid) {
			result.UnpaidAdHoc++
		}
	} else if charge.GetInt(schedulingstore.CurrentAmountMinorField) > 0 && (state == string(ledger.PendingPayment) || state == string(ledger.IntentionallyUnpaid)) {
		result.UnpaidCharges++
	}
	if chargeView(app, charge, now, assignments).Overdue {
		result.OverdueCharges++
	}
	if result.Currency == "" {
		result.Currency = charge.GetString(schedulingstore.CurrencyField)
	}
}

func (s *PocketBaseService) FinancialHistory(_ context.Context, actor ledger.Actor, assignmentID string, page, perPage int) (FinancialHistoryPage, error) {
	if page < 1 || perPage < 1 || perPage > 100 {
		return FinancialHistoryPage{}, errPagination
	}
	if _, err := assignmentForActor(s.app, assignmentID, actor); err != nil {
		return FinancialHistoryPage{}, err
	}
	rows, err := s.app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return FinancialHistoryPage{}, err
	}
	owned := make([]*core.Record, 0)
	for _, row := range rows {
		if row.GetString(schedulingstore.AssignmentField) == assignmentID {
			owned = append(owned, row)
		}
	}
	sort.SliceStable(owned, func(left, right int) bool {
		return recordTime(owned[left], schedulingstore.EventAtField).After(recordTime(owned[right], schedulingstore.EventAtField))
	})
	return entryPage(owned, page, perPage), nil
}

func (s *PocketBaseService) UnresolvedWork(_ context.Context, actor ledger.Actor) (UnresolvedWork, error) {
	assignments, err := assignmentsForActor(s.app, actor)
	if err != nil {
		return UnresolvedWork{}, err
	}
	owned := assignmentSet(assignments)
	rows, err := s.app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		return UnresolvedWork{}, err
	}
	now := s.instant()
	result := UnresolvedWork{AwaitingOutcome: make([]UnresolvedOutcome, 0), PendingSettlements: make([]PendingSettlement, 0), UnpaidCharges: make([]ChargeView, 0), OverdueCharges: make([]ChargeView, 0)}
	result.AwaitingOutcome = awaitingOutcomes(rows, owned, now)
	charges, err := chargesForAssignments(s.app, owned)
	if err != nil {
		return UnresolvedWork{}, err
	}
	result.PendingSettlements, result.UnpaidCharges, result.OverdueCharges = unresolvedCharges(s.app, charges, assignments, now)
	result.AwaitingOutcome = capped(result.AwaitingOutcome, 100)
	result.PendingSettlements = capped(result.PendingSettlements, 100)
	result.UnpaidCharges = capped(result.UnpaidCharges, 100)
	result.OverdueCharges = capped(result.OverdueCharges, 100)
	return result, nil
}

func awaitingOutcomes(rows []*core.Record, owned map[string]bool, now time.Time) []UnresolvedOutcome {
	result := make([]UnresolvedOutcome, 0)
	for _, row := range rows {
		if !owned[row.GetString(schedulingstore.AssignmentField)] || row.GetString(schedulingstore.ScheduleStateField) != "scheduled" || !recordTime(row, schedulingstore.EndAtField).Before(now) {
			continue
		}
		if row.GetString(schedulingstore.OutcomeField) == "" || row.GetString(schedulingstore.OutcomeField) == "awaiting_outcome" {
			result = append(result, UnresolvedOutcome{LessonID: row.Id, AssignmentID: row.GetString(schedulingstore.AssignmentField), EndedAt: utc(recordTime(row, schedulingstore.EndAtField))})
		}
	}
	return result
}

func unresolvedCharges(app core.App, rows []*core.Record, assignments []*core.Record, now time.Time) ([]PendingSettlement, []ChargeView, []ChargeView) {
	pending, unpaid, overdue := make([]PendingSettlement, 0), make([]ChargeView, 0), make([]ChargeView, 0)
	for _, row := range rows {
		view := chargeView(app, row, now, assignments)
		state := row.GetString(schedulingstore.SettlementStateField)
		if row.GetString(schedulingstore.SourceTypeField) == "ad_hoc" && state == string(ledger.PendingSettlement) && lessonEnded(app, row.GetString(schedulingstore.SourceIDField), now) {
			pending = append(pending, PendingSettlement{LessonID: row.GetString(schedulingstore.SourceIDField), AssignmentID: row.GetString(schedulingstore.AssignmentField), AmountMinor: int64(row.GetInt(schedulingstore.OriginalAmountMinorField)), Currency: row.GetString(schedulingstore.CurrencyField)})
		}
		if row.GetInt(schedulingstore.CurrentAmountMinorField) > 0 && (state == string(ledger.PendingPayment) || state == string(ledger.IntentionallyUnpaid)) {
			unpaid = append(unpaid, view)
		}
		if view.Overdue {
			overdue = append(overdue, view)
		}
	}
	return pending, unpaid, overdue
}

func lessonEnded(app core.App, lessonID string, now time.Time) bool {
	lesson, err := app.FindRecordById(schedulingstore.LessonsCollectionName, lessonID)
	return err == nil && !recordTime(lesson, schedulingstore.EndAtField).After(now)
}

func assignmentSet(rows []*core.Record) map[string]bool {
	result := make(map[string]bool, len(rows))
	for _, row := range rows {
		result[row.Id] = true
	}
	return result
}

func entriesForAssignments(app core.App, assignments map[string]bool) ([]*core.Record, error) {
	rows, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return nil, err
	}
	result := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if assignments[row.GetString(schedulingstore.AssignmentField)] {
			result = append(result, row)
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return recordTime(result[left], schedulingstore.EventAtField).After(recordTime(result[right], schedulingstore.EventAtField))
	})
	return capped(result, 100), nil
}

func entryPage(rows []*core.Record, page, perPage int) FinancialHistoryPage {
	start := (page - 1) * perPage
	if start > len(rows) {
		start = len(rows)
	}
	end := start + perPage
	if end > len(rows) {
		end = len(rows)
	}
	items := entryViews(rows[start:end])
	totalPages := int(math.Ceil(float64(len(rows)) / float64(perPage)))
	return FinancialHistoryPage{Items: items, Page: page, PerPage: perPage, TotalItems: len(rows), TotalPages: totalPages, Total: len(rows)}
}

func entryViews(rows []*core.Record) []FinancialEntryView {
	result := make([]FinancialEntryView, 0, len(rows))
	for _, row := range rows {
		result = append(result, FinancialEntryView{ID: row.Id, AssignmentID: row.GetString(schedulingstore.AssignmentField), ChargeID: row.GetString(schedulingstore.ChargeField), EntryType: row.GetString(schedulingstore.EntryTypeField), AmountMinor: int64(row.GetInt(schedulingstore.AmountMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), EffectiveOn: row.GetString(schedulingstore.EffectiveOnField), RelatedLessonID: row.GetString(schedulingstore.RelatedLessonField), RelatedEntryID: row.GetString(schedulingstore.RelatedEntryField), ActorRole: row.GetString(schedulingstore.ActorRoleField), EventAt: utc(recordTime(row, schedulingstore.EventAtField))})
	}
	return result
}

func creditViews(rows []*core.Record) []FinancialEntryView {
	result := make([]FinancialEntryView, 0)
	for _, row := range rows {
		if row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryCreditCreated) {
			result = append(result, entryViews([]*core.Record{row})...)
		}
	}
	return result
}

func chargeViews(app core.App, rows []*core.Record, now time.Time, assignments []*core.Record) []ChargeView {
	result := make([]ChargeView, 0, len(rows))
	for _, row := range rows {
		result = append(result, chargeView(app, row, now, assignments))
	}
	return capped(result, 100)
}

func chargeView(app core.App, row *core.Record, now time.Time, assignments []*core.Record) ChargeView {
	location := time.UTC
	for _, assignment := range assignments {
		if assignment.Id == row.GetString(schedulingstore.AssignmentField) {
			location = teacherLocation(app, assignment)
			break
		}
	}
	charge := ledger.Charge{ID: row.Id, AssignmentID: row.GetString(schedulingstore.AssignmentField), OriginalAmountMinor: int64(row.GetInt(schedulingstore.OriginalAmountMinorField)), CurrentAmountMinor: int64(row.GetInt(schedulingstore.CurrentAmountMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), SettlementState: ledger.SettlementState(row.GetString(schedulingstore.SettlementStateField)), DueOn: row.GetString(schedulingstore.DueOnField)}
	return ChargeView{ID: row.Id, AssignmentID: charge.AssignmentID, SourceType: row.GetString(schedulingstore.SourceTypeField), SourceID: row.GetString(schedulingstore.SourceIDField), Period: row.GetString(schedulingstore.PeriodField), OriginalAmountMinor: charge.OriginalAmountMinor, CurrentAmountMinor: charge.CurrentAmountMinor, Currency: charge.Currency, SettlementState: string(charge.SettlementState), DerivedState: string(charge.DerivedState(now, location)), Overdue: charge.IsOverdue(now, location), DueOn: charge.DueOn, PaidAt: utc(recordTime(row, schedulingstore.PaidAtField))}
}

func teacherLocation(app core.App, assignment *core.Record) *time.Location {
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err == nil {
		if location, locationErr := time.LoadLocation(teacher.GetString(schedulingstore.TeacherTimezoneField)); locationErr == nil {
			return location
		}
	}
	return time.UTC
}

func utc(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
