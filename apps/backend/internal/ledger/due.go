// This file decides which charges a learner still has to pay, their order, and their total.
// Credits already reduce charge amounts, so the total never subtracts open credit again.
package ledger

import (
	"sort"
	"time"
)

const (
	AdHocSource          = "ad_hoc"
	DueKindContractMonth = "contract_month"
	DueKindLesson        = "lesson"
)

// DueCharge is a charge with the lesson facts that the ad hoc rule needs.
type DueCharge struct {
	Charge        Charge
	LessonStartAt time.Time
	LessonEnded   bool
}

// DueItem is one open amount of an assignment.
type DueItem struct {
	ChargeID      string
	Kind          string
	Period        string
	LessonStartAt time.Time
	AmountMinor   int64
	Currency      string
	DueOn         string
	Overdue       bool
}

// DueItems returns open items with overdue items first, then by due date or lesson start, and their total.
func DueItems(charges []DueCharge, now time.Time, location *time.Location) ([]DueItem, int64) {
	items := make([]DueItem, 0)
	var total int64
	for _, due := range charges {
		if !isOpen(due) {
			continue
		}
		items = append(items, dueItem(due, now, location))
		total += due.Charge.CurrentAmountMinor
	}
	sort.SliceStable(items, func(left, right int) bool {
		if items[left].Overdue != items[right].Overdue {
			return items[left].Overdue
		}
		return dueOrder(items[left]) < dueOrder(items[right])
	})
	return items, total
}

func isOpen(due DueCharge) bool {
	state := due.Charge.SettlementState
	if due.Charge.CurrentAmountMinor <= 0 {
		return false
	}
	if due.Charge.SourceType == AdHocSource {
		return state == IntentionallyUnpaid || (state == PendingSettlement && due.LessonEnded)
	}
	return state == PendingPayment || state == IntentionallyUnpaid
}

func dueItem(due DueCharge, now time.Time, location *time.Location) DueItem {
	item := DueItem{ChargeID: due.Charge.ID, Kind: DueKindContractMonth, Period: due.Charge.Period, AmountMinor: due.Charge.CurrentAmountMinor, Currency: due.Charge.Currency, DueOn: due.Charge.DueOn, Overdue: due.Charge.IsOverdue(now, location)}
	if due.Charge.SourceType == AdHocSource {
		item.Kind = DueKindLesson
		item.LessonStartAt = due.LessonStartAt.UTC()
	}
	return item
}

// dueOrder compares a date-only due date and a lesson instant by their calendar date text.
func dueOrder(item DueItem) string {
	if item.Kind == DueKindLesson {
		return item.LessonStartAt.Format(time.RFC3339)
	}
	return item.DueOn
}
