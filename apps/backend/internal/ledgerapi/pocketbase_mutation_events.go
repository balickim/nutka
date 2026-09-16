// This file appends role-safe business events for ledger commands.
// Event writes share the command transaction and preserve correction history.
package ledgerapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func appendPaymentEvent(app core.App, assignmentID, chargeID string, actor ledger.Actor, prior, next ledger.SettlementState, at time.Time) error {
	event, err := history.NewPaymentRecorded(history.EventInput{AggregateType: "charge", AggregateID: chargeID, AssignmentID: assignmentID, Actor: history.Actor{Role: history.ActorRole(actor.Role), ID: actor.ID}, EventAt: at, PriorState: map[string]any{"settlement_state": prior}, NewState: map[string]any{"settlement_state": next}})
	if err != nil {
		return err
	}
	return (history.Storage{}).Append(app, event)
}

func appendRefundEvent(app core.App, assignmentID, chargeID string, actor ledger.Actor, amount int64, reason string, at time.Time) error {
	event, err := history.NewRefundRecorded(history.EventInput{AggregateType: "charge", AggregateID: chargeID, AssignmentID: assignmentID, Actor: history.Actor{Role: history.ActorRole(actor.Role), ID: actor.ID}, EventAt: at, Reason: reason, NewState: map[string]any{"amount_minor": amount}})
	if err != nil {
		return err
	}
	return (history.Storage{}).Append(app, event)
}

func appendCorrectionEvent(app core.App, assignmentID, chargeID string, actor ledger.Actor, entryID, reason, note string, at time.Time) error {
	event, err := history.NewEvent(history.AdministrativeCorrection, history.EventInput{AggregateType: "charge", AggregateID: chargeID, AssignmentID: assignmentID, Actor: history.Actor{Role: history.ActorRole(actor.Role), ID: actor.ID}, EventAt: at, Reason: reason, InternalNote: note, PriorState: map[string]any{"entry_id": entryID}})
	if err != nil {
		return err
	}
	event.CorrectsEventID = entryID
	return (history.Storage{}).Append(app, event)
}

func availableCredit(app core.App, chargeID, assignmentID, currency string) (ledger.Credit, error) {
	rows, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return ledger.Credit{}, err
	}
	matching := matchingCreditEntries(rows, chargeID, assignmentID, currency)
	if len(matching) == 0 {
		return ledger.Credit{}, ledger.ErrInsufficientCredit
	}
	source := matching[0]
	remaining := creditBalance(matching)
	if remaining <= 0 {
		return ledger.Credit{}, ledger.ErrInsufficientCredit
	}
	return ledger.Credit{ID: source.Id, AssignmentID: assignmentID, AmountMinor: remaining, RemainingMinor: remaining, Currency: currency, SourceChargeID: chargeID, CreatedAt: recordTime(source, schedulingstore.EventAtField)}, nil
}

func matchingCreditEntries(rows []*core.Record, chargeID, assignmentID, currency string) []*core.Record {
	result := make([]*core.Record, 0)
	for _, row := range rows {
		if row.GetString(schedulingstore.AssignmentField) != assignmentID || row.GetString(schedulingstore.ChargeField) != chargeID || row.GetString(schedulingstore.CurrencyField) != currency {
			continue
		}
		if row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryCreditCreated) || row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryCreditApplied) || row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryRefund) {
			result = append(result, row)
		}
	}
	return result
}

func creditBalance(rows []*core.Record) int64 {
	var remaining int64
	for _, row := range rows {
		amount := int64(row.GetInt(schedulingstore.AmountMinorField))
		if row.GetString(schedulingstore.EntryTypeField) == string(ledger.EntryCreditCreated) {
			remaining += amount
		} else {
			remaining -= amount
		}
	}
	return remaining
}
