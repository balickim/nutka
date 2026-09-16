package ledger

import (
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
)

func ledgerTime(year int, month time.Month, day, hour int) time.Time {
	return time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
}

func occurrences(month string, dispositions ...OccurrenceDisposition) []OccurrenceSnapshot {
	values := make([]OccurrenceSnapshot, len(dispositions))
	for index, disposition := range dispositions {
		values[index] = OccurrenceSnapshot{ID: month + string(rune('a'+index)), ContractID: "contract", AssignmentID: "assignment", Month: month, UnitPriceMinor: 5000, Currency: "PLN", Disposition: disposition}
	}
	return values
}

func TestForecastIncludesFiveOrFourBillableOccurrences(t *testing.T) {
	five, err := ForecastContractMonth("contract", "assignment", "2026-09", occurrences("2026-09", BillableOccurrence, BillableOccurrence, BillableOccurrence, BillableOccurrence, NoShow))
	if err != nil || five.BillableCount != 5 || five.ForecastAmountMinor != 25000 {
		t.Fatalf("five billable occurrences must total 250 PLN: %#v %v", five, err)
	}
	four, err := ForecastContractMonth("contract", "assignment", "2026-09", occurrences("2026-09", BillableOccurrence, BillableOccurrence, BillableOccurrence, BillableOccurrence, PlannedOmission))
	if err != nil || four.BillableCount != 4 || four.ForecastAmountMinor != 20000 {
		t.Fatalf("omission and free cancellation must not remain billable: %#v %v", four, err)
	}
}

func TestForecastRejectsInvalidOccurrenceSnapshot(t *testing.T) {
	base := OccurrenceSnapshot{ID: "occurrence", ContractID: "contract", AssignmentID: "assignment", Month: "2026-09", UnitPriceMinor: 5000, Currency: "PLN", Disposition: BillableOccurrence}
	for _, invalid := range []OccurrenceSnapshot{
		base,
		{ID: "occurrence", ContractID: "contract", AssignmentID: "assignment", Month: "2026-09", UnitPriceMinor: 0, Currency: "PLN", Disposition: BillableOccurrence},
		{ID: "occurrence", ContractID: "contract", AssignmentID: "assignment", Month: "2026-09", UnitPriceMinor: 5000, Currency: "", Disposition: BillableOccurrence},
		{ID: "occurrence", ContractID: "contract", AssignmentID: "assignment", Month: "2026-09", UnitPriceMinor: 5000, Currency: "PLN", Disposition: ""},
	} {
		if invalid == base {
			continue
		}
		if _, err := ForecastContractMonth("contract", "assignment", "2026-09", []OccurrenceSnapshot{invalid}); err == nil {
			t.Fatalf("invalid snapshot must be rejected: %#v", invalid)
		}
	}
	if _, err := ForecastContractMonth("contract", "assignment", "2026-09", []OccurrenceSnapshot{base}); err != nil {
		t.Fatalf("valid snapshot must remain supported: %v", err)
	}
}

func TestMidMonthReconciliationCreatesOneChargeAndIsIdempotent(t *testing.T) {
	location := time.FixedZone("UTC", 0)
	input := MonthReconciliationInput{ContractID: "contract", AssignmentID: "assignment", Month: "2026-09", ActivationDate: "2026-09-12", Now: ledgerTime(2026, time.September, 12, 10), Location: location, Occurrences: occurrences("2026-09", BillableOccurrence, BillableOccurrence)}
	first, err := ReconcileContractMonth(input)
	if err != nil || !first.Created || first.Charge == nil || first.Charge.CurrentAmountMinor != 10000 {
		t.Fatalf("mid-month activation must create current charge: %#v %v", first, err)
	}
	input.Charge = first.Charge
	second, err := ReconcileContractMonth(input)
	if err != nil || second.Created || second.Charge == nil || second.Charge.ID != first.Charge.ID {
		t.Fatalf("reconciliation must be idempotent: %#v %v", second, err)
	}
}

func TestReconciliationDoesNotChargeBeforeContractActivation(t *testing.T) {
	location := time.FixedZone("UTC", 0)
	input := MonthReconciliationInput{ContractID: "contract", AssignmentID: "assignment", Month: "2026-09", ActivationDate: "2026-10-12", Now: ledgerTime(2026, time.September, 12, 10), Location: location, Currency: "PLN", Occurrences: occurrences("2026-09", BillableOccurrence)}
	decision, err := ReconcileContractMonth(input)
	if err != nil || decision.Payable || decision.Charge != nil {
		t.Fatalf("a month before activation must remain a forecast: %#v %v", decision, err)
	}
}

func TestOverdueIsDerivedAfterTeacherLocalDayFive(t *testing.T) {
	charge := Charge{CurrentAmountMinor: 5000, OriginalAmountMinor: 5000, SettlementState: PendingPayment, DueOn: "2026-09-05"}
	location := time.FixedZone("UTC", 0)
	if charge.IsOverdue(ledgerTime(2026, time.September, 5, 23), location) {
		t.Fatal("charge must not be overdue during day five")
	}
	if !charge.IsOverdue(ledgerTime(2026, time.September, 6, 0), location) {
		t.Fatal("charge must be overdue after day five")
	}
	charge.SettlementState = Paid
	if charge.IsOverdue(ledgerTime(2026, time.September, 7, 0), location) {
		t.Fatal("paid charge cannot be overdue")
	}
}

