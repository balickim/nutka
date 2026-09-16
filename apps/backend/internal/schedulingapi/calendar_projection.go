// This file builds role-scoped calendar projections and separates near-term lessons from later contract occurrences.
package schedulingapi

import (
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercialapi"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func calendarValue(app core.App, assignments, rules, exceptions, lessons []*core.Record, role string, now time.Time) (calendarDTO, error) {
	result := newCalendarDTO(assignments, rules, exceptions, role)
	if err := appendCalendarAssignments(app, assignments, role, now, &result); err != nil {
		return calendarDTO{}, err
	}
	appendCalendarAvailability(rules, exceptions, &result)
	partitionCalendarLessons(lessons, role, now, &result)
	if role == "teacher" {
		if err := collectUnresolvedFinancial(app, assignments, lessons, now, result.UnresolvedWork); err != nil {
			return calendarDTO{}, err
		}
	}
	sortCalendarLessons(&result)
	return result, nil
}

func newCalendarDTO(assignments, rules, exceptions []*core.Record, role string) calendarDTO {
	result := calendarDTO{Assignments: make([]assignmentDTO, 0, len(assignments)), Rules: make([]ruleDTO, 0, len(rules)), Exceptions: make([]exceptionDTO, 0, len(exceptions)), CommercialSummaries: make([]any, 0, len(assignments)), NearTermLessons: []lessonDTO{}}
	if role == "teacher" {
		result.LaterContractLessons = []lessonDTO{}
		result.UnresolvedWork = &unresolvedDTO{}
	} else {
		result.PaymentSummary = make([]any, 0, len(assignments))
		result.HistorySummary = make([]historyCount, 0, len(assignments))
	}
	return result
}

func appendCalendarAssignments(app core.App, assignments []*core.Record, role string, now time.Time, result *calendarDTO) error {
	for _, row := range assignments {
		item, err := assignmentValue(app, row)
		if err != nil {
			return err
		}
		result.Assignments = append(result.Assignments, item)
		summary, err := commercialapi.SummaryForAssignment(app, row.Id, now)
		if err != nil {
			return err
		}
		result.CommercialSummaries = append(result.CommercialSummaries, summary)
		if err := appendLearnerCalendarSummary(app, row, role, now, result); err != nil {
			return err
		}
	}
	return nil
}

func appendLearnerCalendarSummary(app core.App, row *core.Record, role string, now time.Time, result *calendarDTO) error {
	if role != "learner" {
		return nil
	}
	payments, err := commercialapi.PaymentSummaryForAssignment(app, row.Id, now)
	if err != nil {
		return err
	}
	result.PaymentSummary = append(result.PaymentSummary, payments)
	events, err := recordsByField(app, schedulingstore.BusinessEventsCollectionName, schedulingstore.AssignmentField, row.Id)
	if err != nil {
		return err
	}
	result.HistorySummary = append(result.HistorySummary, historyCount{Assignment: row.Id, EventCount: len(events)})
	return nil
}

func appendCalendarAvailability(rules, exceptions []*core.Record, result *calendarDTO) {
	for _, row := range rules {
		result.Rules = append(result.Rules, ruleValue(row))
	}
	for _, row := range exceptions {
		result.Exceptions = append(result.Exceptions, exceptionValue(row))
	}
}

func partitionCalendarLessons(lessons []*core.Record, role string, now time.Time, result *calendarDTO) {
	horizon := now.Add(businesspolicy.Current().BookingHorizon)
	for _, row := range lessons {
		start := row.GetDateTime(schedulingstore.StartAtField).Time().UTC()
		if role == "learner" && (start.Before(now) || start.After(horizon)) {
			continue
		}
		if role == "teacher" && row.GetString(schedulingstore.ContractField) != "" && start.After(horizon) {
			result.LaterContractLessons = append(result.LaterContractLessons, lessonValueAt(row, now))
			continue
		}
		result.NearTermLessons = append(result.NearTermLessons, lessonValueAt(row, now))
		if role == "teacher" {
			collectUnresolved(result.UnresolvedWork, row, now)
		}
	}
}

func sortCalendarLessons(result *calendarDTO) {
	sort.Slice(result.NearTermLessons, func(i, j int) bool { return result.NearTermLessons[i].StartAt < result.NearTermLessons[j].StartAt })
	sort.Slice(result.LaterContractLessons, func(i, j int) bool {
		return result.LaterContractLessons[i].StartAt < result.LaterContractLessons[j].StartAt
	})
}

func enabledRules(rows []*core.Record) []*core.Record { return enabledAvailabilityRows(rows) }

func enabledExceptions(rows []*core.Record) []*core.Record { return enabledAvailabilityRows(rows) }

func enabledAvailabilityRows(rows []*core.Record) []*core.Record {
	result := make([]*core.Record, 0, len(rows))
	for _, row := range rows {
		if row.GetBool(schedulingstore.EnabledField) {
			result = append(result, row)
		}
	}
	return result
}

func collectUnresolved(result *unresolvedDTO, row *core.Record, now time.Time) {
	if result == nil || row.GetString(schedulingstore.ScheduleStateField) != "scheduled" || row.GetDateTime(schedulingstore.EndAtField).Time().UTC().After(now) {
		return
	}
	if row.GetString(schedulingstore.OutcomeField) == "" || row.GetString(schedulingstore.OutcomeField) == "awaiting_outcome" {
		result.AwaitingOutcome++
	}
}
