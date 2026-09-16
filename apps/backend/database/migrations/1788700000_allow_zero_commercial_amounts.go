// Package migrations aligns legacy commercial amount fields with zero-valued ledger outcomes.
// The repair changes validation metadata and preserves every stored amount.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(allowZeroCommercialAmounts, func(core.App) error { return nil })
}

func allowZeroCommercialAmounts(app core.App) error {
	fields := []struct {
		collection string
		name       string
	}{
		{schedulingstore.ChargesCollectionName, schedulingstore.OriginalAmountMinorField},
		{schedulingstore.ChargesCollectionName, schedulingstore.CurrentAmountMinorField},
		{schedulingstore.ContractMonthsCollectionName, schedulingstore.ForecastAmountMinorField},
		{schedulingstore.FinancialEntriesCollectionName, schedulingstore.AmountMinorField},
	}
	for _, definition := range fields {
		collection, err := app.FindCollectionByNameOrId(definition.collection)
		if err != nil {
			return fmt.Errorf("find %s collection for zero amount repair: %w", definition.collection, err)
		}
		field, ok := collection.Fields.GetByName(definition.name).(*core.NumberField)
		if !ok {
			return fmt.Errorf("%s.%s must be a number field", definition.collection, definition.name)
		}
		field.Required = false
		if err := app.Save(collection); err != nil {
			return fmt.Errorf("allow zero in %s.%s: %w", definition.collection, definition.name, err)
		}
	}
	return nil
}
