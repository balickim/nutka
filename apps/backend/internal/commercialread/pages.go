// This file serves bounded contract, financial, history, and unresolved-work projections.
package commercialread

import (
	"sort"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/history"
)

type ContractSeriesPage struct {
	NearTerm   []ContractOccurrenceView `json:"near_term"`
	Later      []ContractOccurrenceView `json:"later"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"per_page"`
	TotalItems int                      `json:"total_items"`
	TotalPages int                      `json:"total_pages"`
}

func QueryContractSeries(request ContractSeriesRequest) (ContractSeriesPage, error) {
	policy, err := validateContractSeries(request)
	if err != nil {
		return ContractSeriesPage{}, err
	}
	near, later := partitionContract(request.Contract, request.Now, policy.BookingHorizon)
	all := append(append([]ContractOccurrenceView{}, occurrencesViews(near)...), occurrencesViews(later)...)
	if request.Viewer.Role == LearnerRole {
		all = occurrencesViews(near)
	}
	start, end := pageBounds(len(all), request.Page, request.PerPage)
	page := ContractSeriesPage{NearTerm: []ContractOccurrenceView{}, Later: []ContractOccurrenceView{}, Page: request.Page, PerPage: request.PerPage, TotalItems: len(all), TotalPages: pages(len(all), request.PerPage)}
	for index, item := range all[start:end] {
		appendSeriesItem(&page, item, request.Viewer.Role, start+index, len(near))
	}
	return page, nil
}

func validateContractSeries(request ContractSeriesRequest) (businesspolicy.Policy, error) {
	if err := validPage(request.Page, request.PerPage); err != nil {
		return businesspolicy.Policy{}, err
	}
	if err := authorize(request.Assignment, request.Viewer); err != nil {
		return businesspolicy.Policy{}, err
	}
	if request.Contract.AssignmentID != request.Assignment.ID || request.Contract.TeacherID != request.Assignment.TeacherID || request.Contract.LearnerID != request.Assignment.LearnerID {
		return businesspolicy.Policy{}, ErrUnauthorized
	}
	policy, err := readPolicy(request.Policy)
	if err != nil || request.Now.IsZero() {
		return businesspolicy.Policy{}, ErrPolicy
	}
	return policy, nil
}

func appendSeriesItem(page *ContractSeriesPage, item ContractOccurrenceView, role Role, index, nearCount int) {
	if role == LearnerRole || index < nearCount {
		page.NearTerm = append(page.NearTerm, item)
		return
	}
	page.Later = append(page.Later, item)
}

type FinancialHistoryRequest struct {
	Assignment    Assignment
	Viewer        Viewer
	Entries       []FinancialEntryInput
	Page, PerPage int
}

func QueryFinancialHistory(request FinancialHistoryRequest) (Page[FinancialEntryView], error) {
	if err := validPage(request.Page, request.PerPage); err != nil {
		return Page[FinancialEntryView]{}, err
	}
	if err := authorize(request.Assignment, request.Viewer); err != nil {
		return Page[FinancialEntryView]{}, err
	}
	values := make([]FinancialEntryView, 0, len(request.Entries))
	for _, entry := range request.Entries {
		if entry.AssignmentID != request.Assignment.ID {
			continue
		}
		values = append(values, financialView(entry, request.Viewer.Role))
	}
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].EventAt > values[j].EventAt || values[i].EventAt == values[j].EventAt && values[i].ID > values[j].ID
	})
	return pageOf(values, request.Page, request.PerPage), nil
}

type HistoryRequest struct {
	Assignment    Assignment
	Viewer        Viewer
	Events        []history.Event
	Page, PerPage int
}

func QueryBusinessHistory(request HistoryRequest) (Page[BusinessEventView], error) {
	if err := validPage(request.Page, request.PerPage); err != nil {
		return Page[BusinessEventView]{}, err
	}
	if err := authorize(request.Assignment, request.Viewer); err != nil {
		return Page[BusinessEventView]{}, err
	}
	values := make([]BusinessEventView, 0, len(request.Events))
	for _, event := range request.Events {
		if event.AssignmentID != request.Assignment.ID || request.Viewer.Role == LearnerRole && event.Type == history.AvailabilityChanged {
			continue
		}
		values = append(values, eventView(event, request.Viewer.Role))
	}
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].EventAt > values[j].EventAt || values[i].EventAt == values[j].EventAt && values[i].ID > values[j].ID
	})
	return pageOf(values, request.Page, request.PerPage), nil
}

