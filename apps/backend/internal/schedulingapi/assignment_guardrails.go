// This file loads assignment obligations for the domain deactivation guard.
// It keeps record queries assignment-scoped and leaves the blocking rule in regularcontract.
package schedulingapi

import (
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func assignmentObligations(app core.App, assignmentID string, now time.Time) (regularcontract.ObligationSet, error) {
	obligations := regularcontract.ObligationSet{}
	active, err := hasAssignmentActiveContract(app, assignmentID, now)
	if err != nil {
		return obligations, err
	}
	obligations.ActiveContract = active
	if err := addPackageObligations(app, assignmentID, &obligations); err != nil {
		return obligations, err
	}
	if err := addLessonObligations(app, assignmentID, now, &obligations); err != nil {
		return obligations, err
	}
	return obligations, nil
}

func hasAssignmentActiveContract(app core.App, assignmentID string, now time.Time) (bool, error) {
	contracts, err := app.FindAllRecords(schedulingstore.RegularContractsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return false, err
	}
	for _, contract := range contracts {
		active, activeErr := assignmentContractActiveAt(app, contract, now)
		if activeErr != nil {
			return false, activeErr
		}
		if active {
			return true, nil
		}
	}
	return false, nil
}

func addPackageObligations(app core.App, assignmentID string, obligations *regularcontract.ObligationSet) error {
	packages, err := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return err
	}
	openPackages := make(map[string]bool)
	for _, pkg := range packages {
		if pkg.GetString(schedulingstore.AssignmentField) == assignmentID && pkg.GetString(schedulingstore.PackageStatusField) == string(commercial.PackageOpen) {
			openPackages[pkg.Id] = true
		}
	}
	if len(openPackages) > 0 {
		for packageID := range openPackages {
			tokens, tokenErr := app.FindAllRecords(schedulingstore.PackageTokensCollectionName, dbx.HashExp{schedulingstore.PackageField: packageID})
			if tokenErr != nil {
				return tokenErr
			}
			for _, token := range tokens {
				if token.GetString(schedulingstore.TokenStateField) == string(commercial.TokenAvailable) {
					obligations.AvailablePackageTokens++
				}
			}
		}
	}
	return nil
}

func addLessonObligations(app core.App, assignmentID string, now time.Time, obligations *regularcontract.ObligationSet) error {
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return err
	}
	for _, lesson := range lessons {
		if lesson.GetString(schedulingstore.AssignmentField) != assignmentID || !futureScheduledLesson(lesson, now) {
			continue
		}
		obligations.FutureScheduledLessons++
		if lesson.GetString(schedulingstore.PlanTypeField) == string(commercial.PackagePlan) {
			obligations.ReservedPackageLessons++
		}
	}
	return nil
}

func assignmentContractActiveAt(app core.App, row *core.Record, now time.Time) (bool, error) {
	status := regularcontract.Status(row.GetString(schedulingstore.StatusField))
	if status != regularcontract.Active && status != regularcontract.NoticeGiven {
		return false, nil
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, row.GetString(schedulingstore.AssignmentField))
	if err != nil {
		return false, err
	}
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return false, err
	}
	zone := teacher.GetString(schedulingstore.TeacherTimezoneField)
	start, err := parseAssignmentDate(row.GetString(schedulingstore.StartOnField), zone)
	if err != nil {
		return false, err
	}
	endValue := row.GetString(schedulingstore.EffectiveEndOnField)
	if endValue == "" {
		endValue = row.GetString(schedulingstore.EndOnField)
	}
	end, err := parseAssignmentDate(endValue, zone)
	if err != nil {
		return false, err
	}
	contract := regularcontract.RegularContract{Status: status, TeacherTimezone: zone, StartOn: start, EffectiveEndOn: end, Policy: businesspolicy.Current()}
	return contract.ActiveAt(now), nil
}

func parseAssignmentDate(value, zone string) (time.Time, error) {
	if value == "" || zone == "" {
		return time.Time{}, fmt.Errorf("invalid contract date")
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation("2006-01-02", value, location)
}

func futureScheduledLesson(lesson *core.Record, now time.Time) bool {
	if lesson.GetString(schedulingstore.StatusField) != string(scheduling.Scheduled) {
		return false
	}
	if state := lesson.GetString(schedulingstore.ScheduleStateField); state != "" && state != string(scheduling.Scheduled) {
		return false
	}
	start := lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()
	return !start.IsZero() && start.After(now.UTC())
}
