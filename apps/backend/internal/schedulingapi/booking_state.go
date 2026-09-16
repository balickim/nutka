// This file adapts assignment-scoped commercial records to pure eligibility values for booking transactions.
package schedulingapi

import (
	"encoding/json"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type bookingState struct {
	contract    *commercial.Contract
	packages    []*commercial.Package
	teacherZone *time.Location
}

func (s bookingState) packageByID(id string) *commercial.Package {
	for _, value := range s.packages {
		if value != nil && value.ID == id {
			return value
		}
	}
	return nil
}

func loadBookingState(app core.App, assignment *core.Record, now, lessonStart time.Time) (bookingState, error) {
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return bookingState{}, errForbidden
	}
	zone, err := time.LoadLocation(teacher.GetString(schedulingstore.TeacherTimezoneField))
	if err != nil {
		return bookingState{}, errInvalid
	}
	state := bookingState{teacherZone: zone}
	state.contract, err = activeBookingContract(app, assignment.Id, now, lessonStart, zone)
	if err != nil {
		return bookingState{}, err
	}
	state.packages, err = bookingPackages(app, assignment.Id, zone)
	if err != nil {
		return bookingState{}, err
	}
	return state, nil
}

func activeBookingContract(app core.App, assignmentID string, now, lessonStart time.Time, zone *time.Location) (*commercial.Contract, error) {
	rows, err := app.FindAllRecords(schedulingstore.RegularContractsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		value, err := bookingContract(row, assignmentID)
		if err != nil {
			return nil, err
		}
		if value != nil && (contractActiveOn(value, now, zone) || contractActiveOn(value, lessonStart, zone)) {
			return value, nil
		}
	}
	return nil, nil
}

func bookingContract(row *core.Record, assignmentID string) (*commercial.Contract, error) {
	if row.GetString(schedulingstore.StatusField) != "active" {
		return nil, nil
	}
	startOn, startErr := time.Parse("2006-01-02", row.GetString(schedulingstore.StartOnField))
	endOn, endErr := time.Parse("2006-01-02", row.GetString(schedulingstore.EndOnField))
	if startErr != nil || endErr != nil {
		return nil, errInvalid
	}
	return &commercial.Contract{ID: row.Id, AssignmentID: assignmentID, Status: commercial.ContractActive, StartOn: startOn, EndOn: endOn}, nil
}

func bookingPackages(app core.App, assignmentID string, zone *time.Location) ([]*commercial.Package, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID})
	if err != nil {
		return nil, err
	}
	result := make([]*commercial.Package, 0, len(rows))
	for _, row := range rows {
		value, err := loadPackageFromRecord(app, row, zone)
		if err != nil {
			return nil, err
		}
		result = append(result, &value)
	}
	return result, nil
}

func contractActiveOn(value *commercial.Contract, instant time.Time, zone *time.Location) bool {
	local := instant.In(zone)
	date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	return value.ActiveAt(date)
}

func loadPackage(app core.App, id string) (commercial.Package, error) {
	row, err := app.FindRecordById(schedulingstore.LessonPackagesCollectionName, id)
	if err != nil {
		return commercial.Package{}, errPackageUnavailable
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, row.GetString(schedulingstore.AssignmentField))
	if err != nil {
		return commercial.Package{}, errForbidden
	}
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return commercial.Package{}, errForbidden
	}
	zone, err := time.LoadLocation(teacher.GetString(schedulingstore.TeacherTimezoneField))
	if err != nil {
		return commercial.Package{}, errInvalid
	}
	return loadPackageFromRecord(app, row, zone)
}

func loadPackageFromRecord(app core.App, row *core.Record, zone *time.Location) (commercial.Package, error) {
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, row.GetString(schedulingstore.AssignmentField))
	if err != nil {
		return commercial.Package{}, errForbidden
	}
	purchased, err := time.Parse("2006-01-02", row.GetString(schedulingstore.PurchasedOnField))
	if err != nil {
		return commercial.Package{}, errInvalid
	}
	validThrough, err := time.Parse("2006-01-02", row.GetString(schedulingstore.ValidThroughField))
	if err != nil {
		return commercial.Package{}, errInvalid
	}
	snapshot, err := bookingPolicySnapshot(row)
	if err != nil {
		return commercial.Package{}, errInvalid
	}
	rows, err := app.FindAllRecords(schedulingstore.PackageTokensCollectionName, dbx.HashExp{schedulingstore.PackageField: row.Id})
	if err != nil {
		return commercial.Package{}, err
	}
	tokens, err := bookingTokens(app, rows)
	if err != nil {
		return commercial.Package{}, err
	}
	return commercial.Package{ID: row.Id, Assignment: commercial.Assignment{ID: assignment.Id, TeacherID: assignment.GetString("teacher"), LearnerID: assignment.GetString("learner")}, Status: commercial.PackageStatus(row.GetString(schedulingstore.PackageStatusField)), PurchasedOn: purchased, ValidThrough: validThrough, ClosedAt: row.GetDateTime(schedulingstore.ClosedAtField).Time().UTC(), Price: commercial.Money{Minor: int64(row.GetInt(schedulingstore.UnitPriceMinorField)), Currency: row.GetString(schedulingstore.CurrencyField)}, Policy: snapshot, TeacherZone: zone.String(), Tokens: tokens}, nil
}

func bookingPolicySnapshot(row *core.Record) (businesspolicy.PolicySnapshot, error) {
	var snapshot businesspolicy.PolicySnapshot
	encoded, err := json.Marshal(row.Get(schedulingstore.PolicySnapshotField))
	if err != nil {
		return snapshot, err
	}
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return snapshot, err
	}
	return snapshot, snapshot.Validate()
}

func bookingTokens(app core.App, rows []*core.Record) ([]commercial.Token, error) {
	result := make([]commercial.Token, 0, len(rows))
	for _, token := range rows {
		lessonStart, err := packageTokenLessonStart(app, token)
		if err != nil {
			return nil, err
		}
		result = append(result, commercial.Token{ID: token.Id, Ordinal: token.GetInt(schedulingstore.OrdinalField), State: commercial.TokenState(token.GetString(schedulingstore.TokenStateField)), LessonID: token.GetString(schedulingstore.LessonField), LessonStart: lessonStart})
	}
	return result, nil
}

func packageTokenLessonStart(app core.App, token *core.Record) (time.Time, error) {
	lessonID := token.GetString(schedulingstore.LessonField)
	if lessonID == "" {
		return time.Time{}, nil
	}
	lesson, err := app.FindRecordById(schedulingstore.LessonsCollectionName, lessonID)
	if err != nil {
		return time.Time{}, errInvalid
	}
	return lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC(), nil
}
