// This file persists fixed-duration booking projections, entitlements, obligations, and immutable transition events.
package schedulingapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func persistBooking(app core.App, assignment *core.Record, command bookingCommand, interval scheduling.Interval, decision commercial.EligibilityDecision, state bookingState) (*core.Record, error) {
	policy := businesspolicy.Current()
	snapshot, price, err := bookingCommercialValues(policy, decision, state)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return nil, errInvalid
	}
	collection, err := app.FindCollectionByNameOrId(schedulingstore.LessonsCollectionName)
	if err != nil {
		return nil, errInvalid
	}
	lesson := core.NewRecord(collection)
	teacherID, learnerID := assignment.GetString("teacher"), assignment.GetString("learner")
	lesson.Set("teacher", teacherID)
	lesson.Set("learner", learnerID)
	lesson.Set(schedulingstore.AssignmentField, assignment.Id)
	lesson.Set(schedulingstore.StartAtField, interval.Start.Format(time.RFC3339))
	lesson.Set(schedulingstore.EndAtField, interval.End.Format(time.RFC3339))
	lesson.Set(schedulingstore.DurationMinutesField, businesspolicy.LessonDurationMinutes)
	lesson.Set(schedulingstore.StatusField, string(scheduling.Scheduled))
	lesson.Set(schedulingstore.PlanTypeField, string(decision.Plan))
	lesson.Set(schedulingstore.OriginalLocalDateField, interval.Start.In(state.teacherZone).Format("2006-01-02"))
	lesson.Set(schedulingstore.OriginalStartAtField, interval.Start.Format(time.RFC3339))
	lesson.Set(schedulingstore.PolicyVersionField, snapshot.Version)
	lesson.Set(schedulingstore.PolicySnapshotField, encoded)
	lesson.Set(schedulingstore.UnitPriceMinorField, price.Minor)
	lesson.Set(schedulingstore.CurrencyField, price.Currency)
	lesson.Set(schedulingstore.ScheduleStateField, "scheduled")
	if err := app.Save(lesson); err != nil {
		return nil, fmt.Errorf("save lesson: %w", err)
	}
	if err := persistBookingEntitlement(app, lesson, assignment.Id, price, decision, command); err != nil {
		return nil, err
	}
	if err := saveLessonEvent(app, lesson, "created", string(command.Actor.Role), command.Actor.ID, command.Now, time.Time{}, time.Time{}, interval.Start, interval.End, policy.LessonDuration, 0); err != nil {
		return nil, err
	}
	event, err := history.NewLessonCreated(history.EventInput{AggregateType: "lesson", AggregateID: lesson.Id, AssignmentID: assignment.Id, Actor: command.Actor, EventAt: command.Now, NewState: map[string]any{"plan_type": string(decision.Plan), "start_at": interval.Start.Format(time.RFC3339), "end_at": interval.End.Format(time.RFC3339)}})
	if err != nil {
		return nil, err
	}
	if err := (history.Storage{}).Append(app, event); err != nil {
		return nil, err
	}
	return lesson, nil
}

func bookingCommercialValues(policy businesspolicy.Policy, decision commercial.EligibilityDecision, state bookingState) (businesspolicy.PolicySnapshot, commercial.Money, error) {
	if decision.Plan == commercial.PackagePlan {
		packageValue := state.packageByID(decision.PackageID)
		if packageValue == nil {
			return businesspolicy.PolicySnapshot{}, commercial.Money{}, errPackageUnavailable
		}
		return packageValue.Policy, commercial.Money{Minor: 0, Currency: policy.Currency}, nil
	}
	if decision.Plan == commercial.RegularContract {
		return policy.Snapshot(), policy.RegularLessonPrice, nil
	}
	return policy.Snapshot(), policy.AdHocPrice, nil
}

func persistBookingEntitlement(app core.App, lesson *core.Record, assignmentID string, price commercial.Money, decision commercial.EligibilityDecision, command bookingCommand) error {
	if decision.Plan == commercial.PackagePlan {
		return reservePackageToken(app, lesson, decision.PackageID, command)
	}
	if decision.Plan == commercial.AdHoc {
		return saveAdHocObligation(app, lesson, assignmentID, price, command)
	}
	return nil
}

