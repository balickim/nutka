// This file persists availability mutations, distant occurrence effects, reconciliation, and audit events.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/availabilityimpact"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/ledgerapi"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func applyMutation(app core.App, teacherID string, mutation AvailabilityMutation, result *CommitResult) error {
	if mutation.Operation == availabilityimpact.Delete {
		collection := schedulingstore.AvailabilityRulesCollectionName
		if mutation.Target == availabilityimpact.Exception {
			collection = schedulingstore.AvailabilityExceptionsCollectionName
		}
		row, err := ownedRecord(app, collection, mutation.ID, "teacher", teacherID)
		if err != nil {
			return err
		}
		return app.Delete(row)
	}
	if mutation.Target == availabilityimpact.RecurringRule {
		return saveRuleMutation(app, teacherID, mutation, result)
	}
	return saveExceptionMutation(app, teacherID, mutation, result)
}

func applyDistantEffects(app core.App, effects []availabilityimpact.DistantEffect, now time.Time) error {
	contracts := make(map[string]bool)
	for _, effect := range effects {
		row, err := app.FindRecordById(schedulingstore.LessonsCollectionName, effect.OccurrenceID)
		if err != nil {
			return errStalePreview
		}
		applyDistantEffect(row, effect)
		if err := app.Save(row); err != nil {
			return err
		}
		if err := saveDistantEffectEvent(app, row, effect, now); err != nil {
			return err
		}
		contracts[row.GetString(schedulingstore.ContractField)] = true
	}
	return reconcileAffectedContracts(app, contracts, now)
}

func applyDistantEffect(row *core.Record, effect availabilityimpact.DistantEffect) {
	if effect.Effect == availabilityimpact.Omit {
		row.Set(schedulingstore.ScheduleStateField, string(domain.OmittedState))
		row.Set(schedulingstore.BillingOutcomeField, "planned_omission")
		row.Set(schedulingstore.OmissionReasonField, "planned_unavailability")
		row.Set(schedulingstore.StatusField, string(domain.CancelledState))
		return
	}
	row.Set(schedulingstore.ScheduleStateField, string(domain.ScheduledState))
	row.Set(schedulingstore.BillingOutcomeField, "billable")
	row.Set(schedulingstore.OmissionReasonField, "")
	row.Set(schedulingstore.StatusField, string(domain.ScheduledState))
}

func reconcileAffectedContracts(app core.App, contracts map[string]bool, now time.Time) error {
	for contractID := range contracts {
		if contractID == "" {
			continue
		}
		if err := ledgerapi.ReconcileContractMonthsInTransaction(app, contractID, now); err != nil {
			return err
		}
	}
	return nil
}

func saveAvailabilityEvents(app core.App, teacherID string, mutation AvailabilityMutation, result CommitResult, now time.Time) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		return err
	}
	assignments, err := recordsByField(app, schedulingstore.TeacherLearnersCollectionName, "teacher", teacherID)
	if err != nil {
		return err
	}
	aggregateID := availabilityAggregateID(mutation, result)
	for _, assignment := range assignments {
		event := core.NewRecord(collection)
		event.Set(schedulingstore.AssignmentField, assignment.Id)
		event.Set(schedulingstore.AggregateTypeField, "availability")
		event.Set(schedulingstore.AggregateIDField, aggregateID)
		event.Set(schedulingstore.EventTypeField, "availability_changed")
		event.Set(schedulingstore.ActorRoleField, "teacher")
		event.Set(schedulingstore.ActorIDField, teacherID)
		event.Set(schedulingstore.EventAtField, now.UTC())
		if err := app.Save(event); err != nil {
			return err
		}
	}
	return nil
}

func availabilityAggregateID(mutation AvailabilityMutation, result CommitResult) string {
	if result.Rule != nil {
		return result.Rule.Id
	}
	if result.Exception != nil {
		return result.Exception.Id
	}
	return mutation.ID
}

func saveDistantEffectEvent(app core.App, lesson *core.Record, effect availabilityimpact.DistantEffect, now time.Time) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		return err
	}
	event := core.NewRecord(collection)
	event.Set(schedulingstore.AssignmentField, lesson.GetString(schedulingstore.AssignmentField))
	event.Set(schedulingstore.AggregateTypeField, "lesson")
	event.Set(schedulingstore.AggregateIDField, lesson.Id)
	event.Set(schedulingstore.EventTypeField, map[availabilityimpact.DistantAction]string{availabilityimpact.Omit: "contract_occurrence_omitted", availabilityimpact.Restore: "contract_occurrence_restored"}[effect.Effect])
	event.Set(schedulingstore.ActorRoleField, "system")
	event.Set(schedulingstore.EventAtField, now.UTC())
	event.Set(schedulingstore.NewStateField, map[string]any{"schedule_state": lesson.GetString(schedulingstore.ScheduleStateField), "billing_outcome": lesson.GetString(schedulingstore.BillingOutcomeField)})
	return app.Save(event)
}
