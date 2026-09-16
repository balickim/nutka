// Package migrations defines the commercial PocketBase collections and lesson projections.
// Collection fields use explicit English names and indexes preserve assignment-scoped uniqueness.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func createCommercialCollections(app core.App, assignments, lessons *core.Collection) error {
	packages, err := createPackagesCollection(app, assignments)
	if err != nil {
		return err
	}
	tokens, err := createTokensCollection(app, packages, lessons)
	if err != nil {
		return err
	}
	contracts, err := createContractsCollection(app, assignments)
	if err != nil {
		return err
	}
	if err := createAmendmentsCollection(app, contracts); err != nil {
		return err
	}
	charges, err := createChargesCollection(app, assignments)
	if err != nil {
		return err
	}
	if err := createMonthsCollection(app, contracts, charges); err != nil {
		return err
	}
	if err := createEntriesCollection(app, assignments, lessons, packages, tokens, charges); err != nil {
		return err
	}
	return createEventsCollection(app, assignments)
}

func createPackagesCollection(app core.App, assignments *core.Collection) (*core.Collection, error) {
	packages, err := ensureCollection(app, schedulingstore.LessonPackagesCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true},
		&core.SelectField{Name: schedulingstore.PackageStatusField, Values: []string{string(commercial.PackageOpen), string(commercial.PackageClosed)}, Required: true},
		&core.TextField{Name: schedulingstore.PurchasedOnField, Required: true, Pattern: datePattern},
		&core.TextField{Name: schedulingstore.ValidThroughField, Required: true, Pattern: datePattern},
		&core.DateField{Name: schedulingstore.ClosedAtField},
		&core.NumberField{Name: schedulingstore.UnitPriceMinorField, Required: true, OnlyInt: true, Min: floatPtr(0)},
		&core.TextField{Name: schedulingstore.CurrencyField, Required: true, Pattern: currencyPattern},
		&core.TextField{Name: schedulingstore.PolicyVersionField, Required: true, Max: 64},
		&core.JSONField{Name: schedulingstore.PolicySnapshotField, Required: true, MaxSize: 16 * 1024},
	})
	if err != nil {
		return nil, err
	}
	packages.AddIndex("idx_lesson_packages_assignment_status", false, "`assignment`, `status`", "")
	if err := app.Save(packages); err != nil {
		return nil, fmt.Errorf("save %s collection: %w", packages.Name, err)
	}
	return packages, nil
}

func createTokensCollection(app core.App, packages, lessons *core.Collection) (*core.Collection, error) {
	tokens, err := ensureCollection(app, schedulingstore.PackageTokensCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.PackageField, CollectionId: packages.Id, Required: true, CascadeDelete: true},
		&core.NumberField{Name: schedulingstore.OrdinalField, Required: true, OnlyInt: true, Min: floatPtr(1), Max: floatPtr(float64(businesspolicy.PackageTokenCount))},
		&core.SelectField{Name: schedulingstore.TokenStateField, Values: []string{string(commercial.TokenAvailable), string(commercial.TokenReserved), string(commercial.TokenUsed), string(commercial.TokenExpired), string(commercial.TokenInvalidated)}, Required: true},
		&core.RelationField{Name: schedulingstore.LessonField, CollectionId: lessons.Id},
		&core.DateField{Name: schedulingstore.ChangedAtField, Required: true},
	})
	if err != nil {
		return nil, err
	}
	tokens.AddIndex("idx_package_tokens_package_ordinal", true, "`package`, `ordinal`", "")
	tokens.AddIndex("idx_package_tokens_active_lesson", true, "`lesson`", "`lesson` != ''")
	tokens.AddIndex("idx_package_tokens_package_state", false, "`package`, `state`", "")
	if err := app.Save(tokens); err != nil {
		return nil, fmt.Errorf("save %s collection: %w", tokens.Name, err)
	}
	return tokens, nil
}

func createContractsCollection(app core.App, assignments *core.Collection) (*core.Collection, error) {
	contracts, err := ensureCollection(app, schedulingstore.RegularContractsCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true},
		&core.SelectField{Name: schedulingstore.StatusField, Values: []string{string(regularcontract.Active), string(regularcontract.NoticeGiven), string(regularcontract.Ended)}, Required: true},
		&core.TextField{Name: schedulingstore.StartOnField, Required: true, Pattern: datePattern},
		&core.TextField{Name: schedulingstore.EndOnField, Required: true, Pattern: datePattern},
		&core.NumberField{Name: schedulingstore.WeekdayField, Required: true, OnlyInt: true, Min: floatPtr(0), Max: floatPtr(6)},
		&core.TextField{Name: schedulingstore.StartTimeField, Required: true, Pattern: clockPattern},
		&core.NumberField{Name: schedulingstore.UnitPriceMinorField, Required: true, OnlyInt: true, Min: floatPtr(0)},
		&core.TextField{Name: schedulingstore.CurrencyField, Required: true, Pattern: currencyPattern},
		&core.DateField{Name: schedulingstore.NoticeAtField},
		&core.TextField{Name: schedulingstore.EffectiveEndOnField, Pattern: datePattern},
		&core.TextField{Name: schedulingstore.PolicyVersionField, Required: true, Max: 64},
		&core.JSONField{Name: schedulingstore.PolicySnapshotField, Required: true, MaxSize: 16 * 1024},
	})
	if err != nil {
		return nil, err
	}
	contracts.AddIndex("idx_regular_contracts_assignment_status", false, "`assignment`, `status`", "")
	if err := app.Save(contracts); err != nil {
		return nil, fmt.Errorf("save %s collection: %w", contracts.Name, err)
	}
	return contracts, nil
}

