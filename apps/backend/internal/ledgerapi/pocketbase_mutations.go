// This file applies teacher ledger commands through ledger domain decisions.
// Charge, financial-entry, and business-event writes share one transaction.
package ledgerapi

import (
	"context"
	"errors"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func (s *PocketBaseService) SettleAdHoc(_ context.Context, actor ledger.Actor, lessonID string, state ledger.SettlementState) (LessonPayment, error) {
	var result LessonPayment
	err := transaction(context.Background(), s.app, func(tx core.App) error {
		lesson, assignment, err := lessonForTeacher(tx, lessonID, actor)
		if err != nil {
			return err
		}
		if lesson.GetString(schedulingstore.PlanTypeField) != "ad_hoc" {
			return errNotAdHoc
		}
		charge, err := chargeForLesson(tx, lessonID)
		if err != nil {
			return err
		}
		domainLesson := ledger.AdHocLesson{ID: lessonID, AssignmentID: assignment.Id, StartAt: recordTime(lesson, schedulingstore.StartAtField), EndAt: recordTime(lesson, schedulingstore.EndAtField), UnitPriceMinor: int64(lesson.GetInt(schedulingstore.UnitPriceMinorField)), Currency: lesson.GetString(schedulingstore.CurrencyField), SettlementState: ledger.SettlementState(charge.GetString(schedulingstore.SettlementStateField))}
		decision, err := ledger.SettleAdHoc(domainLesson, ledger.AdHocSettlementCommand{State: state, Actor: actor, At: s.instant()})
		if err != nil {
			return err
		}
		charge.Set(schedulingstore.SettlementStateField, string(decision.Lesson.SettlementState))
		if decision.Lesson.SettlementState == ledger.Paid {
			charge.Set(schedulingstore.PaidAtField, s.instant())
		}
		if err := tx.Save(charge); err != nil {
			return err
		}
		if decision.Entry == nil {
			return errors.New("settlement entry is missing")
		}
		if err := saveEntry(tx, *decision.Entry, charge.Id); err != nil {
			return err
		}
		if err := appendSettlementEvent(tx, assignment.Id, lessonID, actor, decision.Transition, s.instant()); err != nil {
			return err
		}
		result = LessonPayment{LessonID: lessonID, AssignmentID: assignment.Id, AmountMinor: decision.Entry.AmountMinor, Currency: decision.Entry.Currency, SettlementState: string(decision.Lesson.SettlementState)}
		return nil
	})
	if err != nil {
		return LessonPayment{}, wrapRecordError("settle ad hoc lesson", err)
	}
	return result, nil
}

func (s *PocketBaseService) PayCharge(_ context.Context, actor ledger.Actor, chargeID string, state ledger.SettlementState) (ChargeView, error) {
	var result ChargeView
	err := transaction(context.Background(), s.app, func(tx core.App) error {
		charge, assignment, err := chargeForTeacher(tx, chargeID, actor)
		if err != nil {
			return err
		}
		prior := ledger.SettlementState(charge.GetString(schedulingstore.SettlementStateField))
		if !validPaymentTransition(prior, state) {
			return ledger.ErrInvalidTransition
		}
		charge.Set(schedulingstore.SettlementStateField, string(state))
		if state == ledger.Paid {
			charge.Set(schedulingstore.PaidAtField, s.instant())
		}
		if err := tx.Save(charge); err != nil {
			return err
		}
		entryType := ledger.EntrySettlementPaid
		if state == ledger.IntentionallyUnpaid {
			entryType = ledger.EntrySettlementUnpaid
		}
		entry := ledger.FinancialEntry{AssignmentID: assignment.Id, ChargeID: charge.Id, EntryType: entryType, AmountMinor: int64(charge.GetInt(schedulingstore.CurrentAmountMinorField)), Currency: charge.GetString(schedulingstore.CurrencyField), Actor: actor, EventAt: s.instant()}
		if err := saveEntry(tx, entry, charge.Id); err != nil {
			return err
		}
		return appendPaymentEvent(tx, assignment.Id, charge.Id, actor, prior, state, s.instant())
	})
	if err != nil {
		return ChargeView{}, wrapRecordError("record charge payment", err)
	}
	result = chargeView(s.app, mustCharge(s.app, chargeID), s.instant(), mustAssignments(s.app, actor))
	return result, nil
}

func validPaymentTransition(prior, next ledger.SettlementState) bool {
	if !ledger.ValidChargeState(prior) || (next != ledger.Paid && next != ledger.IntentionallyUnpaid) {
		return false
	}
	return prior != ledger.Paid || next == ledger.Paid
}

func (s *PocketBaseService) RefundCharge(_ context.Context, actor ledger.Actor, chargeID string, amount int64, reason string) (ChargeView, error) {
	var result ChargeView
	err := transaction(context.Background(), s.app, func(tx core.App) error {
		charge, assignment, err := chargeForTeacher(tx, chargeID, actor)
		if err != nil {
			return err
		}
		credit, err := availableCredit(tx, chargeID, assignment.Id, charge.GetString(schedulingstore.CurrencyField))
		if err != nil {
			return err
		}
		if amount <= 0 || amount > credit.RemainingMinor {
			return ledger.ErrInsufficientCredit
		}
		credit.RemainingMinor = amount
		_, refund, entry, err := ledger.RefundCredit(credit, chargeID, reason, actor, s.instant())
		if err != nil {
			return err
		}
		entry.AmountMinor = amount
		entry.RelatedEntryID = refund.CreditID
		if err := saveEntry(tx, entry, chargeID); err != nil {
			return err
		}
		return appendRefundEvent(tx, assignment.Id, chargeID, actor, amount, reason, s.instant())
	})
	if err != nil {
		return ChargeView{}, wrapRecordError("record charge refund", err)
	}
	result = chargeView(s.app, mustCharge(s.app, chargeID), s.instant(), mustAssignments(s.app, actor))
	return result, nil
}

func (s *PocketBaseService) CorrectCharge(_ context.Context, actor ledger.Actor, chargeID, reason, note string) (ChargeView, error) {
	err := transaction(context.Background(), s.app, func(tx core.App) error {
		charge, assignment, err := chargeForTeacher(tx, chargeID, actor)
		if err != nil {
			return err
		}
		entries, err := tx.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
		if err != nil {
			return err
		}
		var target *core.Record
		for _, entry := range entries {
			if entry.GetString(schedulingstore.ChargeField) == charge.Id && (target == nil || recordTime(entry, schedulingstore.EventAtField).After(recordTime(target, schedulingstore.EventAtField))) {
				target = entry
			}
		}
		if target == nil {
			return ledger.ErrInvalidTransition
		}
		correction := ledger.Correction{ID: charge.Id + ":correction:" + s.instant().Format("20060102150405.000000000"), AssignmentID: assignment.Id, CorrectsEntryID: target.Id, Currency: charge.GetString(schedulingstore.CurrencyField), Reason: reason, Actor: actor, RecordedAt: s.instant()}
		_, entry, err := ledger.NewCorrection(correction)
		if err != nil {
			return err
		}
		entry.ChargeID = charge.Id
		if err := saveEntry(tx, entry, charge.Id); err != nil {
			return err
		}
		return appendCorrectionEvent(tx, assignment.Id, charge.Id, actor, target.Id, reason, note, s.instant())
	})
	if err != nil {
		return ChargeView{}, wrapRecordError("correct charge", err)
	}
	return chargeView(s.app, mustCharge(s.app, chargeID), s.instant(), mustAssignments(s.app, actor)), nil
}

func chargeForTeacher(app core.App, id string, actor ledger.Actor) (*core.Record, *core.Record, error) {
	charge, err := findRecord(app, schedulingstore.ChargesCollectionName, id)
	if err != nil {
		return nil, nil, errOwnership
	}
	assignment, err := assignmentForActor(app, charge.GetString(schedulingstore.AssignmentField), actor)
	if err != nil || actor.Role != ledger.TeacherActor {
		return nil, nil, errOwnership
	}
	return charge, assignment, nil
}

func chargeForLesson(app core.App, lessonID string) (*core.Record, error) {
	rows, err := app.FindAllRecords(schedulingstore.ChargesCollectionName)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.GetString(schedulingstore.SourceTypeField) == "ad_hoc" && row.GetString(schedulingstore.SourceIDField) == lessonID {
			return row, nil
		}
	}
	return nil, errRecordNotFound
}

