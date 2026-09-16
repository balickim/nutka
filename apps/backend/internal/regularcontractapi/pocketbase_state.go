// This file loads availability, commercial overlap, packages, and participant lessons for contract commands.
package regularcontractapi

import (
	"encoding/json"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func lessonValues(app core.App, teacherID, learnerID, assignmentID string) ([]scheduling.Lesson, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return nil, err
	}
	result := make([]scheduling.Lesson, 0, len(rows))
	for _, row := range rows {
		start, end := instant(row, schedulingstore.StartAtField), instant(row, schedulingstore.EndAtField)
		interval, intervalErr := scheduling.NewInterval(start, end)
		if intervalErr != nil {
			continue
		}
		result = append(result, scheduling.Lesson{TeacherID: teacherID, LearnerID: learnerID, Interval: interval, Status: scheduling.LessonStatus(row.GetString(schedulingstore.ScheduleStateField))})
	}
	return result, nil
}

func availability(app core.App, now time.Time, zone, teacherID string) ([]scheduling.Interval, []scheduling.Interval, error) {
	rulesRows, err := app.FindAllRecords(schedulingstore.AvailabilityRulesCollectionName, dbx.HashExp{"teacher": teacherID})
	if err != nil {
		return nil, nil, err
	}
	rules := make([]scheduling.WeekdayRule, 0, len(rulesRows))
	for _, row := range rulesRows {
		start, startErr := parseRuleMinute(row.GetString(schedulingstore.StartTimeField))
		end, endErr := parseRuleMinute(row.GetString(schedulingstore.EndTimeField))
		if startErr != nil || endErr != nil {
			return nil, nil, scheduling.ErrInvalidRule
		}
		rules = append(rules, scheduling.WeekdayRule{Weekday: time.Weekday(row.GetInt(schedulingstore.WeekdayField)), StartMinute: start, EndMinute: end, Enabled: row.GetBool(schedulingstore.EnabledField)})
	}
	exceptionRows, err := app.FindAllRecords(schedulingstore.AvailabilityExceptionsCollectionName, dbx.HashExp{"teacher": teacherID})
	if err != nil {
		return nil, nil, err
	}
	exceptions := make([]scheduling.AvailabilityException, 0, len(exceptionRows))
	unavailable := make([]scheduling.Interval, 0)
	for _, row := range exceptionRows {
		if !row.GetBool(schedulingstore.EnabledField) {
			continue
		}
		interval, intervalErr := scheduling.NewInterval(instant(row, schedulingstore.StartAtField), instant(row, schedulingstore.EndAtField))
		if intervalErr != nil {
			return nil, nil, scheduling.ErrInvalidException
		}
		kind := scheduling.ExceptionKind(row.GetString(schedulingstore.KindField))
		exceptions = append(exceptions, scheduling.AvailabilityException{Interval: interval, Kind: kind})
		if kind == scheduling.UnavailableException {
			unavailable = append(unavailable, interval)
		}
	}
	start, end := now.UTC().AddDate(-2, 0, 0), now.UTC().AddDate(2, 0, 0)
	available, err := scheduling.EffectiveAvailability(start, end, zone, rules, exceptions)
	return available, unavailable, err
}
func parseRuleMinute(value string) (int, error) {
	if value == "24:00" {
		return 24 * 60, nil
	}
	return parseClock(value)
}
func overlapState(app core.App, assignmentValue Assignment, now time.Time, zone string) (commercial.OverlapState, error) {
	packages, err := packageValues(app, assignmentValue, zone)
	if err != nil {
		return commercial.OverlapState{}, err
	}
	activeContract, err := activeOverlapContract(app, assignmentValue.ID, zone)
	if err != nil {
		return commercial.OverlapState{}, err
	}
	refs, err := futureAdHocReferences(app, assignmentValue, now)
	if err != nil {
		return commercial.OverlapState{}, err
	}
	return commercial.OverlapState{Assignment: commercial.Assignment{ID: assignmentValue.ID, TeacherID: assignmentValue.TeacherID, LearnerID: assignmentValue.LearnerID}, Now: now, Contract: activeContract, ActiveContract: activeContract != nil, Packages: packages, FutureAdHocLessons: refs}, nil
}

