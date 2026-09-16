// This file exposes contract aggregate loading and persistence to transaction-owning lifecycle services.
package regularcontractapi

import (
	"context"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/ledgerapi"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

// LoadContract loads one complete contract aggregate from the caller transaction.
func LoadContract(app core.App, id string) (regularcontract.RegularContract, error) {
	row, err := app.FindRecordById(schedulingstore.RegularContractsCollectionName, id)
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	return contractFromRecord(app, row)
}

// SaveContract persists one complete aggregate in the caller transaction.
func SaveContract(app core.App, value regularcontract.RegularContract) error {
	if err := saveContract(app, context.Background(), value); err != nil {
		return err
	}
	now := latestContractEventAt(value)
	if now.IsZero() {
		return nil
	}
	return ledgerapi.ReconcileContractMonthsInTransaction(app, value.ID, now)
}

func latestContractEventAt(value regularcontract.RegularContract) time.Time {
	var result time.Time
	for _, event := range value.Events {
		if event.At.After(result) {
			result = event.At
		}
	}
	return result.UTC()
}