func saveAdHocObligation(app core.App, lesson *core.Record, assignmentID string, price commercial.Money, command bookingCommand) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.ChargesCollectionName)
	if err != nil {
		return errInvalid
	}
	charge := core.NewRecord(collection)
	charge.Set(schedulingstore.AssignmentField, assignmentID)
	charge.Set(schedulingstore.SourceTypeField, string(commercial.AdHoc))
	charge.Set(schedulingstore.SourceIDField, lesson.Id)
	charge.Set(schedulingstore.OriginalAmountMinorField, price.Minor)
	charge.Set(schedulingstore.CurrentAmountMinorField, price.Minor)
	charge.Set(schedulingstore.CurrencyField, price.Currency)
	charge.Set(schedulingstore.SettlementStateField, string(ledger.PendingSettlement))
	charge.Set(schedulingstore.CreatedAtField, command.Now.Format(time.RFC3339))
	if err := app.Save(charge); err != nil {
		return errConflict
	}
	entries, err := app.FindCollectionByNameOrId(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return errInvalid
	}
	entry := core.NewRecord(entries)
	entry.Set(schedulingstore.AssignmentField, assignmentID)
	entry.Set(schedulingstore.EntryTypeField, string(ledger.EntryChargeCreated))
	entry.Set(schedulingstore.AmountMinorField, price.Minor)
	entry.Set(schedulingstore.CurrencyField, price.Currency)
	entry.Set(schedulingstore.SourceTypeField, string(commercial.AdHoc))
	entry.Set(schedulingstore.SourceIDField, lesson.Id)
	entry.Set(schedulingstore.RelatedLessonField, lesson.Id)
	entry.Set(schedulingstore.ActorRoleField, string(command.Actor.Role))
	entry.Set(schedulingstore.ActorIDField, command.Actor.ID)
	entry.Set(schedulingstore.EventAtField, command.Now.Format(time.RFC3339))
	if err := app.Save(entry); err != nil {
		return errConflict
	}
	event, err := history.NewChargeCreated(history.EventInput{AggregateType: "charge", AggregateID: charge.Id, AssignmentID: assignmentID, Actor: command.Actor, EventAt: command.Now, RelatedIDs: map[string]string{"lesson": lesson.Id}, NewState: map[string]any{"amount_minor": price.Minor, "settlement_state": string(ledger.PendingSettlement)}})
	if err != nil {
		return err
	}
	return (history.Storage{}).Append(app, event)
}

func reservePackageToken(app core.App, lesson *core.Record, packageID string, command bookingCommand) error {
	packageValue, err := loadPackage(app, packageID)
	if err != nil {
		return err
	}
	token, err := packageValue.Reserve(commercial.LessonReference{ID: lesson.Id, AssignmentID: lesson.GetString(schedulingstore.AssignmentField), TeacherID: lesson.GetString("teacher"), LearnerID: lesson.GetString("learner"), StartAt: lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()}, command.Now)
	if err != nil {
		return mapEligibilityError(err)
	}
	tokenRecord, err := app.FindRecordById(schedulingstore.PackageTokensCollectionName, token.ID)
	if err != nil {
		return errConflict
	}
	tokenRecord.Set(schedulingstore.TokenStateField, string(commercial.TokenReserved))
	tokenRecord.Set(schedulingstore.LessonField, lesson.Id)
	tokenRecord.Set(schedulingstore.ChangedAtField, command.Now.Format(time.RFC3339))
	if err := app.Save(tokenRecord); err != nil {
		return fmt.Errorf("save package token: %w", err)
	}
	lesson.Set(schedulingstore.PackageTokenField, token.ID)
	if err := app.Save(lesson); err != nil {
		return fmt.Errorf("link package token: %w", err)
	}
	event, err := history.NewPackageTokenReserved(history.EventInput{AggregateType: "package_token", AggregateID: token.ID, AssignmentID: lesson.GetString(schedulingstore.AssignmentField), Actor: command.Actor, EventAt: command.Now, RelatedIDs: map[string]string{"lesson": lesson.Id, "package": packageID}, NewState: map[string]any{"state": string(commercial.TokenReserved), "lesson": lesson.Id}})
	if err != nil {
		return err
	}
	return (history.Storage{}).Append(app, event)
}
