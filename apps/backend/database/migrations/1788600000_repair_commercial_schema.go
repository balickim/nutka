// Package migrations repairs commercial databases created by an earlier schema revision.
// The repair reuses current schema builders and preserves existing commercial records.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(repairCommercialSchema, func(core.App) error { return nil })
}

func repairCommercialSchema(app core.App) error {
	if err := normalizeLegacyCommercialValues(app); err != nil {
		return err
	}
	assignments, lessons, err := commercialBaseCollections(app)
	if err != nil {
		return err
	}
	if err := extendCommercialSchema(app, assignments, lessons); err != nil {
		return fmt.Errorf("extend repaired commercial schema: %w", err)
	}
	if err := synchronizeCommercialSelectFields(app); err != nil {
		return err
	}
	return backfillLegacyFinancialEntries(app)
}

func normalizeLegacyCommercialValues(app core.App) error {
	updates := []struct {
		collection string
		field      string
		legacy     string
		canonical  string
	}{
		{schedulingstore.LessonPackagesCollectionName, schedulingstore.PackageStatusField, "expired", string(commercial.PackageClosed)},
		{schedulingstore.RegularContractsCollectionName, schedulingstore.StatusField, "cancelled", string(regularcontract.Ended)},
		{schedulingstore.FinancialEntriesCollectionName, schedulingstore.EntryTypeField, "creation", string(ledger.EntryChargeCreated)},
		{schedulingstore.FinancialEntriesCollectionName, schedulingstore.EntryTypeField, "payment", string(ledger.EntrySettlementPaid)},
		{schedulingstore.FinancialEntriesCollectionName, schedulingstore.EntryTypeField, "credit", string(ledger.EntryCreditCreated)},
	}
	for _, update := range updates {
		query := fmt.Sprintf("UPDATE {{%s}} SET [[%s]] = {:canonical} WHERE [[%s]] = {:legacy}", update.collection, update.field, update.field)
		if _, err := app.DB().NewQuery(query).Bind(dbx.Params{"canonical": update.canonical, "legacy": update.legacy}).Execute(); err != nil {
			return fmt.Errorf("normalize %s.%s value %q: %w", update.collection, update.field, update.legacy, err)
		}
	}
	return nil
}

func synchronizeCommercialSelectFields(app core.App) error {
	definitions := []struct {
		collection string
		field      string
		values     []string
	}{
		{schedulingstore.LessonPackagesCollectionName, schedulingstore.PackageStatusField, []string{string(commercial.PackageOpen), string(commercial.PackageClosed)}},
		{schedulingstore.RegularContractsCollectionName, schedulingstore.StatusField, []string{string(regularcontract.Active), string(regularcontract.NoticeGiven), string(regularcontract.Ended)}},
		{schedulingstore.ChargesCollectionName, schedulingstore.SettlementStateField, []string{"pending", "pending_settlement", "paid", "intentionally_unpaid", "not_applicable"}},
		{schedulingstore.FinancialEntriesCollectionName, schedulingstore.EntryTypeField, []string{"charge_created", "package_purchase", "settlement_paid", "settlement_unpaid", "settlement_not_applicable", "adjustment", "credit_created", "credit_applied", "refund", "correction"}},
	}
	for _, definition := range definitions {
		collection, err := app.FindCollectionByNameOrId(definition.collection)
		if err != nil {
			return fmt.Errorf("find %s collection for repair: %w", definition.collection, err)
		}
		field, ok := collection.Fields.GetByName(definition.field).(*core.SelectField)
		if !ok {
			return fmt.Errorf("%s.%s must be a select field", definition.collection, definition.field)
		}
		field.Values = definition.values
		if err := app.Save(collection); err != nil {
			return fmt.Errorf("save repaired %s.%s values: %w", definition.collection, definition.field, err)
		}
	}
	return nil
}

func backfillLegacyFinancialEntries(app core.App) error {
	entries, err := app.FindAllRecords(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return fmt.Errorf("read financial entries for repair: %w", err)
	}
	for _, entry := range entries {
		if entry.GetString(schedulingstore.AssignmentField) != "" {
			continue
		}
		assignmentID, sourceType, sourceID, err := legacyFinancialEntryOwnership(app, entry)
		if err != nil {
			return fmt.Errorf("repair financial entry %s: %w", entry.Id, err)
		}
		entry.Set(schedulingstore.AssignmentField, assignmentID)
		entry.Set(schedulingstore.SourceTypeField, sourceType)
		entry.Set(schedulingstore.SourceIDField, sourceID)
		entry.Set(schedulingstore.ActorRoleField, "system")
		if err := app.Save(entry); err != nil {
			return fmt.Errorf("save repaired financial entry %s: %w", entry.Id, err)
		}
	}
	return nil
}

func legacyFinancialEntryOwnership(app core.App, entry *core.Record) (string, string, string, error) {
	if chargeID := entry.GetString(schedulingstore.ChargeField); chargeID != "" {
		charge, err := app.FindRecordById(schedulingstore.ChargesCollectionName, chargeID)
		if err != nil {
			return "", "", "", fmt.Errorf("find charge %s: %w", chargeID, err)
		}
		return charge.GetString(schedulingstore.AssignmentField), charge.GetString(schedulingstore.SourceTypeField), charge.GetString(schedulingstore.SourceIDField), nil
	}
	if lessonID := entry.GetString(schedulingstore.RelatedLessonField); lessonID != "" {
		lesson, err := app.FindRecordById(schedulingstore.LessonsCollectionName, lessonID)
		if err != nil {
			return "", "", "", fmt.Errorf("find lesson %s: %w", lessonID, err)
		}
		return lesson.GetString(schedulingstore.AssignmentField), lesson.GetString(schedulingstore.PlanTypeField), lesson.Id, nil
	}
	return "", "", "", fmt.Errorf("cannot derive assignment")
}
