// Package commercialread builds bounded, role-scoped commercial and calendar read models.
// It accepts domain values at the persistence boundary and never exposes PocketBase records.
package commercialread

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/domain"
)

type Role string

const (
	TeacherRole Role = "teacher"
	LearnerRole Role = "learner"
)

type Viewer struct {
	Role Role
	ID   string
}

func (v Viewer) valid() bool { return (v.Role == TeacherRole || v.Role == LearnerRole) && v.ID != "" }

type Assignment struct {
	ID        string `json:"id"`
	TeacherID string `json:"teacher"`
	LearnerID string `json:"learner"`
	Active    bool   `json:"active"`
}

type TokenBalance struct {
	Total       int `json:"total"`
	Available   int `json:"available"`
	Reserved    int `json:"reserved"`
	Used        int `json:"used"`
	Expired     int `json:"expired"`
	Invalidated int `json:"invalidated"`
}

type PaymentSummary struct {
	Pending             int64  `json:"pending"`
	IntentionallyUnpaid int64  `json:"intentionally_unpaid"`
	Overdue             int64  `json:"overdue"`
	CreditMinor         int64  `json:"credit_minor"`
	Currency            string `json:"currency,omitempty"`
}

type ContractAllowances struct {
	MonthlyReschedulesRemaining int `json:"monthly_reschedules_remaining"`
	FreeCancellationsRemaining  int `json:"free_cancellations_remaining"`
}

type AssignmentCommercialSummary struct {
	AssignmentID string           `json:"assignment"`
	ActivePlan   domain.PlanType  `json:"active_plan,omitempty"`
	Package      *PackageSummary  `json:"package"`
	Contract     *ContractSummary `json:"contract"`
	Payments     PaymentSummary   `json:"payments"`
}

type PackageSummary struct {
	ID           string       `json:"id"`
	ValidThrough string       `json:"valid_through"`
	TokenBalance TokenBalance `json:"token_balance"`
}

type ContractSummary struct {
	Status                      string `json:"status"`
	ID                          string `json:"id"`
	StartOn                     string `json:"start_on"`
	EndOn                       string `json:"end_on"`
	RemainingMonthlyReschedules int    `json:"remaining_monthly_reschedules"`
	RemainingFreeCancellations  int    `json:"remaining_free_cancellations"`
	PriceMinor                  int64  `json:"price_minor"`
	Currency                    string `json:"currency"`
}

type LessonInput struct {
	ID, AssignmentID, TeacherID, LearnerID string
	Plan                                   domain.PlanType
	PackageTokenID, ContractID             string
	StartAt, EndAt                         time.Time
	DurationMinutes                        int
	UnitPriceMinor                         int64
	Currency                               string
	ScheduleState                          domain.ScheduleState
	Outcome                                domain.Outcome
	PolicyVersion                          string
}

type LessonView struct {
	ID               string `json:"id"`
	Teacher          string `json:"teacher"`
	Learner          string `json:"learner"`
	Assignment       string `json:"assignment"`
	PlanType         string `json:"plan_type"`
	PackageToken     string `json:"package_token,omitempty"`
	Contract         string `json:"contract,omitempty"`
	StartAt          string `json:"start_at"`
	EndAt            string `json:"end_at"`
	DurationMinutes  int    `json:"duration_minutes"`
	UnitPriceMinor   int64  `json:"unit_price_minor"`
	Currency         string `json:"currency"`
	PolicyVersion    string `json:"policy_version"`
	ScheduleState    string `json:"schedule_state"`
	Outcome          string `json:"outcome,omitempty"`
	ProtectedStartAt string `json:"protected_start_at,omitempty"`
	ProtectedEndAt   string `json:"protected_end_at,omitempty"`
}

type ContractOccurrenceView struct {
	ID, Contract, Assignment string
	OriginalLocalDate        string `json:"original_local_date"`
	StartAt, EndAt           string `json:"start_at"`
	ScheduleState, Outcome   string `json:"schedule_state,omitempty"`
	UnitPriceMinor           int64  `json:"unit_price_minor"`
	Currency                 string `json:"currency"`
}

type AssignmentView struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
}

type UnresolvedCounts struct {
	AwaitingOutcome   int `json:"awaiting_outcome"`
	PendingSettlement int `json:"pending_settlement"`
	UnpaidCharges     int `json:"unpaid_charges"`
	OverdueCharges    int `json:"overdue_charges"`
}

type TeacherCalendar struct {
	Assignments          []AssignmentView              `json:"assignments"`
	CommercialSummaries  []AssignmentCommercialSummary `json:"commercial_summaries"`
	NearTermLessons      []LessonView                  `json:"near_term_lessons"`
	LaterContractLessons []ContractOccurrenceView      `json:"later_contract_lessons"`
	UnresolvedWork       UnresolvedCounts              `json:"unresolved_work"`
}

type LearnerCalendar struct {
	Assignments         []AssignmentView              `json:"assignments"`
	CommercialSummaries []AssignmentCommercialSummary `json:"commercial_summaries"`
	NearTermLessons     []LessonView                  `json:"near_term_lessons"`
	PaymentSummary      []PaymentSummary              `json:"payment_summary"`
	HistorySummary      []HistorySummary              `json:"history_summary"`
}

type HistorySummary struct {
	Assignment string `json:"assignment"`
	EventCount int    `json:"event_count"`
}

type FinancialEntryView struct {
	ID            string `json:"id"`
	Assignment    string `json:"assignment"`
	Charge        string `json:"charge,omitempty"`
	EntryType     string `json:"entry_type"`
	AmountMinor   int64  `json:"amount_minor"`
	Currency      string `json:"currency"`
	EffectiveOn   string `json:"effective_on,omitempty"`
	RelatedLesson string `json:"related_lesson,omitempty"`
	Reason        string `json:"reason,omitempty"`
	EventAt       string `json:"event_at"`
}

type BusinessEventView struct {
	ID            string            `json:"id"`
	EventType     string            `json:"event_type"`
	AggregateType string            `json:"aggregate_type"`
	AggregateID   string            `json:"aggregate_id"`
	Assignment    string            `json:"assignment"`
	ActorRole     string            `json:"actor_role"`
	EventAt       string            `json:"event_at"`
	RelatedIDs    map[string]string `json:"related_ids,omitempty"`
	PriorState    map[string]any    `json:"prior_state,omitempty"`
	NewState      map[string]any    `json:"new_state,omitempty"`
	Reason        string            `json:"reason,omitempty"`
	InternalNote  string            `json:"internal_note,omitempty"`
	CorrectsEvent string            `json:"corrects_event,omitempty"`
}

type UnresolvedItem struct {
	ID          string `json:"id"`
	Assignment  string `json:"assignment"`
	Lesson      string `json:"lesson,omitempty"`
	Charge      string `json:"charge,omitempty"`
	EndedAt     string `json:"ended_at,omitempty"`
	DueOn       string `json:"due_on,omitempty"`
	AmountMinor int64  `json:"amount_minor,omitempty"`
	Currency    string `json:"currency,omitempty"`
}

type UnresolvedWork struct {
	AwaitingOutcome    []UnresolvedItem `json:"awaiting_outcome"`
	PendingSettlements []UnresolvedItem `json:"pending_settlements"`
	UnpaidCharges      []UnresolvedItem `json:"unpaid_charges"`
	OverdueCharges     []UnresolvedItem `json:"overdue_charges"`
}
