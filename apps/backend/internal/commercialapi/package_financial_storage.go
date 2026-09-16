// This file persists package charges, refunds, and invalidated token state.
package commercialapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func savePayment(app core.App, entry ledger.FinancialEntry, packageID string) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set(schedulingstore.AssignmentField, entry.AssignmentID)
	record.Set(schedulingstore.EntryTypeField, string(entry.EntryType))
	record.Set(schedulingstore.AmountMinorField, entry.AmountMinor)
	record.Set(schedulingstore.CurrencyField, entry.Currency)
	record.Set(schedulingstore.SourceTypeField, "package")
	record.Set(schedulingstore.SourceIDField, packageID)
	record.Set(schedulingstore.EffectiveOnField, entry.EffectiveOn)
	record.Set(schedulingstore.RelatedPackageField, packageID)
	record.Set(schedulingstore.ActorRoleField, entry.Actor.Role)
	record.Set(schedulingstore.ActorIDField, entry.Actor.ID)
	record.Set(schedulingstore.EventAtField, entry.EventAt.Format(time.RFC3339))
	return app.Save(record)
}

func savePackageRefund(app core.App, packageRecord *core.Record, refund refundInput, reason, teacherID string, now time.Time) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set(schedulingstore.AssignmentField, packageRecord.GetString(schedulingstore.AssignmentField))
	record.Set(schedulingstore.EntryTypeField, string(ledger.EntryRefund))
	record.Set(schedulingstore.AmountMinorField, refund.AmountMinor)
	record.Set(schedulingstore.CurrencyField, refund.Currency)
	record.Set(schedulingstore.SourceTypeField, "package")
	record.Set(schedulingstore.SourceIDField, packageRecord.Id)
	record.Set(schedulingstore.EffectiveOnField, now.UTC().Format("2006-01-02"))
	record.Set(schedulingstore.RelatedPackageField, packageRecord.Id)
	record.Set(schedulingstore.ActorRoleField, string(history.TeacherActor))
	record.Set(schedulingstore.ActorIDField, teacherID)
	record.Set(schedulingstore.ReasonField, reason+": "+refund.Note)
	record.Set(schedulingstore.EventAtField, now.UTC())
	return app.Save(record)
}

func invalidateTokens(app core.App, value commercial.Package, now time.Time) error {
	for _, token := range value.Tokens {
		if token.State != commercial.TokenInvalidated {
			continue
		}
		row, err := app.FindRecordById(schedulingstore.PackageTokensCollectionName, token.ID)
		if err != nil {
			return err
		}
		row.Set(schedulingstore.TokenStateField, string(token.State))
		row.Set(schedulingstore.ChangedAtField, now.Format(time.RFC3339))
		if err := app.Save(row); err != nil {
			return err
		}
	}
	return nil
}
