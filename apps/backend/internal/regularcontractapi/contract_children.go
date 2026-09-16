// This file persists and restores contract amendments and immutable allowance events.
package regularcontractapi

import (
	"encoding/json"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func addContractChildren(app core.App, value regularcontract.RegularContract) (regularcontract.RegularContract, error) {
	var err error
	if value.Occurrences, err = loadOccurrences(app, value); err != nil {
		return value, err
	}
	if value.Amendments, err = loadAmendments(app, value.ID); err != nil {
		return value, err
	}
	if value.Events, err = loadContractEvents(app, value.ID); err != nil {
		return value, err
	}
	return value, nil
}

func loadOccurrences(app core.App, contract regularcontract.RegularContract) ([]regularcontract.Occurrence, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.ContractField: contract.ID})
	if err != nil {
		return nil, err
	}
	result := make([]regularcontract.Occurrence, 0, len(rows))
	for _, row := range rows {
		interval, intervalErr := scheduling.NewInterval(instant(row, schedulingstore.StartAtField), instant(row, schedulingstore.EndAtField))
		if intervalErr != nil {
			return nil, intervalErr
		}
		billing := regularcontract.BillingOutcome(row.GetString(schedulingstore.BillingOutcomeField))
		if billing == "" {
			billing = regularcontract.BillableOrdinary
		}
		result = append(result, regularcontract.Occurrence{ID: row.Id, ContractID: contract.ID, AssignmentID: contract.AssignmentID, TeacherID: contract.TeacherID, LearnerID: contract.LearnerID, OriginalLocalDate: row.GetString(schedulingstore.OriginalLocalDateField), OriginalStartAt: instant(row, schedulingstore.OriginalStartAtField), Interval: interval, ScheduleState: regularcontract.ScheduleState(row.GetString(schedulingstore.ScheduleStateField)), Outcome: row.GetString(schedulingstore.OutcomeField), BillingOutcome: billing, UnitPriceMinor: int64(row.GetInt(schedulingstore.UnitPriceMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), IndividuallyRescheduled: row.GetBool(schedulingstore.IndividuallyRescheduledField), OmissionReason: row.GetString(schedulingstore.OmissionReasonField)})
	}
	return result, nil
}

func loadAmendments(app core.App, contractID string) ([]regularcontract.Amendment, error) {
	rows, err := app.FindAllRecords(schedulingstore.ContractAmendmentsCollectionName, dbx.HashExp{schedulingstore.ContractField: contractID})
	if err != nil {
		return nil, err
	}
	result := make([]regularcontract.Amendment, 0, len(rows))
	for _, row := range rows {
		effective := row.GetString(schedulingstore.EffectiveOnField)
		if len(effective) == 10 {
			effective = effective[:7]
		}
		result = append(result, regularcontract.Amendment{EffectiveMonth: effective, PriceMinor: int64(row.GetInt(schedulingstore.UnitPriceMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), CreatedAt: instant(row, schedulingstore.CreatedAtField), Actor: regularcontract.Actor{Role: regularcontract.Teacher}})
	}
	return result, nil
}

func saveAmendments(app core.App, value regularcontract.RegularContract) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.ContractAmendmentsCollectionName)
	if err != nil {
		return err
	}
	for _, amendment := range value.Amendments {
		effectiveOn := amendment.EffectiveMonth + "-01"
		rows, _ := app.FindAllRecords(schedulingstore.ContractAmendmentsCollectionName, dbx.And(dbx.HashExp{schedulingstore.ContractField: value.ID}, dbx.HashExp{schedulingstore.EffectiveOnField: effectiveOn}))
		row := core.NewRecord(collection)
		if len(rows) > 0 {
			row = rows[0]
		}
		snapshot, _ := json.Marshal(value.PolicySnapshot)
		row.Set(schedulingstore.ContractField, value.ID)
		row.Set(schedulingstore.EffectiveOnField, effectiveOn)
		row.Set(schedulingstore.UnitPriceMinorField, amendment.PriceMinor)
		row.Set(schedulingstore.CurrencyField, amendment.Currency)
		row.Set(schedulingstore.PolicyVersionField, value.PolicySnapshot.Version)
		row.Set(schedulingstore.PolicySnapshotField, snapshot)
		row.Set(schedulingstore.CreatedAtField, amendment.CreatedAt.UTC().Format(time.RFC3339Nano))
		if err := app.Save(row); err != nil {
			return err
		}
	}
	return nil
}

