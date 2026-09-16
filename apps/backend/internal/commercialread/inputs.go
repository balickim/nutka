// This file defines persistence-independent inputs and request envelopes for read projections.
package commercialread

import (
	"errors"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
)

type AssignmentData struct {
	Assignment    Assignment
	Contract      *regularcontract.RegularContract
	Packages      []*commercial.Package
	AdHocLessons  []ledger.AdHocLesson
	Lessons       []LessonInput
	Charges       []ledger.Charge
	Credits       []ledger.Credit
	HistoryEvents []history.Event
}

type SummaryInput struct {
	Assignment   Assignment
	Contract     *regularcontract.RegularContract
	Packages     []*commercial.Package
	AdHocLessons []ledger.AdHocLesson
	Charges      []ledger.Charge
	Credits      []ledger.Credit
	Now          time.Time
	Location     *time.Location
	Viewer       Viewer
}

type Policy = businesspolicy.Policy

type CalendarRequest struct {
	Assignments []AssignmentData
	Now         time.Time
	Policy      Policy
	Viewer      Viewer
}

type ContractSeriesRequest struct {
	Assignment    Assignment
	Contract      regularcontract.RegularContract
	Now           time.Time
	Policy        Policy
	Viewer        Viewer
	Page, PerPage int
}

type FinancialEntryInput struct {
	ledger.FinancialEntry
	SourceType string
	SourceID   string
}

type UnresolvedInput struct {
	AwaitingOutcomes   []UnresolvedItem
	PendingSettlements []UnresolvedItem
	UnpaidCharges      []UnresolvedItem
	OverdueCharges     []UnresolvedItem
}

type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

var (
	ErrUnauthorized = errors.New("read authorization failed")
	ErrPagination   = errors.New("read pagination is invalid")
	ErrPolicy       = errors.New("read policy is invalid")
	ErrTimezone     = errors.New("read timezone is invalid")
	ErrDuration     = errors.New("lesson duration is not fixed by policy")
)
