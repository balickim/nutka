// This file persists complete plan-aware lesson decisions and their entitlement, financial, and history projections atomically.
package schedulingapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/commercialapi"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/lesson"
	"github.com/balickim/nutka/apps/backend/internal/regularcontractapi"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func persistLifecycleDecision(app core.App, row *core.Record, value lesson.Decision, now time.Time) error {
	priorStart, priorEnd := recordInstant(row, schedulingstore.StartAtField), recordInstant(row, schedulingstore.EndAtField)
	applyLessonDecision(row, value, now)
	if err := app.Save(row); err != nil {
		return err
	}
	if err := persistPlanProjection(app, row, value, now); err != nil {
		return err
	}
	if err := appendLifecycleEvents(app, value.Events); err != nil {
		return err
	}
	return persistLegacyLifecycleEvent(app, row, value, now, priorStart, priorEnd)
}

func applyLessonDecision(row *core.Record, value lesson.Decision, now time.Time) {
	row.Set(schedulingstore.StartAtField, value.Lesson.Interval.Start.UTC())
	row.Set(schedulingstore.EndAtField, value.Lesson.Interval.End.UTC())
	row.Set(schedulingstore.DurationMinutesField, int(value.Lesson.Interval.Duration()/time.Minute))
	row.Set(schedulingstore.StatusField, string(value.Lesson.ScheduleState))
	row.Set(schedulingstore.ScheduleStateField, string(value.Lesson.ScheduleState))
	row.Set(schedulingstore.OutcomeField, string(value.Lesson.Outcome))
	if len(value.Events) > 0 && value.Events[0].Type == history.LessonCancelled {
		row.Set(schedulingstore.CancellationInitiatorRoleField, string(value.Events[0].Actor.Role))
		row.Set(schedulingstore.CancellationInitiatorIDField, value.Events[0].Actor.ID)
		row.Set(schedulingstore.CancelledAtField, now.UTC())
	}
	if value.Lesson.Outcome != "" && value.Lesson.Outcome != "awaiting_outcome" {
		row.Set(schedulingstore.OutcomeAtField, now.UTC())
	}

}

func persistPlanProjection(app core.App, row *core.Record, value lesson.Decision, now time.Time) error {
	if value.Package != nil {
		if err := commercialapi.SavePackageProjection(app, value.Package, now); err != nil {
			return err
		}
	}
	if value.Contract != nil {
		if err := regularcontractapi.SaveContract(app, *value.Contract); err != nil {
			return err
		}
	}
	if value.Lesson.Plan == commercial.AdHoc && value.Lesson.Settlement == "not_applicable" {
		if len(value.Events) == 0 {
			return errInvalid
		}
		if err := persistNotApplicable(app, row, value.Events[0].Actor, now); err != nil {
			return err
		}
	}
	return nil
}

func appendLifecycleEvents(app core.App, events []history.Event) error {
	for _, event := range events {
		if err := (history.Storage{}).Append(app, event); err != nil {
			return err
		}
	}
	return nil
}

func persistLegacyLifecycleEvent(app core.App, row *core.Record, value lesson.Decision, now, priorStart, priorEnd time.Time) error {
	if len(value.Events) == 0 {
		return nil
	}
	kind := ""
	newStart, newEnd := value.Lesson.Interval.Start, value.Lesson.Interval.End
	switch value.Events[0].Type {
	case history.LessonRescheduled:
		kind = "rescheduled"
	case history.LessonCancelled:
		kind, newStart, newEnd = "cancelled", time.Time{}, time.Time{}
	default:
		return nil
	}
	duration := value.Lesson.Interval.Duration()
	return saveLessonEvent(app, row, kind, string(value.Events[0].Actor.Role), value.Events[0].Actor.ID, now, priorStart, priorEnd, newStart, newEnd, duration, int(priorEnd.Sub(priorStart)/time.Minute))
}

func persistNotApplicable(app core.App, lessonRow *core.Record, actor history.Actor, now time.Time) error {
	rows, err := app.FindAllRecords(schedulingstore.ChargesCollectionName)
	if err != nil {
		return err
	}
	for _, charge := range rows {
		if charge.GetString(schedulingstore.SourceTypeField) != "ad_hoc" || charge.GetString(schedulingstore.SourceIDField) != lessonRow.Id {
			continue
		}
		charge.Set(schedulingstore.SettlementStateField, string(ledger.NotApplicable))
		charge.Set(schedulingstore.CurrentAmountMinorField, 0)
		if err := app.Save(charge); err != nil {
			return err
		}
		return saveNotApplicableEntry(app, lessonRow, charge, actor, now)
	}
	return errInvalid
}

func saveNotApplicableEntry(app core.App, lessonRow, charge *core.Record, actor history.Actor, now time.Time) error {
	collection, err := app.FindCollectionByNameOrId(schedulingstore.FinancialEntriesCollectionName)
	if err != nil {
		return err
	}
	entry := core.NewRecord(collection)
	entry.Set(schedulingstore.AssignmentField, lessonRow.GetString(schedulingstore.AssignmentField))
	entry.Set(schedulingstore.ChargeField, charge.Id)
	entry.Set(schedulingstore.EntryTypeField, string(ledger.EntrySettlementNAA))
	entry.Set(schedulingstore.AmountMinorField, 0)
	entry.Set(schedulingstore.CurrencyField, lessonRow.GetString(schedulingstore.CurrencyField))
	entry.Set(schedulingstore.SourceTypeField, string(commercial.AdHoc))
	entry.Set(schedulingstore.SourceIDField, lessonRow.Id)
	entry.Set(schedulingstore.RelatedLessonField, lessonRow.Id)
	entry.Set(schedulingstore.ActorRoleField, string(actor.Role))
	entry.Set(schedulingstore.ActorIDField, actor.ID)
	entry.Set(schedulingstore.EventAtField, now.UTC())
	return app.Save(entry)
}