func TestChargeRejectsDerivedOrAdHocSettlementStates(t *testing.T) {
	base := Charge{ID: "charge", AssignmentID: "assignment", OriginalAmountMinor: 5000, CurrentAmountMinor: 5000, Currency: "PLN", SettlementState: PendingPayment}
	if !base.Valid() {
		t.Fatal("pending charge should be valid")
	}
	base.SettlementState = Overdue
	if base.Valid() {
		t.Fatal("derived overdue must not be persisted as a charge state")
	}
	base.SettlementState = PendingSettlement
	if base.Valid() {
		t.Fatal("ad hoc pending settlement must not be persisted as a contract charge state")
	}
}

func TestPostPaymentCancellationCreatesCreditAndRefundAlternative(t *testing.T) {
	charge := Charge{ID: "charge", AssignmentID: "assignment", CurrentAmountMinor: 10000, OriginalAmountMinor: 10000, Currency: "PLN", SettlementState: Paid}
	adjustment, err := ReduceCharge(charge, 5000, "teacher cancellation", Actor{Role: TeacherActor, ID: "teacher"}, ledgerTime(2026, time.September, 7, 12))
	if err != nil || adjustment.Credit == nil || adjustment.Credit.RemainingMinor != 5000 || adjustment.Charge.CurrentAmountMinor != 10000 || len(adjustment.Entries) != 2 || adjustment.Entries[1].EntryType != EntryCreditCreated {
		t.Fatalf("paid cancellation must create a credit: %#v %v", adjustment, err)
	}
	credit, refund, entry, err := RefundCredit(*adjustment.Credit, charge.ID, "cash refund recorded", Actor{Role: TeacherActor, ID: "teacher"}, ledgerTime(2026, time.September, 7, 13))
	if err != nil || credit.RemainingMinor != 0 || refund.AmountMinor != 5000 || entry.EntryType != EntryRefund {
		t.Fatalf("refund alternative must consume credit and append refund: %#v %#v %#v %v", credit, refund, entry, err)
	}
	if _, err := ReduceCharge(charge, 10001, "too much", Actor{Role: TeacherActor, ID: "teacher"}, ledgerTime(2026, time.September, 7, 14)); err != ErrAdjustmentExceedsCharge {
		t.Fatalf("paid adjustment must not create credit above charge: %v", err)
	}
}

func TestCorrectionEntryRetainsValidatedCurrency(t *testing.T) {
	correction, entry, err := NewCorrection(Correction{ID: "correction", AssignmentID: "assignment", CorrectsEntryID: "entry", AmountMinor: -5000, Currency: "PLN", Reason: "corrected cancellation", Actor: Actor{Role: TeacherActor, ID: "teacher"}, RecordedAt: ledgerTime(2026, time.September, 7, 12)})
	if err != nil || correction.Currency != "PLN" || entry.Currency != "PLN" {
		t.Fatalf("correction must retain currency in entry: %#v %#v %v", correction, entry, err)
	}
	if _, _, err := NewCorrection(Correction{AssignmentID: "assignment", CorrectsEntryID: "entry", Currency: "", Reason: "missing currency", Actor: Actor{Role: TeacherActor, ID: "teacher"}, RecordedAt: ledgerTime(2026, time.September, 7, 12)}); err == nil {
		t.Fatal("correction with missing currency must be rejected")
	}
}

func TestCreditsApplyOldestFirst(t *testing.T) {
	charge := Charge{ID: "next", AssignmentID: "assignment", CurrentAmountMinor: 7000, OriginalAmountMinor: 7000, Currency: "PLN", SettlementState: PendingPayment}
	credits := []Credit{{ID: "new", AssignmentID: "assignment", RemainingMinor: 4000, Currency: "PLN", CreatedAt: ledgerTime(2026, time.September, 2, 0)}, {ID: "old", AssignmentID: "assignment", RemainingMinor: 4000, Currency: "PLN", CreatedAt: ledgerTime(2026, time.September, 1, 0)}}
	decision, err := ApplyCredits(charge, credits, ledgerTime(2026, time.September, 3, 0))
	if err != nil || decision.AppliedMinor != 7000 || decision.Charge.CurrentAmountMinor != 0 || decision.Credits[0].RemainingMinor != 0 || decision.Credits[1].RemainingMinor != 1000 {
		t.Fatalf("credits must apply oldest first: %#v %v", decision, err)
	}
	charge.SettlementState = Paid
	if _, err := ApplyCredits(charge, credits, ledgerTime(2026, time.September, 3, 0)); err == nil {
		t.Fatal("credits must not apply to paid charges")
	}
	charge.SettlementState = NotApplicable
	if _, err := ApplyCredits(charge, credits, ledgerTime(2026, time.September, 3, 0)); err == nil {
		t.Fatal("credits must not apply to not-applicable charges")
	}
	charge.SettlementState = PendingPayment
	charge.Currency = ""
	if _, err := ApplyCredits(charge, credits, ledgerTime(2026, time.September, 3, 0)); err == nil {
		t.Fatal("credits must reject invalid charge money")
	}
}

