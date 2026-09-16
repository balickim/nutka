// This file exposes package aggregate loading and projection persistence to transaction-owning application services.
package commercialapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

// PackageForToken loads the package that owns an assignment-scoped token.
func PackageForToken(app core.App, tokenID string) (*commercial.Package, error) {
	token, err := app.FindRecordById(schedulingstore.PackageTokensCollectionName, tokenID)
	if err != nil {
		return nil, errForbidden
	}
	row, err := app.FindRecordById(schedulingstore.LessonPackagesCollectionName, token.GetString(schedulingstore.PackageField))
	if err != nil {
		return nil, errForbidden
	}
	value, err := packageFromRecords(app, row)
	return &value, err
}

// SavePackageProjection writes package validity and token states in the caller transaction.
func SavePackageProjection(app core.App, value *commercial.Package, changedAt time.Time) error {
	if value == nil {
		return errInvalid
	}
	row, err := app.FindRecordById(schedulingstore.LessonPackagesCollectionName, value.ID)
	if err != nil {
		return err
	}
	row.Set(schedulingstore.PackageStatusField, string(value.Status))
	row.Set(schedulingstore.ValidThroughField, value.ValidThrough.Format("2006-01-02"))
	if !value.ClosedAt.IsZero() {
		row.Set(schedulingstore.ClosedAtField, value.ClosedAt.UTC())
	}
	if err := app.Save(row); err != nil {
		return err
	}
	for _, token := range value.Tokens {
		tokenRow, findErr := app.FindRecordById(schedulingstore.PackageTokensCollectionName, token.ID)
		if findErr != nil {
			return findErr
		}
		tokenRow.Set(schedulingstore.TokenStateField, string(token.State))
		tokenRow.Set(schedulingstore.LessonField, token.LessonID)
		tokenRow.Set(schedulingstore.ChangedAtField, changedAt.UTC())
		if err := app.Save(tokenRow); err != nil {
			return err
		}
	}
	return nil
}
