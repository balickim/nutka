// This file defines bounded, English ledger DTOs with no PocketBase record fields.
// Learner views intentionally contain only assignment-owned commercial values.
package ledgerapi

// FinancialWork is the teacher's bounded actionable payment and outcome queue.
type FinancialWork struct {
	Charges []ChargeView         `json:"charges"`
	Entries []FinancialEntryView `json:"entries"`
	Credits []FinancialEntryView `json:"credits"`
	Next    string               `json:"next,omitempty"`
}

// FinancialSummary is the learner's compact owned payment projection.
type FinancialSummary struct {
	Charges           []ChargeView         `json:"charges"`
	Credits           []FinancialEntryView `json:"credits"`
	PendingSettlement int                  `json:"pending_settlement"`
	UnpaidAdHoc       int                  `json:"unpaid_ad_hoc"`
	UnpaidCharges     int                  `json:"unpaid_charges"`
	OverdueCharges    int                  `json:"overdue_charges"`
	OpenCreditMinor   int64                `json:"open_credit_minor"`
	Currency          string               `json:"currency"`
}

// FinancialHistoryPage is a bounded assignment-scoped financial history page.
type FinancialHistoryPage struct {
	Items      []FinancialEntryView `json:"items"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalItems int                  `json:"total_items"`
	TotalPages int                  `json:"total_pages"`
	Total      int                  `json:"total"`
}

// FinancialEntryView omits internal notes and unrelated identities by construction.
type FinancialEntryView struct {
	ID              string `json:"id"`
	AssignmentID    string `json:"assignment"`
	ChargeID        string `json:"charge,omitempty"`
	EntryType       string `json:"entry_type"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	EffectiveOn     string `json:"effective_on"`
	RelatedLessonID string `json:"related_lesson,omitempty"`
	RelatedEntryID  string `json:"related_entry,omitempty"`
	ActorRole       string `json:"actor_role"`
	EventAt         string `json:"event_at"`
}

// ContractMonthPage contains forecasts before a month and charges after its boundary.
type ContractMonthPage struct {
	Items      []ContractMonthView `json:"months"`
	Page       int                 `json:"page,omitempty"`
	PerPage    int                 `json:"per_page,omitempty"`
	TotalItems int                 `json:"total_items,omitempty"`
	TotalPages int                 `json:"total_pages,omitempty"`
}

// ContractMonthView keeps forecast and charge values separate and explicit.
type ContractMonthView struct {
	ID                  string   `json:"id"`
	ContractID          string   `json:"contract"`
	AssignmentID        string   `json:"assignment"`
	Month               string   `json:"month"`
	BillableCount       int      `json:"billable_count"`
	ForecastAmountMinor int64    `json:"forecast_amount_minor"`
	CurrentAmountMinor  int64    `json:"current_amount_minor,omitempty"`
	Currency            string   `json:"currency"`
	Forecast            bool     `json:"forecast"`
	ChargeID            *string  `json:"charge"`
	SettlementState     string   `json:"settlement_state,omitempty"`
	DerivedState        string   `json:"derived_state,omitempty"`
	DueOn               string   `json:"due_on,omitempty"`
	GeneratedAt         string   `json:"generated_at,omitempty"`
	OccurrenceIDs       []string `json:"occurrence_ids,omitempty"`
}

// UnresolvedWork is the complete teacher action queue without unbounded history.
type UnresolvedWork struct {
	AwaitingOutcome    []UnresolvedOutcome `json:"awaiting_outcome"`
	PendingSettlements []PendingSettlement `json:"pending_settlements"`
	UnpaidCharges      []ChargeView        `json:"unpaid_charges"`
	OverdueCharges     []ChargeView        `json:"overdue_charges"`
}

// UnresolvedOutcome identifies an ended lesson requiring an explicit outcome.
type UnresolvedOutcome struct {
	LessonID     string `json:"lesson"`
	AssignmentID string `json:"assignment"`
	EndedAt      string `json:"ended_at"`
}

// PendingSettlement identifies one ended ad hoc lesson awaiting teacher settlement.
type PendingSettlement struct {
	LessonID     string `json:"lesson"`
	AssignmentID string `json:"assignment"`
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
}

// ChargeView is safe for both teacher and learner financial reads.
type ChargeView struct {
	ID                  string `json:"id"`
	AssignmentID        string `json:"assignment"`
	SourceType          string `json:"source_type"`
	SourceID            string `json:"source_id"`
	Period              string `json:"period,omitempty"`
	OriginalAmountMinor int64  `json:"original_amount_minor"`
	CurrentAmountMinor  int64  `json:"current_amount_minor"`
	Currency            string `json:"currency"`
	SettlementState     string `json:"settlement_state"`
	DerivedState        string `json:"derived_state"`
	Overdue             bool   `json:"overdue"`
	DueOn               string `json:"due_on,omitempty"`
	PaidAt              string `json:"paid_at,omitempty"`
}

// LessonPayment is the safe projection returned after an ad hoc settlement.
type LessonPayment struct {
	LessonID        string `json:"lesson"`
	AssignmentID    string `json:"assignment"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	SettlementState string `json:"settlement_state"`
}