func TestAdHocSettlementAndCancellationHistory(t *testing.T) {
	lesson := AdHocLesson{ID: "lesson", AssignmentID: "assignment", StartAt: ledgerTime(2026, time.September, 1, 10), EndAt: ledgerTime(2026, time.September, 1, 11), UnitPriceMinor: 8000, Currency: "PLN", SettlementState: PendingSettlement}
	paid, err := SettleAdHoc(lesson, AdHocSettlementCommand{State: Paid, Actor: Actor{Role: TeacherActor, ID: "teacher"}, At: ledgerTime(2026, time.September, 1, 12)})
	if err != nil || paid.Lesson.SettlementState != Paid || len(paid.Lesson.Transitions) != 1 || paid.Entry.AmountMinor != 8000 || paid.Entry.Currency != "PLN" {
		t.Fatalf("teacher paid settlement must append transition: %#v %v", paid, err)
	}
	unpaid, err := SettleAdHoc(lesson, AdHocSettlementCommand{State: IntentionallyUnpaid, Actor: Actor{Role: TeacherActor, ID: "teacher"}, At: ledgerTime(2026, time.September, 1, 12)})
	if err != nil || unpaid.Lesson.SettlementState != IntentionallyUnpaid {
		t.Fatalf("teacher intentional nonpayment must be distinct: %#v %v", unpaid, err)
	}
	later, err := SettleAdHoc(unpaid.Lesson, AdHocSettlementCommand{State: Paid, Actor: Actor{Role: TeacherActor, ID: "teacher"}, At: ledgerTime(2026, time.September, 2, 12)})
	if err != nil || len(later.Lesson.Transitions) != 2 || later.Lesson.Transitions[0].To != IntentionallyUnpaid || later.Lesson.SettlementState != Paid {
		t.Fatalf("later ad hoc payment must preserve unpaid history: %#v %v", later, err)
	}
	notApplicable, err := ResolveAdHocNotApplicable(lesson, "learner_no_show", Actor{Role: TeacherActor, ID: "teacher"}, ledgerTime(2026, time.September, 1, 12))
	if err != nil || notApplicable.Lesson.SettlementState != NotApplicable || notApplicable.Entry.AmountMinor != 0 {
		t.Fatalf("no-show must have no debt: %#v %v", notApplicable, err)
	}
	cancelled, err := ResolveAdHocNotApplicable(lesson, "cancelled", Actor{Role: LearnerActor, ID: "learner"}, ledgerTime(2026, time.September, 1, 9))
	if err != nil || cancelled.Lesson.SettlementState != NotApplicable {
		t.Fatalf("learner cancellation must resolve ad hoc settlement: %#v %v", cancelled, err)
	}
	if _, err := ResolveAdHocNotApplicable(lesson, "learner_no_show", Actor{Role: LearnerActor, ID: "learner"}, ledgerTime(2026, time.September, 1, 12)); err != ErrTeacherRequired {
		t.Fatalf("learner must not record a no-show: %v", err)
	}
}

func TestUnpaidAdHocDoesNotBlockSummaryOrBooking(t *testing.T) {
	lesson := AdHocLesson{ID: "lesson", UnitPriceMinor: 8000, Currency: "PLN", SettlementState: IntentionallyUnpaid}
	summary := BuildFinancialSummary([]AdHocLesson{lesson}, nil, nil, ledgerTime(2026, time.September, 7, 0), time.UTC)
	if summary.PendingSettlement != 0 || summary.UnpaidAdHoc != 1 {
		t.Fatalf("intentional nonpayment remains visible but non-blocking: %#v", summary)
	}
}

func TestPackagePurchaseCreatesPaidEntryAndFourTokens(t *testing.T) {
	assignment := commercial.Assignment{ID: "assignment", TeacherID: "teacher", LearnerID: "learner"}
	purchased, err := commercial.NewPackage(commercial.PurchaseRequest{ID: "package", Assignment: assignment, PurchaseDate: ledgerTime(2026, time.September, 13, 0), Now: ledgerTime(2026, time.September, 13, 12), TeacherZone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := RecordPackagePurchase(PackagePurchaseCommand{Package: &purchased, Actor: Actor{Role: TeacherActor, ID: "teacher"}, At: ledgerTime(2026, time.September, 13, 12)})
	if err != nil || decision.TokenCount != 4 || decision.Payment.EntryType != EntryPackagePurchase || decision.Payment.AmountMinor != 26000 || purchased.ValidThrough.Format("2006-01-02") != "2026-11-11" {
		t.Fatalf("package purchase must produce paid payment for four tokens: %#v %v", decision, err)
	}
}
