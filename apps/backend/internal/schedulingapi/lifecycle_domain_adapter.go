// This file loads the complete persistence-independent state required by one plan-aware lesson lifecycle command.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/commercialapi"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/lesson"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/regularcontractapi"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func lifecycleCommand(app core.App, row *core.Record, actor history.Actor, now, replacement time.Time) (lesson.Command, error) {
	interval, err := scheduling.NewInterval(recordInstant(row, schedulingstore.StartAtField), recordInstant(row, schedulingstore.EndAtField))
	if err != nil {
		return lesson.Command{}, errInvalid
	}
	state := domain.ScheduleState(row.GetString(schedulingstore.ScheduleStateField))
	if state == "" {
		state = domain.ScheduleState(row.GetString(schedulingstore.StatusField))
	}
	command := lesson.Command{Lesson: lesson.Lesson{ID: row.Id, AssignmentID: row.GetString(schedulingstore.AssignmentField), TeacherID: row.GetString("teacher"), LearnerID: row.GetString("learner"), Plan: commercial.PlanType(row.GetString(schedulingstore.PlanTypeField)), Interval: interval, OriginalLocalDate: row.GetString(schedulingstore.OriginalLocalDateField), ScheduleState: state, Outcome: domain.Outcome(row.GetString(schedulingstore.OutcomeField)), Settlement: settlementForLesson(app, row.Id), UnitPriceMinor: int64(row.GetInt(schedulingstore.UnitPriceMinorField)), Currency: row.GetString(schedulingstore.CurrencyField)}, Actor: actor, Now: now.UTC(), ReplacementStart: replacement.UTC()}
	if !replacement.IsZero() {
		command.Availability, err = lifecycleAvailability(app, row.GetString("teacher"), now, replacement)
		if err != nil {
			return lesson.Command{}, err
		}
	}
	command.ParticipantLessons, err = lifecycleParticipantLessons(app, row)
	if err != nil {
		return lesson.Command{}, err
	}
	if tokenID := row.GetString(schedulingstore.PackageTokenField); tokenID != "" {
		command.Package, err = commercialapi.PackageForToken(app, tokenID)
	}
	if contractID := row.GetString(schedulingstore.ContractField); err == nil && contractID != "" {
		value, contractErr := regularcontractapi.LoadContract(app, contractID)
		command.Contract, err = &value, contractErr
	}
	return command, err
}

func lifecycleAvailability(app core.App, teacherID string, now, replacement time.Time) ([]scheduling.Interval, error) {
	teacher, err := app.FindRecordById("teachers", teacherID)
	if err != nil {
		return nil, errForbidden
	}
	ruleRows, err := recordsByField(app, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacherID)
	if err != nil {
		return nil, err
	}
	exceptionRows, err := recordsByField(app, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacherID)
	if err != nil {
		return nil, err
	}
	rules, err := weeklyRules(ruleRows)
	if err != nil {
		return nil, err
	}
	exceptions, err := domainExceptions(exceptionRows)
	if err != nil {
		return nil, err
	}
	end := replacement.Add(24 * time.Hour)
	if end.Before(now.Add(24 * time.Hour)) {
		end = now.Add(24 * time.Hour)
	}
	return scheduling.EffectiveAvailability(now.UTC(), end.UTC(), teacher.GetString(schedulingstore.TeacherTimezoneField), rules, exceptions)
}

func lifecycleParticipantLessons(app core.App, current *core.Record) ([]regularcontract.ParticipantLesson, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName)
	if err != nil {
		return nil, err
	}
	result := make([]regularcontract.ParticipantLesson, 0)
	for _, row := range rows {
		if row.Id == current.Id || row.GetString(schedulingstore.ScheduleStateField) != "scheduled" || row.GetString("teacher") != current.GetString("teacher") && row.GetString("learner") != current.GetString("learner") {
			continue
		}
		interval, intervalErr := scheduling.NewInterval(recordInstant(row, schedulingstore.StartAtField), recordInstant(row, schedulingstore.EndAtField))
		if intervalErr != nil {
			return nil, errInvalid
		}
		result = append(result, regularcontract.ParticipantLesson{ID: row.Id, TeacherID: row.GetString("teacher"), LearnerID: row.GetString("learner"), Interval: interval, Status: scheduling.Scheduled})
	}
	return result, nil
}

func settlementForLesson(app core.App, lessonID string) ledger.SettlementState {
	rows, err := app.FindAllRecords(schedulingstore.ChargesCollectionName)
	if err != nil {
		return ""
	}
	for _, row := range rows {
		if row.GetString(schedulingstore.SourceTypeField) == "ad_hoc" && row.GetString(schedulingstore.SourceIDField) == lessonID {
			return ledger.SettlementState(row.GetString(schedulingstore.SettlementStateField))
		}
	}
	return ""
}

func recordInstant(row *core.Record, field string) time.Time {
	return row.GetDateTime(field).Time().UTC()
}
