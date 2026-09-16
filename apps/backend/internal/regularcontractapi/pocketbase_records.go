// This file maps PocketBase contract, occurrence, event, and amendment records to domain aggregates.
package regularcontractapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func contractFromRecord(app core.App, row *core.Record) (regularcontract.RegularContract, error) {
	assignmentValue, err := assignment(app, row.GetString(schedulingstore.AssignmentField))
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	teacher, err := app.FindRecordById("teachers", assignmentValue.TeacherID)
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	zone := teacher.GetString(schedulingstore.TeacherTimezoneField)
	start, err := parseLocalDate(row.GetString(schedulingstore.StartOnField), zone)
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	end, err := parseLocalDate(row.GetString(schedulingstore.EndOnField), zone)
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	minute, err := parseClock(row.GetString(schedulingstore.StartTimeField))
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	policy := businesspolicy.Current()
	snapshot := businesspolicy.CurrentSnapshot()
	if raw := row.GetString(schedulingstore.PolicySnapshotField); raw != "" {
		_ = json.Unmarshal([]byte(raw), &snapshot)
		policy = policyFromSnapshot(snapshot)
	}
	value := regularcontract.RegularContract{ID: row.Id, AssignmentID: assignmentValue.ID, TeacherID: assignmentValue.TeacherID, LearnerID: assignmentValue.LearnerID, TeacherTimezone: zone, StartOn: start, EndOn: end, EffectiveEndOn: end, Weekday: time.Weekday(row.GetInt(schedulingstore.WeekdayField)), StartMinute: minute, PriceMinor: int64(row.GetInt(schedulingstore.UnitPriceMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), Status: regularcontract.Status(row.GetString(schedulingstore.StatusField)), Policy: policy, PolicySnapshot: snapshot}
	if effective := row.GetString(schedulingstore.EffectiveEndOnField); effective != "" {
		value.EffectiveEndOn, _ = parseLocalDate(effective, zone)
	}
	if row.GetString(schedulingstore.NoticeAtField) != "" {
		value.NoticeAt = instant(row, schedulingstore.NoticeAtField)
	}
	return addContractChildren(app, value)
}

func policyFromSnapshot(s businesspolicy.PolicySnapshot) businesspolicy.Policy {
	p := businesspolicy.Current()
	p.Version, p.Currency = s.Version, s.Currency
	p.AdHocPrice.Minor, p.PackagePrice.Minor, p.RegularLessonPrice.Minor = s.AdHocPriceMinor, s.PackagePriceMinor, s.RegularLessonPriceMinor
	p.AdHocPrice.Currency, p.PackagePrice.Currency, p.RegularLessonPrice.Currency = s.Currency, s.Currency, s.Currency
	p.LessonDuration = time.Duration(s.LessonDurationMinutes) * time.Minute
	p.StartGrid = time.Duration(s.StartGridMinutes) * time.Minute
	p.ParticipantBuffer = time.Duration(s.ParticipantBufferMinutes) * time.Minute
	p.LearnerBookingMinimum = time.Duration(s.LearnerBookingMinimumHours) * time.Hour
	p.LearnerChangeCutoff = time.Duration(s.LearnerChangeCutoffHours) * time.Hour
	p.BookingHorizon = time.Duration(s.BookingHorizonDays) * 24 * time.Hour
	p.PackageTokenCount, p.PackageValidityDays, p.TeacherCancellationExtensionDays = s.PackageTokenCount, s.PackageValidityDays, s.TeacherCancellationExtensionDays
	p.ContractMonthlyReschedules, p.ContractFreeCancellations, p.ContractReplacementDays = s.ContractMonthlyReschedules, s.ContractFreeCancellations, s.ContractReplacementDays
	p.MonthlyPaymentDueDay, p.ContractEndMonth, p.ContractEndDay = s.MonthlyPaymentDueDay, time.Month(s.ContractEndMonth), s.ContractEndDay
	return p
}

func saveContract(app core.App, ctx context.Context, value regularcontract.RegularContract) error {
	row, err := app.FindRecordById(schedulingstore.RegularContractsCollectionName, value.ID)
	if err != nil {
		collection, collectionErr := app.FindCollectionByNameOrId(schedulingstore.RegularContractsCollectionName)
		if collectionErr != nil {
			return collectionErr
		}
		row = core.NewRecord(collection)
		row.Id = value.ID
	}
	zone, err := time.LoadLocation(value.TeacherTimezone)
	if err != nil {
		return err
	}
	row.Set(schedulingstore.AssignmentField, value.AssignmentID)
	row.Set(schedulingstore.StatusField, string(value.Status))
	row.Set(schedulingstore.StartOnField, value.StartOn.In(zone).Format("2006-01-02"))
	row.Set(schedulingstore.EndOnField, value.EndOn.In(zone).Format("2006-01-02"))
	row.Set(schedulingstore.WeekdayField, int(value.Weekday))
	row.Set(schedulingstore.StartTimeField, fmt.Sprintf("%02d:%02d", value.StartMinute/60, value.StartMinute%60))
	row.Set(schedulingstore.UnitPriceMinorField, value.PriceMinor)
	row.Set(schedulingstore.CurrencyField, value.Currency)
	row.Set(schedulingstore.EffectiveEndOnField, value.EffectiveEndOn.In(zone).Format("2006-01-02"))
	snapshot, _ := json.Marshal(value.PolicySnapshot)
	row.Set(schedulingstore.PolicyVersionField, value.PolicySnapshot.Version)
	row.Set(schedulingstore.PolicySnapshotField, snapshot)
	if !value.NoticeAt.IsZero() {
		row.Set(schedulingstore.NoticeAtField, value.NoticeAt.UTC().Format(time.RFC3339))
	}
	if err := app.Save(row); err != nil {
		return err
	}
	if err := saveAmendments(app, value); err != nil {
		return err
	}
	if err := saveOccurrences(app, ctx, value); err != nil {
		return err
	}
	return saveEvents(app, value)
}

func saveOccurrences(app core.App, _ context.Context, value regularcontract.RegularContract) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		return err
	}
	for _, occurrence := range value.Occurrences {
		rows, _ := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.And(dbx.HashExp{schedulingstore.ContractField: value.ID}, dbx.HashExp{schedulingstore.OriginalLocalDateField: occurrence.OriginalLocalDate}))
		var row *core.Record
		if len(rows) > 0 {
			row = rows[0]
		} else {
			row = core.NewRecord(collection)
			row.Id = occurrence.ID
		}
		row.Set("teacher", value.TeacherID)
		row.Set("learner", value.LearnerID)
		row.Set(schedulingstore.AssignmentField, value.AssignmentID)
		row.Set(schedulingstore.ContractField, value.ID)
		row.Set(schedulingstore.StartAtField, occurrence.Interval.Start.UTC().Format(time.RFC3339))
		row.Set(schedulingstore.EndAtField, occurrence.Interval.End.UTC().Format(time.RFC3339))
		row.Set(schedulingstore.DurationMinutesField, 45)
		row.Set(schedulingstore.StatusField, string(occurrence.ScheduleState))
		row.Set(schedulingstore.PlanTypeField, string(commercial.RegularContract))
		row.Set(schedulingstore.OriginalLocalDateField, occurrence.OriginalLocalDate)
		row.Set(schedulingstore.OriginalStartAtField, occurrence.OriginalStartAt.UTC().Format(time.RFC3339))
		row.Set(schedulingstore.PolicyVersionField, value.PolicySnapshot.Version)
		snapshot, _ := json.Marshal(value.PolicySnapshot)
		row.Set(schedulingstore.PolicySnapshotField, snapshot)
		row.Set(schedulingstore.UnitPriceMinorField, occurrence.UnitPriceMinor)
		row.Set(schedulingstore.CurrencyField, occurrence.Currency)
		row.Set(schedulingstore.ScheduleStateField, string(occurrence.ScheduleState))
		row.Set(schedulingstore.OutcomeField, occurrence.Outcome)
		row.Set(schedulingstore.BillingOutcomeField, string(occurrence.BillingOutcome))
		row.Set(schedulingstore.IndividuallyRescheduledField, occurrence.IndividuallyRescheduled)
		row.Set(schedulingstore.OmissionReasonField, occurrence.OmissionReason)
		if err := app.Save(row); err != nil {
			return err
		}
	}
	return nil
}
