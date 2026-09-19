package ledger

import (
	"testing"
	"time"
)

func TestDueItemsListsOpenChargesOverdueFirst(t *testing.T) {
	warsaw, _ := time.LoadLocation("Europe/Warsaw")
	now := time.Date(2026, time.October, 6, 10, 0, 0, 0, time.UTC)
	lesson := time.Date(2026, time.October, 2, 15, 0, 0, 0, time.UTC)
	month := func(id, period, due string, state SettlementState, amount int64) DueCharge {
		return DueCharge{Charge: Charge{ID: id, SourceType: "regular_contract", Period: period, CurrentAmountMinor: amount, Currency: "PLN", SettlementState: state, DueOn: due}}
	}
	adHoc := func(id string, state SettlementState, ended bool) DueCharge {
		return DueCharge{Charge: Charge{ID: id, SourceType: AdHocSource, CurrentAmountMinor: 8000, Currency: "PLN", SettlementState: state}, LessonStartAt: lesson, LessonEnded: ended}
	}
	charges := []DueCharge{
		month("november", "2026-11", "2026-11-05", PendingPayment, 20000),
		month("october", "2026-10", "2026-10-05", PendingPayment, 20000),
		month("paid", "2026-09", "2026-09-05", Paid, 20000),
		month("credited", "2026-12", "2026-12-05", PendingPayment, 0),
		adHoc("ended", PendingSettlement, true),
		adHoc("future", PendingSettlement, false),
		adHoc("unpaid", IntentionallyUnpaid, false),
		adHoc("cancelled", NotApplicable, true),
	}
	items, total := DueItems(charges, now, warsaw)
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.ChargeID)
	}
	want := []string{"october", "ended", "unpaid", "november"}
	if len(got) != len(want) {
		t.Fatalf("open items %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("open items %v, want %v", got, want)
		}
	}
	if total != 56000 {
		t.Fatalf("total %d, want 56000", total)
	}
	if !items[0].Overdue || items[0].Kind != DueKindContractMonth || items[1].Kind != DueKindLesson || items[1].Overdue {
		t.Fatalf("item flags: %+v", items[:2])
	}
}

func TestDueItemsIsEmptyWithoutOpenCharges(t *testing.T) {
	items, total := DueItems(nil, time.Now(), time.UTC)
	if len(items) != 0 || total != 0 {
		t.Fatalf("empty input produced %v %d", items, total)
	}
}