type UnresolvedRequest struct {
	Viewer        Viewer
	Assignments   []Assignment
	Work          []UnresolvedInput
	Page, PerPage int
}

func QueryUnresolvedWork(request UnresolvedRequest) (UnresolvedWork, error) {
	if request.Viewer.Role != TeacherRole || !request.Viewer.valid() {
		return UnresolvedWork{}, ErrUnauthorized
	}
	if request.Page == 0 {
		request.Page = 1
	}
	if request.PerPage == 0 {
		request.PerPage = 100
	}
	if err := validPage(request.Page, request.PerPage); err != nil {
		return UnresolvedWork{}, err
	}
	allowed := make(map[string]bool)
	for _, assignment := range request.Assignments {
		if authorize(assignment, request.Viewer) == nil {
			allowed[assignment.ID] = true
		}
	}
	result := UnresolvedWork{AwaitingOutcome: []UnresolvedItem{}, PendingSettlements: []UnresolvedItem{}, UnpaidCharges: []UnresolvedItem{}, OverdueCharges: []UnresolvedItem{}}
	for _, value := range request.Work {
		result.AwaitingOutcome = append(result.AwaitingOutcome, allowedItems(value.AwaitingOutcomes, allowed)...)
		result.PendingSettlements = append(result.PendingSettlements, allowedItems(value.PendingSettlements, allowed)...)
		result.UnpaidCharges = append(result.UnpaidCharges, allowedItems(value.UnpaidCharges, allowed)...)
		result.OverdueCharges = append(result.OverdueCharges, allowedItems(value.OverdueCharges, allowed)...)
	}
	result.AwaitingOutcome = boundedItems(result.AwaitingOutcome, request.Page, request.PerPage)
	result.PendingSettlements = boundedItems(result.PendingSettlements, request.Page, request.PerPage)
	result.UnpaidCharges = boundedItems(result.UnpaidCharges, request.Page, request.PerPage)
	result.OverdueCharges = boundedItems(result.OverdueCharges, request.Page, request.PerPage)
	return result, nil
}

func safeItems(values []UnresolvedItem) []UnresolvedItem {
	result := make([]UnresolvedItem, 0, len(values))
	for _, item := range values {
		result = append(result, UnresolvedItem{ID: item.ID, Assignment: item.Assignment, Lesson: item.Lesson, Charge: item.Charge, EndedAt: item.EndedAt, DueOn: item.DueOn, AmountMinor: item.AmountMinor, Currency: item.Currency})
	}
	return result
}

func allowedItems(values []UnresolvedItem, allowed map[string]bool) []UnresolvedItem {
	result := make([]UnresolvedItem, 0, len(values))
	for _, value := range values {
		if allowed[value.Assignment] {
			result = append(result, safeItems([]UnresolvedItem{value})...)
		}
	}
	return result
}

func financialView(value FinancialEntryInput, role Role) FinancialEntryView {
	view := FinancialEntryView{ID: value.ID, Assignment: value.AssignmentID, Charge: value.ChargeID, EntryType: string(value.EntryType), AmountMinor: value.AmountMinor, Currency: value.Currency, EffectiveOn: value.EffectiveOn, RelatedLesson: value.RelatedLessonID, EventAt: instant(value.EventAt)}
	if role == TeacherRole {
		view.Reason = value.Reason
	}
	return view
}

func eventView(value history.Event, role Role) BusinessEventView {
	view := BusinessEventView{ID: value.ID, EventType: string(value.Type), AggregateType: value.AggregateType, AggregateID: value.AggregateID, Assignment: value.AssignmentID, ActorRole: string(value.Actor.Role), EventAt: instant(value.EventAt), RelatedIDs: safeRelatedIDs(value.RelatedIDs), PriorState: cloneMap(value.PriorState), NewState: cloneMap(value.NewState), Reason: value.Reason, InternalNote: value.InternalNote, CorrectsEvent: value.CorrectsEventID}
	if role == LearnerRole {
		view.Reason = ""
		view.InternalNote = ""
		view.PriorState = redact(value.PriorState)
		view.NewState = redact(value.NewState)
	}
	return view
}