func mustCharge(app core.App, id string) *core.Record {
	row, _ := findRecord(app, schedulingstore.ChargesCollectionName, id)
	return row
}
func mustAssignments(app core.App, actor ledger.Actor) []*core.Record {
	rows, _ := assignmentsForActor(app, actor)
	return rows
}

func saveEntry(app core.App, entry ledger.FinancialEntry, chargeID string) error {
	_, err := saveEntryRecord(app, entry, chargeID)
	return err
}

func saveEntryRecord(app core.App, entry ledger.FinancialEntry, chargeID string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return nil, err
	}
	row := core.NewRecord(collection)
	row.Set(schedulingstore.AssignmentField, entry.AssignmentID)
	row.Set(schedulingstore.ChargeField, chargeID)
	row.Set(schedulingstore.EntryTypeField, string(entry.EntryType))
	row.Set(schedulingstore.AmountMinorField, entry.AmountMinor)
	row.Set(schedulingstore.CurrencyField, entry.Currency)
	row.Set(schedulingstore.EffectiveOnField, entry.EffectiveOn)
	row.Set(schedulingstore.RelatedLessonField, entry.RelatedLessonID)
	row.Set(schedulingstore.RelatedEntryField, entry.RelatedEntryID)
	row.Set(schedulingstore.ActorRoleField, entry.Actor.Role)
	row.Set(schedulingstore.ActorIDField, entry.Actor.ID)
	row.Set(schedulingstore.ReasonField, entry.Reason)
	row.Set(schedulingstore.EventAtField, entry.EventAt)
	if err := app.Save(row); err != nil {
		return nil, err
	}
	return row, nil
}

func appendSettlementEvent(app core.App, assignmentID, lessonID string, actor ledger.Actor, transition ledger.SettlementTransition, at time.Time) error {
	event, err := history.NewSettlementChanged(history.EventInput{AggregateType: "lesson", AggregateID: lessonID, AssignmentID: assignmentID, Actor: history.Actor{Role: history.ActorRole(actor.Role), ID: actor.ID}, EventAt: at, PriorState: map[string]any{"settlement": transition.From}, NewState: map[string]any{"settlement": transition.To}})
	if err != nil {
		return err
	}
	return (history.Storage{}).Append(app, event)
}