func activeOverlapContract(app core.App, assignmentID, zone string) (*commercial.Contract, error) {
	contracts, err := app.FindAllRecords(schedulingstore.RegularContractsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return nil, err
	}
	for _, row := range contracts {
		if row.GetString(schedulingstore.StatusField) != string(regularcontract.Active) {
			continue
		}
		start, startErr := parseLocalDate(row.GetString(schedulingstore.StartOnField), zone)
		end, endErr := parseLocalDate(row.GetString(schedulingstore.EndOnField), zone)
		if startErr == nil && endErr == nil {
			return &commercial.Contract{ID: row.Id, AssignmentID: assignmentID, Status: commercial.ContractActive, StartOn: start, EndOn: end}, nil
		}
	}
	return nil, nil
}

func futureAdHocReferences(app core.App, assignmentValue Assignment, now time.Time) ([]commercial.LessonReference, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentValue.ID})
	if err != nil {
		return nil, err
	}
	refs := make([]commercial.LessonReference, 0)
	for _, row := range rows {
		if row.GetString(schedulingstore.PlanTypeField) != string(commercial.AdHoc) || row.GetString(schedulingstore.StatusField) != string(regularcontract.Scheduled) {
			continue
		}
		start := instant(row, schedulingstore.StartAtField)
		if start.After(now) {
			refs = append(refs, commercial.LessonReference{ID: row.Id, AssignmentID: assignmentValue.ID, TeacherID: assignmentValue.TeacherID, LearnerID: assignmentValue.LearnerID, Plan: commercial.AdHoc, StartAt: start})
		}
	}
	return refs, nil
}

func packageValues(app core.App, assignmentValue Assignment, zone string) ([]*commercial.Package, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentValue.ID})
	if err != nil {
		return nil, err
	}
	result := make([]*commercial.Package, 0, len(rows))
	for _, row := range rows {
		purchased, purchasedErr := parseLocalDate(row.GetString(schedulingstore.PurchasedOnField), zone)
		validThrough, validErr := parseLocalDate(row.GetString(schedulingstore.ValidThroughField), zone)
		if purchasedErr != nil || validErr != nil {
			return nil, commercial.ErrInvalidPurchaseDate
		}
		policy := businesspolicy.CurrentSnapshot()
		if raw := row.GetString(schedulingstore.PolicySnapshotField); raw != "" {
			_ = json.Unmarshal([]byte(raw), &policy)
		}
		value := &commercial.Package{ID: row.Id, Assignment: commercial.Assignment{ID: assignmentValue.ID, TeacherID: assignmentValue.TeacherID, LearnerID: assignmentValue.LearnerID}, Status: commercial.PackageStatus(row.GetString(schedulingstore.PackageStatusField)), PurchasedOn: purchased, ValidThrough: validThrough, Price: commercial.Money{Minor: int64(row.GetInt(schedulingstore.UnitPriceMinorField)), Currency: row.GetString(schedulingstore.CurrencyField)}, Policy: policy, TeacherZone: zone}
		tokenRows, tokenErr := app.FindAllRecords(schedulingstore.PackageTokensCollectionName, dbx.HashExp{schedulingstore.PackageField: row.Id})
		if tokenErr != nil {
			return nil, tokenErr
		}
		for _, tokenRow := range tokenRows {
			value.Tokens = append(value.Tokens, commercial.Token{ID: tokenRow.Id, Ordinal: tokenRow.GetInt(schedulingstore.OrdinalField), State: commercial.TokenState(tokenRow.GetString(schedulingstore.TokenStateField)), LessonID: tokenRow.GetString(schedulingstore.LessonField), LessonStart: lessonStart(app, tokenRow.GetString(schedulingstore.LessonField))})
		}
		result = append(result, value)
	}
	return result, nil
}

func lessonStart(app core.App, id string) time.Time {
	if id == "" {
		return time.Time{}
	}
	row, err := app.FindRecordById(schedulingstore.LessonsCollectionName, id)
	if err != nil {
		return time.Time{}
	}
	return instant(row, schedulingstore.StartAtField)
}
