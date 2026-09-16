// This file loads package, lesson, and contract state required for package transition decisions.
package commercialapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func purchaseOverlapState(app core.App, assignment *core.Record, now time.Time) (commercial.OverlapState, error) {
	packages, err := loadOverlapPackages(app, assignment.Id)
	if err != nil {
		return commercial.OverlapState{}, err
	}
	lessons, err := loadFutureAdHocLessons(app, assignment, now)
	if err != nil {
		return commercial.OverlapState{}, err
	}
	contract, err := loadActiveContract(app, assignment, now)
	if err != nil {
		return commercial.OverlapState{}, err
	}
	value := commercial.Assignment{ID: assignment.Id, TeacherID: assignment.GetString("teacher"), LearnerID: assignment.GetString("learner")}
	return commercial.OverlapState{Assignment: value, Now: now, ActiveContract: contract != nil, Contract: contract, Packages: packages, FutureAdHocLessons: lessons}, nil
}

func loadOverlapPackages(app core.App, assignmentID string) ([]*commercial.Package, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return nil, err
	}
	values := make([]*commercial.Package, 0, len(rows))
	for _, row := range rows {
		value, err := packageFromRecords(app, row)
		if err != nil {
			return nil, err
		}
		values = append(values, &value)
	}
	return values, nil
}

func loadFutureAdHocLessons(app core.App, assignment *core.Record, now time.Time) ([]commercial.LessonReference, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignment.Id})
	if err != nil {
		return nil, err
	}
	refs := make([]commercial.LessonReference, 0)
	for _, row := range rows {
		if row.GetString(schedulingstore.StatusField) != "scheduled" || row.GetString(schedulingstore.PlanTypeField) != string(commercial.AdHoc) || !row.GetDateTime(schedulingstore.StartAtField).Time().UTC().After(now) {
			continue
		}
		refs = append(refs, commercial.LessonReference{ID: row.Id, AssignmentID: assignment.Id, TeacherID: assignment.GetString("teacher"), LearnerID: assignment.GetString("learner"), Plan: commercial.AdHoc, StartAt: row.GetDateTime(schedulingstore.StartAtField).Time().UTC()})
	}
	return refs, nil
}

func loadActiveContract(app core.App, assignment *core.Record, now time.Time) (*commercial.Contract, error) {
	rows, err := app.FindAllRecords(schedulingstore.RegularContractsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignment.Id})
	if err != nil {
		return nil, err
	}
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(teacher.GetString("timezone"))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		candidate, ok := activeContractCandidate(row, assignment.Id)
		if ok && candidate.ActiveAt(now.In(location)) {
			return &candidate, nil
		}
	}
	return nil, nil
}

func activeContractCandidate(row *core.Record, assignmentID string) (commercial.Contract, bool) {
	status := row.GetString(schedulingstore.StatusField)
	if status != "active" && status != "notice_given" {
		return commercial.Contract{}, false
	}
	startOn, startErr := time.Parse("2006-01-02", row.GetString(schedulingstore.StartOnField))
	endText := row.GetString(schedulingstore.EffectiveEndOnField)
	if endText == "" {
		endText = row.GetString(schedulingstore.EndOnField)
	}
	endOn, endErr := time.Parse("2006-01-02", endText)
	if startErr != nil || endErr != nil {
		return commercial.Contract{}, false
	}
	return commercial.Contract{ID: row.Id, AssignmentID: assignmentID, Status: commercial.ContractActive, StartOn: startOn, EndOn: endOn}, true
}