func createAmendmentsCollection(app core.App, contracts *core.Collection) error {
	amendments, err := ensureCollection(app, schedulingstore.ContractAmendmentsCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.ContractField, CollectionId: contracts.Id, Required: true, CascadeDelete: true},
		&core.TextField{Name: schedulingstore.EffectiveOnField, Required: true, Pattern: monthStartPattern},
		&core.NumberField{Name: schedulingstore.UnitPriceMinorField, Required: true, OnlyInt: true, Min: floatPtr(0)},
		&core.TextField{Name: schedulingstore.CurrencyField, Required: true, Pattern: currencyPattern},
		&core.TextField{Name: schedulingstore.PolicyVersionField, Required: true, Max: 64},
		&core.JSONField{Name: schedulingstore.PolicySnapshotField, Required: true, MaxSize: 16 * 1024},
		&core.DateField{Name: schedulingstore.CreatedAtField, Required: true},
	})
	if err != nil {
		return err
	}
	amendments.AddIndex("idx_contract_amendments_contract_effective", true, "`contract`, `effective_on`", "")
	if err := app.Save(amendments); err != nil {
		return fmt.Errorf("save %s collection: %w", amendments.Name, err)
	}
	return nil
}

func createChargesCollection(app core.App, assignments *core.Collection) (*core.Collection, error) {
	charges, err := ensureCollection(app, schedulingstore.ChargesCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.AssignmentField, CollectionId: assignments.Id, Required: true, CascadeDelete: true},
		&core.SelectField{Name: schedulingstore.SourceTypeField, Values: []string{"ad_hoc", "regular_contract", "package"}, Required: true},
		&core.TextField{Name: schedulingstore.SourceIDField, Required: true, Max: 64},
		&core.TextField{Name: schedulingstore.PeriodField, Pattern: monthPattern},
		&core.NumberField{Name: schedulingstore.OriginalAmountMinorField, OnlyInt: true, Min: floatPtr(0)},
		&core.NumberField{Name: schedulingstore.CurrentAmountMinorField, OnlyInt: true, Min: floatPtr(0)},
		&core.TextField{Name: schedulingstore.CurrencyField, Required: true, Pattern: currencyPattern},
		&core.SelectField{Name: schedulingstore.SettlementStateField, Values: []string{"pending", "pending_settlement", "paid", "intentionally_unpaid", "not_applicable"}, Required: true},
		&core.TextField{Name: schedulingstore.DueOnField, Pattern: datePattern},
		&core.DateField{Name: schedulingstore.PaidAtField},
		&core.DateField{Name: schedulingstore.CreatedAtField},
	})
	if err != nil {
		return nil, err
	}
	charges.AddIndex("idx_charges_assignment_settlement", false, "`assignment`, `settlement_state`", "")
	charges.AddIndex("idx_charges_contract_source_period", true, "`assignment`, `source_type`, `source_id`, `period`", "`source_type` = 'regular_contract' AND `source_id` != '' AND `period` != ''")
	charges.AddIndex("idx_charges_source", true, "`assignment`, `source_type`, `source_id`", "`source_type` != 'regular_contract' AND `source_id` != ''")
	if err := app.Save(charges); err != nil {
		return nil, fmt.Errorf("save %s collection: %w", charges.Name, err)
	}
	return charges, nil
}

func createMonthsCollection(app core.App, contracts, charges *core.Collection) error {
	months, err := ensureCollection(app, schedulingstore.ContractMonthsCollectionName, []core.Field{
		&core.RelationField{Name: schedulingstore.ContractField, CollectionId: contracts.Id, Required: true, CascadeDelete: true},
		&core.TextField{Name: schedulingstore.MonthField, Required: true, Pattern: monthPattern},
		&core.NumberField{Name: schedulingstore.ForecastAmountMinorField, OnlyInt: true, Min: floatPtr(0)},
		&core.RelationField{Name: schedulingstore.ChargeField, CollectionId: charges.Id},
		&core.DateField{Name: schedulingstore.GeneratedAtField, Required: true},
	})
	if err != nil {
		return err
	}
	months.AddIndex("idx_contract_months_contract_month", true, "`contract`, `month`", "")
	if err := app.Save(months); err != nil {
		return fmt.Errorf("save %s collection: %w", months.Name, err)
	}
	return nil
}
