// Package migrations defines commercial financial, business event, and lesson projection fields.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func createEntriesCollection(app core.App, assignments, lessons, packages, tokens, charges *core.Collection) error {
	entries, err := ensureCollection(app, schedulingstore.FinancialEntriesCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true},
		&core.RelationField{Name: schedulingstore.ChargeField, CollectionId: charges.Id},
		&core.SelectField{Name: schedulingstore.EntryTypeField, Values: []string{"charge_created", "package_purchase", "settlement_paid", "settlement_unpaid", "settlement_not_applicable", "adjustment", "credit_created", "credit_applied", "refund", "correction"}, Required: true},
		&core.NumberField{Name: schedulingstore.AmountMinorField, OnlyInt: true},
		&core.TextField{Name: schedulingstore.CurrencyField, Required: true, Pattern: currencyPattern},
		&core.SelectField{Name: schedulingstore.SourceTypeField, Values: []string{"ad_hoc", "regular_contract", "package"}},
		&core.TextField{Name: schedulingstore.SourceIDField, Max: 64},
		&core.TextField{Name: schedulingstore.EffectiveOnField, Pattern: datePattern},
		&core.RelationField{Name: schedulingstore.RelatedLessonField, CollectionId: lessons.Id},
		&core.RelationField{Name: schedulingstore.RelatedPackageField, CollectionId: packages.Id},
		&core.RelationField{Name: schedulingstore.RelatedTokenField, CollectionId: tokens.Id},
		&core.SelectField{Name: schedulingstore.ActorRoleField, Values: []string{"teacher", "learner", "system"}, Required: true},
		&core.TextField{Name: schedulingstore.ActorIDField, Max: 64},
		&core.TextField{Name: schedulingstore.ReasonField, Max: 1000},
		&core.DateField{Name: schedulingstore.EventAtField, Required: true},
	})
	if err != nil {
		return err
	}
	if err := app.Save(entries); err != nil {
		return fmt.Errorf("save %s collection: %w", entries.Name, err)
	}
	entries.AddIndex("idx_financial_entries_assignment_event_at", false, "`assignment`, `event_at`", "")
	entries.AddIndex("idx_financial_entries_source", false, "`assignment`, `source_type`, `source_id`, `event_at`", "`source_id` != ''")
	if entries.Fields.GetByName(schedulingstore.RelatedEntryField) == nil {
		entries.Fields.Add(&core.RelationField{Name: schedulingstore.RelatedEntryField, CollectionId: entries.Id})
	}
	if err := app.Save(entries); err != nil {
		return fmt.Errorf("save %s self relation: %w", entries.Name, err)
	}
	return nil
}

func createEventsCollection(app core.App, assignments *core.Collection) error {
	events, err := ensureCollection(app, schedulingstore.BusinessEventsCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true},
		&core.SelectField{Name: schedulingstore.AggregateTypeField, Values: []string{"lesson", "package", "package_token", "contract", "contract_amendment", "contract_month", "charge", "financial_entry", "availability"}, Required: true},
		&core.TextField{Name: schedulingstore.AggregateIDField, Required: true, Max: 64},
		&core.TextField{Name: schedulingstore.EventTypeField, Required: true, Max: 128},
		&core.SelectField{Name: schedulingstore.ActorRoleField, Values: []string{"teacher", "learner", "system"}, Required: true},
		&core.TextField{Name: schedulingstore.ActorIDField, Max: 64},
		&core.DateField{Name: schedulingstore.EventAtField, Required: true},
		&core.JSONField{Name: schedulingstore.RelatedIDsField, MaxSize: 16 * 1024},
		&core.JSONField{Name: schedulingstore.PriorStateField, MaxSize: 64 * 1024},
		&core.JSONField{Name: schedulingstore.NewStateField, MaxSize: 64 * 1024},
		&core.TextField{Name: schedulingstore.ReasonField, Max: 1000},
		&core.TextField{Name: schedulingstore.InternalNoteField, Max: 4000},
		&core.TextField{Name: schedulingstore.LegacyEventIDField, Max: 64},
	})
	if err != nil {
		return err
	}
	events.AddIndex("idx_business_events_assignment_event_at", false, "`assignment`, `event_at`", "")
	events.AddIndex("idx_business_events_aggregate", false, "`aggregate_type`, `aggregate_id`, `event_at`", "")
	events.AddIndex("idx_business_events_legacy_event", true, "`legacy_event_id`", "`legacy_event_id` != ''")
	if err := app.Save(events); err != nil {
		return fmt.Errorf("save %s collection: %w", events.Name, err)
	}
	if events.Fields.GetByName(schedulingstore.CorrectsEventField) == nil {
		events.Fields.Add(&core.RelationField{Name: schedulingstore.CorrectsEventField, CollectionId: events.Id})
	}
	if err := app.Save(events); err != nil {
		return fmt.Errorf("save %s self relation: %w", events.Name, err)
	}
	return nil
}

