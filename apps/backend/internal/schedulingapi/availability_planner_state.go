// This file converts persisted availability and lesson rows into deterministic planner input state.
package schedulingapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/availabilityimpact"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func plannerRecords(rows []*core.Record, horizon time.Time) ([]availabilityimpact.Lesson, []availabilityimpact.ContractOccurrence, map[string]*core.Record, time.Time) {
	near := make([]availabilityimpact.Lesson, 0, len(rows))
	distant := make([]availabilityimpact.ContractOccurrence, 0)
	lookup := make(map[string]*core.Record, len(rows))
	latest := time.Time{}
	for _, row := range rows {
		start := row.GetDateTime(schedulingstore.StartAtField).Time().UTC()
		end := row.GetDateTime(schedulingstore.EndAtField).Time().UTC()
		if end.After(latest) {
			latest = end
		}
		state := lessonScheduleState(row)
		if row.GetString(schedulingstore.ContractField) != "" && start.After(horizon) {
			distant = append(distant, availabilityimpact.ContractOccurrence{ID: row.Id, Interval: scheduling.Interval{Start: start, End: end}, State: state, OmissionReason: row.GetString(schedulingstore.OmissionReasonField)})
		} else {
			near = append(near, availabilityimpact.Lesson{ID: row.Id, Plan: lessonPlan(row), Interval: scheduling.Interval{Start: start, End: end}, State: state})
		}
		lookup[row.Id] = row
	}
	return near, distant, lookup, latest
}

func lessonScheduleState(row *core.Record) domain.ScheduleState {
	state := domain.ScheduleState(row.GetString(schedulingstore.ScheduleStateField))
	if state == "" {
		state = domain.ScheduleState(row.GetString(schedulingstore.StatusField))
	}
	if state == "" {
		return domain.ScheduledState
	}
	return state
}

func lessonPlan(row *core.Record) domain.PlanType {
	plan := domain.PlanType(row.GetString(schedulingstore.PlanTypeField))
	if plan.Valid() {
		return plan
	}
	if row.GetString(schedulingstore.ContractField) != "" {
		return domain.RegularContractPlan
	}
	return domain.AdHocPlan
}

func availabilityRulesAfterMutation(rows []*core.Record, mutation AvailabilityMutation) ([]scheduling.WeekdayRule, error) {
	values := make([]scheduling.WeekdayRule, 0, len(rows)+1)
	for _, row := range rows {
		value, include, err := ruleAfterMutation(row, mutation)
		if err != nil {
			return nil, err
		}
		if include {
			values = append(values, value)
		}
	}
	created, include, err := createdRule(mutation)
	if err != nil {
		return nil, err
	}
	if include {
		values = append(values, created)
	}
	return values, nil
}

func ruleAfterMutation(row *core.Record, mutation AvailabilityMutation) (scheduling.WeekdayRule, bool, error) {
	matched := mutation.Target == availabilityimpact.RecurringRule && mutation.ID == row.Id
	if matched && mutation.Operation == availabilityimpact.Delete {
		return scheduling.WeekdayRule{}, false, nil
	}
	day, start, end, enabled, err := ruleRecordValues(row)
	if err == nil && matched {
		day, start, end, enabled, err = updatedRuleValues(day, start, end, enabled, mutation.Rule, mutation.Operation)
	}
	return scheduling.WeekdayRule{Weekday: time.Weekday(day), StartMinute: start, EndMinute: end, Enabled: enabled}, true, err
}

func createdRule(mutation AvailabilityMutation) (scheduling.WeekdayRule, bool, error) {
	if mutation.Target != availabilityimpact.RecurringRule || mutation.Operation != availabilityimpact.Create {
		return scheduling.WeekdayRule{}, false, nil
	}
	if mutation.Rule == nil {
		return scheduling.WeekdayRule{}, false, errInvalid
	}
	day, start, end, enabled, err := newRuleValues(mutation.Rule)
	return scheduling.WeekdayRule{Weekday: time.Weekday(day), StartMinute: start, EndMinute: end, Enabled: enabled}, true, err
}

func availabilityExceptionsAfterMutation(rows []*core.Record, mutation AvailabilityMutation) ([]scheduling.AvailabilityException, error) {
	values := make([]scheduling.AvailabilityException, 0, len(rows)+1)
	for _, row := range rows {
		value, include, err := exceptionAfterMutation(row, mutation)
		if err != nil {
			return nil, err
		}
		if include {
			values = append(values, value)
		}
	}
	created, include, err := createdException(mutation)
	if err != nil {
		return nil, err
	}
	if include {
		values = append(values, created)
	}
	return values, nil
}

func exceptionAfterMutation(row *core.Record, mutation AvailabilityMutation) (scheduling.AvailabilityException, bool, error) {
	matched := mutation.Target == availabilityimpact.Exception && mutation.ID == row.Id
	if matched && mutation.Operation == availabilityimpact.Delete {
		return scheduling.AvailabilityException{}, false, nil
	}
	start, end, kind, note, enabled, err := exceptionRecordValues(row)
	if err == nil && matched {
		start, end, kind, note, enabled, err = updatedExceptionValuesForMutation(start, end, kind, note, enabled, mutation.Exception, mutation.Operation)
	}
	interval, _ := scheduling.NewInterval(start, end)
	return scheduling.AvailabilityException{Interval: interval, Kind: scheduling.ExceptionKind(kind), Note: note}, enabled, err
}

func createdException(mutation AvailabilityMutation) (scheduling.AvailabilityException, bool, error) {
	if mutation.Target != availabilityimpact.Exception || mutation.Operation != availabilityimpact.Create {
		return scheduling.AvailabilityException{}, false, nil
	}
	if mutation.Exception == nil {
		return scheduling.AvailabilityException{}, false, errInvalid
	}
	start, end, kind, note, enabled, err := newExceptionValues(mutation.Exception)
	interval, _ := scheduling.NewInterval(start, end)
	return scheduling.AvailabilityException{Interval: interval, Kind: scheduling.ExceptionKind(kind), Note: note}, enabled, err
}

func plannerRecordsVersion(rows []*core.Record) []string {
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.Id+":"+row.GetString(schedulingstore.StartAtField)+":"+row.GetString(schedulingstore.EndAtField)+":"+row.GetString(schedulingstore.EnabledField)+":"+row.GetString(schedulingstore.KindField))
	}
	sort.Strings(values)
	return values
}

func sourceVersion(rules, exceptions []*core.Record) string {
	value, _ := json.Marshal(struct{ Rules, Exceptions []string }{plannerRecordsVersion(rules), plannerRecordsVersion(exceptions)})
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
