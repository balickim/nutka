// This file provides the PocketBase repository and transaction adapter for regular contract commands.
package regularcontractapi

import (
	"context"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type PocketBaseRepository struct {
	app   core.App
	clock Clock
}

// NewPocketBaseRepository returns an adapter using the supplied PocketBase app.
func NewPocketBaseRepository(app core.App, clock Clock) *PocketBaseRepository {
	if clock == nil {
		clock = time.Now
	}
	return &PocketBaseRepository{app: app, clock: clock}
}

func (r *PocketBaseRepository) Assignment(_ context.Context, id string) (Assignment, error) {
	return assignment(r.app, id)
}
func (r *PocketBaseRepository) Contracts(_ context.Context, id string) ([]regularcontract.RegularContract, error) {
	rows, err := r.app.FindAllRecords(schedulingstore.RegularContractsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: id})
	if err != nil {
		return nil, err
	}
	result := make([]regularcontract.RegularContract, 0, len(rows))
	for _, row := range rows {
		value, valueErr := contractFromRecord(r.app, row)
		if valueErr != nil {
			return nil, valueErr
		}
		result = append(result, value)
	}
	return result, nil
}
func (r *PocketBaseRepository) Contract(_ context.Context, id string) (regularcontract.RegularContract, error) {
	row, err := r.app.FindRecordById(schedulingstore.RegularContractsCollectionName, id)
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	return contractFromRecord(r.app, row)
}
func (r *PocketBaseRepository) RunInTransaction(ctx context.Context, fn func(Transaction) error) error {
	return r.app.RunInTransaction(func(tx core.App) error { return fn(&pocketBaseTransaction{app: tx, clock: r.clock}) })
}

type pocketBaseTransaction struct {
	app   core.App
	clock Clock
}

func (t *pocketBaseTransaction) NewContractID(context.Context) (string, error) {
	return core.GenerateDefaultRandomId(), nil
}
func (t *pocketBaseTransaction) ActivationContext(ctx context.Context, id string) (ActivationContext, error) {
	assignmentValue, err := assignment(t.app, id)
	if err != nil {
		return ActivationContext{}, err
	}
	teacher, err := t.app.FindRecordById("teachers", assignmentValue.TeacherID)
	if err != nil {
		return ActivationContext{}, err
	}
	zone := teacher.GetString(schedulingstore.TeacherTimezoneField)
	if _, err := time.LoadLocation(zone); err != nil || zone == "" {
		return ActivationContext{}, scheduling.ErrInvalidTimezone
	}
	now := t.clock().UTC()
	policy := businesspolicy.Current()
	available, unavailable, err := availability(t.app, now, zone, assignmentValue.TeacherID)
	if err != nil {
		return ActivationContext{}, err
	}
	lessons, err := lessonValues(t.app, assignmentValue.TeacherID, assignmentValue.LearnerID, id)
	if err != nil {
		return ActivationContext{}, err
	}
	commercialState, err := overlapState(t.app, assignmentValue, now, zone)
	if err != nil {
		return ActivationContext{}, err
	}
	contractID, err := t.NewContractID(ctx)
	if err != nil {
		return ActivationContext{}, err
	}
	return ActivationContext{Assignment: assignmentValue, ContractID: contractID, TeacherTimezone: zone, Policy: policy, Availability: available, Unavailable: unavailable, Lessons: lessons, CommercialState: commercialState}, nil
}
func (t *pocketBaseTransaction) ContractContext(_ context.Context, id string) (ContractContext, error) {
	row, err := t.app.FindRecordById(schedulingstore.RegularContractsCollectionName, id)
	if err != nil {
		return ContractContext{}, err
	}
	contract, err := contractFromRecord(t.app, row)
	if err != nil {
		return ContractContext{}, err
	}
	zone := contract.TeacherTimezone
	now := t.clock().UTC()
	available, unavailable, err := availability(t.app, now, zone, contract.TeacherID)
	if err != nil {
		return ContractContext{}, err
	}
	lessons, err := lessonValues(t.app, contract.TeacherID, contract.LearnerID, contract.AssignmentID)
	if err != nil {
		return ContractContext{}, err
	}
	return ContractContext{Contract: contract, Policy: businesspolicy.Current(), Availability: available, Unavailable: unavailable, Lessons: lessons}, nil
}
func (t *pocketBaseTransaction) ConvertContractLessons(_ context.Context, contractID string, lessons []commercial.LessonReference) error {
	for _, lesson := range lessons {
		row, err := t.app.FindRecordById(schedulingstore.LessonsCollectionName, lesson.ID)
		if err != nil {
			return err
		}
		row.Set(schedulingstore.PlanTypeField, string(commercial.RegularContract))
		row.Set(schedulingstore.ContractField, contractID)
		row.Set(schedulingstore.PolicyVersionField, businesspolicy.CurrentVersion)
		if err := t.app.Save(row); err != nil {
			return err
		}
	}
	return nil
}
func (t *pocketBaseTransaction) SaveContract(ctx context.Context, value regularcontract.RegularContract) error {
	return SaveContract(t.app, value)
}

func assignment(app core.App, id string) (Assignment, error) {
	row, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, id)
	if err != nil {
		return Assignment{}, err
	}
	return Assignment{ID: row.Id, TeacherID: row.GetString("teacher"), LearnerID: row.GetString("learner"), Active: row.GetBool(schedulingstore.ActiveField)}, nil
}
func parseLocalDate(value, zone string) (time.Time, error) {
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation("2006-01-02", value, location)
}
func parseClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}
func instant(row *core.Record, field string) time.Time { return row.GetDateTime(field).Time().UTC() }
