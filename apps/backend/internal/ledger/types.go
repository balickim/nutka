// Package ledger models manual commercial settlement and immutable financial history.
// It has no persistence, transport, clock, or payment-provider dependencies.
package ledger

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidMoney            = errors.New("money must use a three-letter currency")
	ErrInvalidState            = errors.New("invalid settlement state")
	ErrInvalidTransition       = errors.New("settlement transition is not allowed")
	ErrTeacherRequired         = errors.New("a teacher actor is required")
	ErrLessonNotEnded          = errors.New("ad hoc settlement requires an ended lesson")
	ErrInvalidDate             = errors.New("date must use YYYY-MM-DD")
	ErrFuturePurchaseDate      = errors.New("purchase date cannot be in the future")
	ErrInvalidReason           = errors.New("correction reason is required")
	ErrNegativeAmount          = errors.New("amount cannot be negative")
	ErrInsufficientCredit      = errors.New("credit is insufficient")
	ErrCurrencyMismatch        = errors.New("currency must match the charge")
	ErrAssignmentMismatch      = errors.New("financial record belongs to another assignment")
	ErrAdjustmentExceedsCharge = errors.New("adjustment exceeds charge amount")
	ErrInvalidTokenCount       = errors.New("package token count does not match policy")
	ErrInvalidOccurrence       = errors.New("occurrence snapshot is invalid")
	ErrInvalidDisposition      = errors.New("occurrence disposition is invalid")
)

// Money stores an integer amount in the smallest currency unit.
type Money struct {
	MinorUnits int64
	Currency   string
}

func NewMoney(minorUnits int64, currency string) (Money, error) {
	value := Money{MinorUnits: minorUnits, Currency: strings.ToUpper(strings.TrimSpace(currency))}
	if value.MinorUnits < 0 || len(value.Currency) != 3 {
		return Money{}, ErrInvalidMoney
	}
	for _, character := range value.Currency {
		if character < 'A' || character > 'Z' {
			return Money{}, ErrInvalidMoney
		}
	}
	return value, nil
}

// Actor identifies the account or system that caused a financial transition.
type Actor struct {
	Role string
	ID   string
}

const (
	TeacherActor = "teacher"
	LearnerActor = "learner"
	SystemActor  = "system"
)

type SettlementState string

// PaymentState is the contract-charge name for the shared settlement states.
type PaymentState = SettlementState

const (
	PendingSettlement   SettlementState = "pending_settlement"
	PendingPayment      SettlementState = "pending"
	Paid                SettlementState = "paid"
	IntentionallyUnpaid SettlementState = "intentionally_unpaid"
	NotApplicable       SettlementState = "not_applicable"
	Overdue             SettlementState = "overdue"
)

const (
	PaymentPending             PaymentState = PendingPayment
	PaymentPaid                PaymentState = Paid
	PaymentIntentionallyUnpaid PaymentState = IntentionallyUnpaid
	PaymentNotApplicable       PaymentState = NotApplicable
	PaymentOverdue             PaymentState = Overdue
)

func (state SettlementState) Valid() bool {
	return state == PendingSettlement || state == PendingPayment || state == Paid || state == IntentionallyUnpaid || state == NotApplicable
}

func ValidChargeState(state SettlementState) bool {
	return state == PendingPayment || state == Paid || state == IntentionallyUnpaid
}

func ValidAdHocState(state SettlementState) bool {
	return state == PendingSettlement || state == Paid || state == IntentionallyUnpaid || state == NotApplicable
}

type EntryType string

const (
	EntryChargeCreated    EntryType = "charge_created"
	EntryPackagePurchase  EntryType = "package_purchase"
	EntrySettlementPaid   EntryType = "settlement_paid"
	EntrySettlementUnpaid EntryType = "settlement_unpaid"
	EntrySettlementNAA    EntryType = "settlement_not_applicable"
	EntryAdjustment       EntryType = "adjustment"
	EntryCreditCreated    EntryType = "credit_created"
	EntryCreditApplied    EntryType = "credit_applied"
	EntryRefund           EntryType = "refund"
	EntryCorrection       EntryType = "correction"
)

// FinancialEntry is append-only. AmountMinor is signed for adjustments.
type FinancialEntry struct {
	ID              string
	AssignmentID    string
	ChargeID        string
	EntryType       EntryType
	AmountMinor     int64
	Currency        string
	EffectiveOn     string
	RelatedLessonID string
	RelatedEntryID  string
	Reason          string
	Actor           Actor
	EventAt         time.Time
}

// Charge is the mutable read projection of an append-only financial stream.
type Charge struct {
	ID                  string
	AssignmentID        string
	SourceType          string
	SourceID            string
	Period              string
	OriginalAmountMinor int64
	CurrentAmountMinor  int64
	Currency            string
	SettlementState     SettlementState
	DueOn               string
	PaidAt              time.Time
	CreatedAt           time.Time
}

func (charge Charge) Valid() bool {
	return charge.ID != "" && charge.AssignmentID != "" && validCurrency(charge.Currency) && charge.OriginalAmountMinor >= 0 && charge.CurrentAmountMinor >= 0 && charge.CurrentAmountMinor <= charge.OriginalAmountMinor && ValidChargeState(charge.SettlementState)
}

// IsOverdue derives overdue presentation without storing an irreversible state.
func (charge Charge) IsOverdue(now time.Time, location *time.Location) bool {
	if charge.SettlementState == Paid || charge.SettlementState == NotApplicable || charge.DueOn == "" || location == nil {
		return false
	}
	due, err := time.ParseInLocation("2006-01-02", charge.DueOn, location)
	if err != nil {
		return false
	}
	return !now.In(location).Before(due.AddDate(0, 0, 1))
}

// DerivedState returns overdue only for an unpaid charge after its local due date.
func (charge Charge) DerivedState(now time.Time, location *time.Location) SettlementState {
	if charge.IsOverdue(now, location) {
		return Overdue
	}
	return charge.SettlementState
}

// OverdueView is a read-model value suitable for API responses.
type OverdueView struct {
	Charge Charge
	Value  bool
}

type Credit struct {
	ID              string
	AssignmentID    string
	AmountMinor     int64
	RemainingMinor  int64
	Currency        string
	SourceChargeID  string
	CreatedAt       time.Time
	RefundedAt      time.Time
	RefundReference string
}

func (credit Credit) Available() bool { return credit.RemainingMinor > 0 && credit.RefundedAt.IsZero() }

type Refund struct {
	ID             string
	AssignmentID   string
	CreditID       string
	ChargeID       string
	AmountMinor    int64
	Currency       string
	Reason         string
	Actor          Actor
	RecordedAt     time.Time
	RelatedEntryID string
}

type Correction struct {
	ID              string
	AssignmentID    string
	CorrectsEntryID string
	AmountMinor     int64
	Currency        string
	Reason          string
	Actor           Actor
	RecordedAt      time.Time
}

func ValidateReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrInvalidReason
	}
	return nil
}

func parseDate(value string, location *time.Location) (time.Time, error) {
	if location == nil {
		return time.Time{}, ErrInvalidDate
	}
	date, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil || date.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidDate
	}
	return date, nil
}
