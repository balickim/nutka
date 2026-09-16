// Package migrations restores the pre-commercial scheduling schema.
// Rollback removes commercial children before parents and restores the legacy duration field.
package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func revertCommercialSchema(app core.App) error {
	if err := removeAvailabilityExceptionState(app); err != nil {
		return err
	}
	lessons, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err == nil {
		lessons.RemoveIndex("idx_lessons_contract_occurrence")
		for _, name := range commercialLessonFields {
			lessons.Fields.RemoveByName(name)
		}
		if err := app.Save(lessons); err != nil {
			return fmt.Errorf("remove commercial lesson fields: %w", err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err := deleteCommercialCollections(app); err != nil {
		return err
	}
	return restoreLegacyDurationField(app)
}

func removeAvailabilityExceptionState(app core.App) error {
	exceptions, err := app.FindCollectionByNameOrId(schedulingstore.AvailabilityExceptionsCollectionName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	exceptions.Fields.RemoveByName(schedulingstore.EnabledField)
	if err := app.Save(exceptions); err != nil {
		return fmt.Errorf("remove availability exception state: %w", err)
	}
	return nil
}

func deleteCommercialCollections(app core.App) error {
	for _, name := range []string{
		schedulingstore.BusinessEventsCollectionName,
		schedulingstore.FinancialEntriesCollectionName,
		schedulingstore.ContractMonthsCollectionName,
		schedulingstore.ChargesCollectionName,
		schedulingstore.ContractAmendmentsCollectionName,
		schedulingstore.RegularContractsCollectionName,
		schedulingstore.PackageTokensCollectionName,
		schedulingstore.LessonPackagesCollectionName,
	} {
		collection, err := app.FindCollectionByNameOrId(name)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if err := app.Delete(collection); err != nil {
			return fmt.Errorf("delete %s collection: %w", name, err)
		}
	}
	return nil
}

func restoreLegacyDurationField(app core.App) error {
	assignments, err := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	if err != nil {
		return err
	}
	if assignments.Fields.GetByName(schedulingstore.DefaultDurationMinutesField) == nil {
		assignments.Fields.Add(&core.NumberField{Name: schedulingstore.DefaultDurationMinutesField, Required: true, OnlyInt: true, Min: floatPtr(15)})
	}
	if err := app.Save(assignments); err != nil {
		return fmt.Errorf("restore assignment duration field: %w", err)
	}
	rows, err := app.FindAllRecords(assignments)
	if err != nil {
		return fmt.Errorf("read assignments for duration restoration: %w", err)
	}
	for _, assignment := range rows {
		assignment.Set(schedulingstore.DefaultDurationMinutesField, commercialLessonMinutes)
		if err := app.Save(assignment); err != nil {
			return fmt.Errorf("restore assignment %s duration: %w", assignment.Id, err)
		}
	}
	return nil
}
