// This file exposes explicit teacher outcome and reasoned entitlement correction commands for ended lessons.
package schedulingapi

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/history"
	lessondomain "github.com/balickim/nutka/apps/backend/internal/lesson"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type outcomeInput struct {
	Outcome string `json:"outcome"`
}

func recordLessonOutcome(e *core.RequestEvent, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	var input outcomeInput
	if err := bindBody(e, &input); err != nil {
		return handleError(e, err)
	}
	if input.Outcome != string(domain.Completed) && input.Outcome != string(domain.LearnerNoShow) {
		return respond(e, http.StatusBadRequest, "invalid_outcome", "The lesson outcome is invalid.")
	}
	now := clock().UTC()
	var result *core.Record
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		row, findErr := findAuthorizedLesson(tx, e.Request.PathValue("id"), "teacher", teacher.Id)
		if findErr != nil {
			return findErr
		}
		command, commandErr := lifecycleCommand(tx, row, history.Actor{Role: history.TeacherActor, ID: teacher.Id}, now, time.Time{})
		if commandErr != nil {
			return commandErr
		}
		decision, decisionErr := lessondomain.RecordOutcome(command, domain.Outcome(input.Outcome))
		if decisionErr != nil {
			return decisionErr
		}
		if err := persistLifecycleDecision(tx, row, decision, now); err != nil {
			return err
		}
		result = row
		return nil
	})
	if err != nil {
		return lifecycleError(e, err)
	}
	return e.JSON(http.StatusOK, lessonValueAt(result, now))
}

func latestContractEntitlementEvent(app core.App, lesson *core.Record) (string, error) {
	rows, err := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		return "", err
	}
	var selected *core.Record
	for _, row := range rows {
		if row.GetString(schedulingstore.AggregateTypeField) != "contract" || row.GetString(schedulingstore.AggregateIDField) != lesson.GetString(schedulingstore.ContractField) || relatedOccurrence(row) != lesson.Id {
			continue
		}
		if selected == nil || recordInstant(row, schedulingstore.EventAtField).After(recordInstant(selected, schedulingstore.EventAtField)) {
			selected = row
		}
	}
	if selected == nil || selected.GetString(schedulingstore.LegacyEventIDField) == "" {
		return "", lessondomain.ErrCorrectionTarget
	}
	return selected.GetString(schedulingstore.LegacyEventIDField), nil
}

func relatedOccurrence(row *core.Record) string {
	encoded, err := json.Marshal(row.Get(schedulingstore.RelatedIDsField))
	if err != nil {
		return ""
	}
	var value map[string]string
	if json.Unmarshal(encoded, &value) != nil {
		return ""
	}
	return value["occurrence_id"]
}

type lessonCorrectionInput struct {
	Correction string `json:"correction"`
	Reason     string `json:"reason"`
}

func correctLessonEntitlement(e *core.RequestEvent, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	var input lessonCorrectionInput
	if err := bindBody(e, &input); err != nil {
		return handleError(e, err)
	}
	if strings.TrimSpace(input.Reason) == "" || input.Correction != "restore_entitlement" && input.Correction != "restore_allowance" {
		return respond(e, http.StatusBadRequest, "correction_reason_required", "A supported correction and non-empty reason are required.")
	}
	now := clock().UTC()
	var result *core.Record
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		row, findErr := findAuthorizedLesson(tx, e.Request.PathValue("id"), "teacher", teacher.Id)
		if findErr != nil {
			return findErr
		}
		eventID, eventErr := latestCorrectableLessonEvent(tx, row)
		if eventErr != nil {
			return eventErr
		}
		command, commandErr := lifecycleCommand(tx, row, history.Actor{Role: history.TeacherActor, ID: teacher.Id}, now, time.Time{})
		if commandErr != nil {
			return commandErr
		}
		if row.GetString(schedulingstore.ContractField) != "" {
			command.EntitlementEventID, commandErr = latestContractEntitlementEvent(tx, row)
			if commandErr != nil {
				return commandErr
			}
		}
		decision, decisionErr := lessondomain.Correct(command, eventID, input.Reason)
		if decisionErr != nil {
			return decisionErr
		}
		if err := persistLifecycleDecision(tx, row, decision, now); err != nil {
			return err
		}
		result, commandErr = tx.FindRecordById(schedulingstore.LessonsCollectionName, row.Id)
		return commandErr
	})
	if err != nil {
		return lifecycleError(e, err)
	}
	return e.JSON(http.StatusOK, lessonValueAt(result, now))
}

func latestCorrectableLessonEvent(app core.App, lesson *core.Record) (string, error) {
	rows, err := app.FindAllRecords(schedulingstore.BusinessEventsCollectionName)
	if err != nil {
		return "", err
	}
	eligible := make([]*core.Record, 0)
	for _, row := range rows {
		if row.GetString(schedulingstore.AssignmentField) == lesson.GetString(schedulingstore.AssignmentField) && row.GetString(schedulingstore.AggregateTypeField) == "lesson" && row.GetString(schedulingstore.AggregateIDField) == lesson.Id && correctableLessonEvent(row.GetString(schedulingstore.EventTypeField), lesson.GetString(schedulingstore.PlanTypeField)) {
			eligible = append(eligible, row)
		}
	}
	if len(eligible) == 0 {
		return "", lessondomain.ErrCorrectionTarget
	}
	sort.Slice(eligible, func(i, j int) bool {
		return recordInstant(eligible[i], schedulingstore.EventAtField).After(recordInstant(eligible[j], schedulingstore.EventAtField))
	})
	return eligible[0].Id, nil
}

func correctableLessonEvent(eventType, plan string) bool {
	if eventType == string(history.LessonCancelled) || eventType == string(history.LessonOutcomeRecorded) {
		return true
	}
	return plan == "regular_contract" && eventType == string(history.LessonRescheduled)
}
