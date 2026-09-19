// This file builds the learner payment-due read for one assignment from charges, credits, contract months, and teacher details.
// The open-item rule lives in ledger.DueItems. This file only loads records and projects DTOs.
package ledgerapi

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/paymentdetails"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const recentPaymentLimit = 5

func (s *PocketBaseService) PaymentDue(ctx context.Context, actor ledger.Actor, assignmentID string) (PaymentDue, error) {
	assignment, err := assignmentForActor(s.app, assignmentID, actor)
	if err != nil {
		return PaymentDue{}, err
	}
	charges, err := chargesForAssignments(s.app, map[string]bool{assignment.Id: true})
	if err != nil {
		return PaymentDue{}, err
	}
	now, location := s.instant(), teacherLocation(s.app, assignment)
	items, total := ledger.DueItems(s.dueCharges(charges, now), now, location)
	result := PaymentDue{AssignmentID: assignment.Id, Currency: "PLN", TotalMinor: total, Items: dueItemViews(items), RecentPayments: s.recentPayments(charges), Instructions: teacherInstructions(s.app, assignment)}
	credits, err := availableCredits(s.app, assignment.Id, "", "")
	if err != nil {
		return PaymentDue{}, err
	}
	for _, credit := range credits {
		result.OpenCreditMinor += credit.RemainingMinor
	}
	result.NextForecast, err = s.nextForecast(ctx, actor, assignment.Id, now.In(location).Format("2006-01"))
	return result, err
}

func (s *PocketBaseService) dueCharges(rows []*core.Record, now time.Time) []ledger.DueCharge {
	result := make([]ledger.DueCharge, 0, len(rows))
	for _, row := range rows {
		due := ledger.DueCharge{Charge: ledger.Charge{ID: row.Id, SourceType: row.GetString(schedulingstore.SourceTypeField), Period: row.GetString(schedulingstore.PeriodField), CurrentAmountMinor: int64(row.GetInt(schedulingstore.CurrentAmountMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), SettlementState: ledger.SettlementState(row.GetString(schedulingstore.SettlementStateField)), DueOn: row.GetString(schedulingstore.DueOnField)}}
		if due.Charge.SourceType == ledger.AdHocSource {
			if lesson, err := s.app.FindRecordById(schedulingstore.LessonsCollectionName, row.GetString(schedulingstore.SourceIDField)); err == nil {
				due.LessonStartAt = recordTime(lesson, schedulingstore.StartAtField)
				due.LessonEnded = !recordTime(lesson, schedulingstore.EndAtField).After(now)
			}
		}
		result = append(result, due)
	}
	return result
}

func dueItemViews(items []ledger.DueItem) []DueItemView {
	result := make([]DueItemView, 0, len(items))
	for _, item := range items {
		result = append(result, DueItemView{ChargeID: item.ChargeID, Kind: item.Kind, Period: item.Period, LessonStartAt: utc(item.LessonStartAt), AmountMinor: item.AmountMinor, DueOn: item.DueOn, Overdue: item.Overdue})
	}
	return result
}

func (s *PocketBaseService) recentPayments(rows []*core.Record) []RecentPayment {
	paid := make([]*core.Record, 0)
	for _, row := range rows {
		if row.GetString(schedulingstore.SettlementStateField) == string(ledger.Paid) {
			paid = append(paid, row)
		}
	}
	sort.SliceStable(paid, func(left, right int) bool {
		return recordTime(paid[left], schedulingstore.PaidAtField).After(recordTime(paid[right], schedulingstore.PaidAtField))
	})
	result := make([]RecentPayment, 0, recentPaymentLimit)
	for _, row := range capped(paid, recentPaymentLimit) {
		payment := RecentPayment{ChargeID: row.Id, Kind: ledger.DueKindContractMonth, Period: row.GetString(schedulingstore.PeriodField), AmountMinor: int64(row.GetInt(schedulingstore.CurrentAmountMinorField)), PaidAt: utc(recordTime(row, schedulingstore.PaidAtField))}
		if row.GetString(schedulingstore.SourceTypeField) == ledger.AdHocSource {
			payment.Kind = ledger.DueKindLesson
			if lesson, err := s.app.FindRecordById(schedulingstore.LessonsCollectionName, row.GetString(schedulingstore.SourceIDField)); err == nil {
				payment.LessonStartAt = utc(recordTime(lesson, schedulingstore.StartAtField))
			}
		}
		result = append(result, payment)
	}
	return result
}

// nextForecast returns the earliest contract month after the current teacher-local month that has no charge yet.
func (s *PocketBaseService) nextForecast(ctx context.Context, actor ledger.Actor, assignmentID, currentMonth string) (*DueForecast, error) {
	contracts, err := s.app.FindAllRecords(schedulingstore.RegularContractsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return nil, err
	}
	var next *DueForecast
	for _, contract := range contracts {
		page, err := s.ContractMonths(ctx, actor, contract.Id, 1, 100)
		if err != nil {
			return nil, err
		}
		for _, month := range page.Items {
			if !month.Forecast || month.Month <= currentMonth || month.BillableCount == 0 || (next != nil && month.Month >= next.Month) {
				continue
			}
			next = &DueForecast{Month: month.Month, LessonCount: month.BillableCount, AmountMinor: month.ForecastAmountMinor, DueOn: fmt.Sprintf("%s-%02d", month.Month, businesspolicy.MonthlyPaymentDueDay)}
		}
	}
	return next, nil
}

func teacherInstructions(app core.App, assignment *core.Record) *paymentdetails.Details {
	teacher, err := app.FindRecordById(authconfig.TeachersCollectionName, assignment.GetString("teacher"))
	if err != nil {
		return nil
	}
	return paymentdetails.FromRecord(teacher)
}