func loadContractEvents(app core.App, contractID string) ([]regularcontract.Event, error) {
	rows, err := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName, dbx.And(dbx.HashExp{schedulingstore.AggregateTypeField: "contract"}, dbx.HashExp{schedulingstore.AggregateIDField: contractID}))
	if err != nil {
		return nil, err
	}
	legacyByRecord := make(map[string]string, len(rows))
	for _, row := range rows {
		legacyByRecord[row.Id] = row.GetString(schedulingstore.LegacyEventIDField)
	}
	result := make([]regularcontract.Event, 0, len(rows))
	for _, row := range rows {
		related := jsonObject(row.Get(schedulingstore.RelatedIDsField))
		prior := eventInterval(jsonObject(row.Get(schedulingstore.PriorStateField)))
		next := eventInterval(jsonObject(row.Get(schedulingstore.NewStateField)))
		result = append(result, regularcontract.Event{ID: row.GetString(schedulingstore.LegacyEventIDField), Type: row.GetString(schedulingstore.EventTypeField), ContractID: contractID, OccurrenceID: textValue(related["occurrence_id"]), OriginalMonth: textValue(related["original_month"]), Actor: regularcontract.Actor{Role: regularcontract.ActorRole(row.GetString(schedulingstore.ActorRoleField)), ID: row.GetString(schedulingstore.ActorIDField)}, At: instant(row, schedulingstore.EventAtField), Reason: row.GetString(schedulingstore.ReasonField), CorrectsEvent: legacyByRecord[row.GetString(schedulingstore.CorrectsEventField)], PriorInterval: prior, NewInterval: next})
	}
	return result, nil
}

func saveEvents(app core.App, value regularcontract.RegularContract) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		return err
	}
	for _, event := range value.Events {
		existing, _ := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName, dbx.HashExp{schedulingstore.LegacyEventIDField: event.ID})
		if len(existing) > 0 {
			continue
		}
		row := core.NewRecord(collection)
		row.Set(schedulingstore.AssignmentField, value.AssignmentID)
		row.Set(schedulingstore.AggregateTypeField, "contract")
		row.Set(schedulingstore.AggregateIDField, value.ID)
		row.Set(schedulingstore.EventTypeField, event.Type)
		row.Set(schedulingstore.ActorRoleField, string(event.Actor.Role))
		row.Set(schedulingstore.ActorIDField, event.Actor.ID)
		row.Set(schedulingstore.EventAtField, event.At.UTC().Format(time.RFC3339Nano))
		row.Set(schedulingstore.RelatedIDsField, map[string]string{"occurrence_id": event.OccurrenceID, "original_month": event.OriginalMonth})
		row.Set(schedulingstore.PriorStateField, intervalState(event.PriorInterval))
		row.Set(schedulingstore.NewStateField, intervalState(event.NewInterval))
		row.Set(schedulingstore.ReasonField, event.Reason)
		row.Set(schedulingstore.LegacyEventIDField, event.ID)
		if event.CorrectsEvent != "" {
			corrected, _ := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName, dbx.HashExp{schedulingstore.LegacyEventIDField: event.CorrectsEvent})
			if len(corrected) > 0 {
				row.Set(schedulingstore.CorrectsEventField, corrected[0].Id)
			}
		}
		if err := app.Save(row); err != nil {
			return err
		}
	}
	return nil
}

func intervalState(value scheduling.Interval) map[string]string {
	if !value.Valid() {
		return nil
	}
	return map[string]string{"start_at": value.Start.UTC().Format(time.RFC3339Nano), "end_at": value.End.UTC().Format(time.RFC3339Nano)}
}

func eventInterval(value map[string]any) scheduling.Interval {
	start, startErr := time.Parse(time.RFC3339Nano, textValue(value["start_at"]))
	end, endErr := time.Parse(time.RFC3339Nano, textValue(value["end_at"]))
	if startErr != nil || endErr != nil {
		return scheduling.Interval{}
	}
	result, _ := scheduling.NewInterval(start.UTC(), end.UTC())
	return result
}

func jsonObject(value any) map[string]any {
	var encoded []byte
	switch typed := value.(type) {
	case string:
		encoded = []byte(typed)
	case []byte:
		encoded = typed
	case types.JSONRaw:
		encoded = []byte(typed)
	case nil:
		return nil
	default:
		encoded, _ = json.Marshal(typed)
	}
	var result map[string]any
	_ = json.Unmarshal(encoded, &result)
	return result
}

func textValue(value any) string {
	text, _ := value.(string)
	return text
}
