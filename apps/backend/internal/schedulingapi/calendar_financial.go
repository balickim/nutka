// This file derives bounded teacher calendar counts from assignment-owned charge records.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func collectUnresolvedFinancial(app core.App, assignments, lessons []*core.Record, now time.Time, result *unresolvedDTO) error {
	if result == nil {
		return nil
	}
	owned := make(map[string]*time.Location, len(assignments))
	for _, assignment := range assignments {
		location, err := calendarAssignmentLocation(app, assignment)
		if err != nil {
			return err
		}
		owned[assignment.Id] = location
	}
	ended := endedLessonIDs(lessons, now)
	charges, err := app.FindAllRecords(schedulingstore.ChargesCollectionName)
	if err != nil {
		return err
	}
	for _, charge := range charges {
		addUnresolvedCharge(result, charge, owned, ended, now)
	}
	return nil
}

func addUnresolvedCharge(result *unresolvedDTO, charge *core.Record, owned map[string]*time.Location, ended map[string]bool, now time.Time) {
	location, isOwned := owned[charge.GetString(schedulingstore.AssignmentField)]
	if !isOwned {
		return
	}
	state := charge.GetString(schedulingstore.SettlementStateField)
	if charge.GetString(schedulingstore.SourceTypeField) == "ad_hoc" && state == "pending_settlement" && ended[charge.GetString(schedulingstore.SourceIDField)] {
		result.PendingSettlement++
	}
	if charge.GetInt(schedulingstore.CurrentAmountMinorField) > 0 && (state == "pending" || state == "intentionally_unpaid") {
		result.UnpaidCharges++
	}
	if state != "paid" && overdueOn(charge.GetString(schedulingstore.DueOnField), now, location) {
		result.OverdueCharges++
	}
}

func endedLessonIDs(rows []*core.Record, now time.Time) map[string]bool {
	result := make(map[string]bool)
	for _, row := range rows {
		if !row.GetDateTime(schedulingstore.EndAtField).Time().UTC().After(now) {
			result[row.Id] = true
		}
	}
	return result
}

func calendarAssignmentLocation(app core.App, assignment *core.Record) (*time.Location, error) {
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return nil, err
	}
	return time.LoadLocation(teacher.GetString(schedulingstore.TeacherTimezoneField))
}

func overdueOn(dueOn string, now time.Time, location *time.Location) bool {
	due, err := time.ParseInLocation("2006-01-02", dueOn, location)
	return err == nil && !now.In(location).Before(due.AddDate(0, 0, 1))
}
