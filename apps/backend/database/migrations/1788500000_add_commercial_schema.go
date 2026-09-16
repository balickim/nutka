// Package migrations adds the commercial lesson projections and migrates existing scheduling data.
// The migration keeps concrete timestamps in UTC and stores teacher-local business dates as text.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

const (
	commercialPolicyVersion = businesspolicy.CurrentVersion
	commercialCurrency      = businesspolicy.CurrencyPLN
	commercialLessonPrice   = businesspolicy.AdHocPriceMinor
	commercialLessonMinutes = businesspolicy.LessonDurationMinutes
)

const (
	datePattern       = `^\d{4}-\d{2}-\d{2}$`
	monthPattern      = `^\d{4}-\d{2}$`
	monthStartPattern = `^\d{4}-\d{2}-01$`
	clockPattern      = `^(?:[01][0-9]|2[0-3]):(?:00|15|30|45)$`
	currencyPattern   = `^[A-Z]{3}$`
)

var commercialLessonFields = []string{
	schedulingstore.PlanTypeField,
	schedulingstore.PackageTokenField,
	schedulingstore.ContractField,
	schedulingstore.OriginalLocalDateField,
	schedulingstore.OriginalStartAtField,
	schedulingstore.PolicyVersionField,
	schedulingstore.PolicySnapshotField,
	schedulingstore.UnitPriceMinorField,
	schedulingstore.CurrencyField,
	schedulingstore.ScheduleStateField,
	schedulingstore.OutcomeField,
	schedulingstore.OutcomeAtField,
	schedulingstore.BillingOutcomeField,
	schedulingstore.IndividuallyRescheduledField,
	schedulingstore.OmissionReasonField,
}

func init() {
	migrations.Register(migrateCommercialSchema, revertCommercialSchema)
}

func migrateCommercialSchema(app core.App) error {
	if err := businesspolicy.ValidateCurrent(); err != nil {
		return fmt.Errorf("validate business policy for migration: %w", err)
	}
	assignments, lessons, err := commercialBaseCollections(app)
	if err != nil {
		return err
	}
	if err := extendCommercialSchema(app, assignments, lessons); err != nil {
		return err
	}
	return backfillCommercialSchema(app, assignments, lessons)
}

func commercialBaseCollections(app core.App) (*core.Collection, *core.Collection, error) {
	assignments, err := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	if err != nil {
		return nil, nil, fmt.Errorf("find assignments collection: %w", err)
	}
	lessons, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		return nil, nil, fmt.Errorf("find lessons collection: %w", err)
	}
	return assignments, lessons, nil
}

func extendCommercialSchema(app core.App, assignments, lessons *core.Collection) error {
	if err := createCommercialCollections(app, assignments, lessons); err != nil {
		return err
	}
	if err := addAvailabilityExceptionState(app); err != nil {
		return err
	}
	tokens, err := app.FindCollectionByNameOrId(schedulingstore.PackageTokensCollectionName)
	if err != nil {
		return err
	}
	contracts, err := app.FindCollectionByNameOrId(schedulingstore.RegularContractsCollectionName)
	if err != nil {
		return err
	}
	if err := ensureCommercialLessonFields(app, lessons, tokens.Id, contracts.Id); err != nil {
		return err
	}
	return nil
}

func backfillCommercialSchema(app core.App, assignments, lessons *core.Collection) error {
	if err := auditAndBackfillScheduling(app, assignments, lessons); err != nil {
		return err
	}
	if err := backfillBusinessEvents(app, lessons); err != nil {
		return err
	}
	if err := makeCommercialLessonFieldsRequired(app, lessons); err != nil {
		return err
	}
	return removeLegacyDurationField(app, assignments)
}

func addAvailabilityExceptionState(app core.App) error {
	exceptions, err := app.FindCollectionByNameOrId(schedulingstore.AvailabilityExceptionsCollectionName)
	if err != nil {
		return fmt.Errorf("find availability exceptions: %w", err)
	}
	if exceptions.Fields.GetByName(schedulingstore.EnabledField) == nil {
		exceptions.Fields.Add(&core.BoolField{Name: schedulingstore.EnabledField})
		if err := app.Save(exceptions); err != nil {
			return fmt.Errorf("add availability exception state: %w", err)
		}
	}
	rows, err := app.FindAllRecords(exceptions)
	if err != nil {
		return fmt.Errorf("read availability exceptions: %w", err)
	}
	for _, row := range rows {
		row.Set(schedulingstore.EnabledField, true)
		if err := app.Save(row); err != nil {
			return fmt.Errorf("enable availability exception %s: %w", row.Id, err)
		}
	}
	return nil
}