func ensureCommercialLessonFields(app core.App, lessons *core.Collection, tokenCollectionID, contractCollectionID string) error {
	fields := commercialLessonProjectionFields(tokenCollectionID, contractCollectionID)
	for _, field := range fields {
		if lessons.Fields.GetByName(field.GetName()) == nil {
			lessons.Fields.Add(field)
		}
	}
	if err := app.Save(lessons); err != nil {
		return fmt.Errorf("save lessons commercial fields: %w", err)
	}
	return nil
}

func commercialLessonProjectionFields(tokenCollectionID, contractCollectionID string) []core.Field {
	return []core.Field{
		&core.SelectField{Name: schedulingstore.PlanTypeField, Values: []string{"regular_contract", "package", "ad_hoc"}},
		&core.RelationField{Name: schedulingstore.PackageTokenField, CollectionId: tokenCollectionID},
		&core.RelationField{Name: schedulingstore.ContractField, CollectionId: contractCollectionID},
		&core.TextField{Name: schedulingstore.OriginalLocalDateField, Pattern: datePattern},
		&core.DateField{Name: schedulingstore.OriginalStartAtField},
		&core.TextField{Name: schedulingstore.PolicyVersionField, Max: 64},
		&core.JSONField{Name: schedulingstore.PolicySnapshotField, MaxSize: 16 * 1024},
		&core.NumberField{Name: schedulingstore.UnitPriceMinorField, OnlyInt: true, Min: floatPtr(0)},
		&core.TextField{Name: schedulingstore.CurrencyField, Pattern: currencyPattern},
		&core.SelectField{Name: schedulingstore.ScheduleStateField, Values: []string{"scheduled", "cancelled", "omitted"}},
		&core.SelectField{Name: schedulingstore.OutcomeField, Values: []string{"awaiting_outcome", "completed", "learner_no_show"}},
		&core.DateField{Name: schedulingstore.OutcomeAtField},
		&core.SelectField{Name: schedulingstore.BillingOutcomeField, Values: []string{"billable", "planned_omission", "free_cancellation", "teacher_cancellation", "late_cancellation", "exhausted_cancellation", "no_show", "contract_termination"}},
		&core.BoolField{Name: schedulingstore.IndividuallyRescheduledField},
		&core.TextField{Name: schedulingstore.OmissionReasonField, Max: 1000},
	}
}

func makeCommercialLessonFieldsRequired(app core.App, lessons *core.Collection) error {
	for _, name := range []string{schedulingstore.PlanTypeField, schedulingstore.OriginalLocalDateField, schedulingstore.PolicyVersionField, schedulingstore.PolicySnapshotField, schedulingstore.CurrencyField, schedulingstore.ScheduleStateField} {
		field := lessons.Fields.GetByName(name)
		switch value := field.(type) {
		case *core.SelectField:
			value.Required = true
		case *core.TextField:
			value.Required = true
		case *core.JSONField:
			value.Required = true
		case *core.NumberField:
			value.Required = true
		default:
			return fmt.Errorf("lessons.%s has an unexpected field type", name)
		}
	}
	lessons.AddIndex("idx_lessons_contract_occurrence", true, "`contract`, `original_local_date`", "`contract` != '' AND `original_local_date` != ''")
	return app.Save(lessons)
}
