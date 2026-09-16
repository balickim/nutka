// This file creates the paid package purchase decision and its four available tokens.
// The caller persists all returned records in one transaction.
package ledger

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
)

type PackagePurchaseCommand struct {
	Package *commercial.Package
	Actor   Actor
	At      time.Time
}

type PackagePurchaseDecision struct {
	PackageID    string
	AssignmentID string
	TokenCount   int
	Payment      FinancialEntry
}

// RecordPackagePurchase creates the paid entry for an already-created commercial package.
// The caller persists this entry with that package and its tokens in one transaction.
func RecordPackagePurchase(command PackagePurchaseCommand) (PackagePurchaseDecision, error) {
	if err := validatePackagePurchase(command); err != nil {
		return PackagePurchaseDecision{}, err
	}
	purchasedOn := command.Package.PurchasedOn.Format("2006-01-02")
	payment := FinancialEntry{AssignmentID: command.Package.Assignment.ID, EntryType: EntryPackagePurchase, AmountMinor: command.Package.Price.Minor, Currency: command.Package.Price.Currency, EffectiveOn: purchasedOn, Actor: command.Actor, EventAt: command.At.UTC()}
	return PackagePurchaseDecision{PackageID: command.Package.ID, AssignmentID: command.Package.Assignment.ID, TokenCount: len(command.Package.Tokens), Payment: payment}, nil
}

func validatePackagePurchase(command PackagePurchaseCommand) error {
	if command.Actor.Role != TeacherActor {
		return ErrTeacherRequired
	}
	if command.Package == nil || command.Package.ID == "" || command.Package.Assignment.ID == "" {
		return ErrInvalidDate
	}
	if command.Actor.ID == "" || command.Actor.ID != command.Package.Assignment.TeacherID {
		return ErrTeacherRequired
	}
	return validatePackagePurchasePolicy(command)
}

func validatePackagePurchasePolicy(command PackagePurchaseCommand) error {
	policy := command.Package.Policy
	if policy.Version == "" || command.At.IsZero() {
		return ErrInvalidDate
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	if command.Package.Price.Minor != policy.PackagePriceMinor || command.Package.Price.Currency != policy.Currency {
		return ErrCurrencyMismatch
	}
	if len(command.Package.Tokens) != policy.PackageTokenCount {
		return ErrInvalidTokenCount
	}
	return nil
}
