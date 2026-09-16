// This file atomically reschedules and cancels authorized future lessons while retaining immutable event history.
package schedulingapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	lessondomain "github.com/balickim/nutka/apps/backend/internal/lesson"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type rescheduleInput struct {
	StartAt         *string         `json:"start_at"`
	DurationMinutes json.RawMessage `json:"duration_minutes"`
}

func rescheduleLesson(e *core.RequestEvent, role string, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	account, err := caller(e, role)
	if err != nil {
		return handleError(e, err)
	}
	var input rescheduleInput
	if err := bindBody(e, &input); err != nil {
		return handleError(e, err)
	}
	if input.DurationMinutes != nil {
		return handleError(e, errDurationOverride)
	}
	if input.StartAt == nil {
		return handleError(e, errInvalid)
	}
	replacement, err := parseInstant(*input.StartAt)
	if err != nil {
		return handleError(e, err)
	}
	now := clock().UTC()
	var result *core.Record
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		lesson, lookupErr := findAuthorizedLesson(tx, e.Request.PathValue("id"), role, account.Id)
		if lookupErr != nil {
			return lookupErr
		}
		assignment, assignmentErr := tx.FindRecordById(schedulingstore.TeacherLearnersCollectionName, lesson.GetString(schedulingstore.AssignmentField))
		if assignmentErr != nil || !assignment.GetBool(schedulingstore.ActiveField) {
			return errForbidden
		}
		command, commandErr := lifecycleCommand(tx, lesson, history.Actor{Role: history.ActorRole(role), ID: account.Id}, now, replacement)
		if commandErr != nil {
			return commandErr
		}
		decision, decisionErr := lessondomain.RescheduleLesson(command)
		if decisionErr != nil {
			return decisionErr
		}
		if err := persistLifecycleDecision(tx, lesson, decision, now); err != nil {
			return err
		}
		result = lesson
		return nil
	})
	if err != nil {
		return lifecycleError(e, err)
	}
	return e.JSON(http.StatusOK, lessonValueAt(result, now))
}

func findAuthorizedLesson(app core.App, id, role, account string) (*core.Record, error) {
	if id == "" {
		return nil, errInvalid
	}
	row, err := app.FindRecordById(schedulingstore.LessonsCollectionName, id)
	if err != nil {
		return nil, errForbidden
	}
	field := "learner"
	if role == "teacher" {
		field = "teacher"
	}
	if row.GetString(field) != account {
		return nil, errForbidden
	}
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, row.GetString(schedulingstore.AssignmentField))
	if err != nil || assignment.GetString("teacher") != row.GetString("teacher") || assignment.GetString("learner") != row.GetString("learner") {
		return nil, errForbidden
	}
	return row, nil
}

func cancelLesson(e *core.RequestEvent, role string, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	account, err := caller(e, role)
	if err != nil {
		return handleError(e, err)
	}
	if e.Request.Body != nil && e.Request.ContentLength != 0 {
		var input struct{}
		if err := bindBody(e, &input); err != nil {
			return handleError(e, err)
		}
	}
	now := clock().UTC()
	var result *core.Record
	var planEffect, cutoff string
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		lesson, lookupErr := findAuthorizedLesson(tx, e.Request.PathValue("id"), role, account.Id)
		if lookupErr != nil {
			return lookupErr
		}
		command, commandErr := lifecycleCommand(tx, lesson, history.Actor{Role: history.ActorRole(role), ID: account.Id}, now, time.Time{})
		if commandErr != nil {
			return commandErr
		}
		decision, decisionErr := lessondomain.CancelLesson(command)
		if decisionErr != nil {
			return decisionErr
		}
		if err := persistLifecycleDecision(tx, lesson, decision, now); err != nil {
			return err
		}
		result = lesson
		planEffect = decision.PlanEffect
		if role == "learner" && command.Lesson.Interval.Start.Sub(now) < commandCutoff(command) {
			cutoff = "late"
		} else {
			cutoff = "timely"
		}
		return nil
	})
	if err != nil {
		return lifecycleError(e, err)
	}
	value := lessonValueAt(result, now)
	value.PlanEffect, value.Cutoff = planEffect, cutoff
	return e.JSON(http.StatusOK, value)
}

func commandCutoff(command lessondomain.Command) time.Duration {
	if command.Contract != nil {
		return command.Contract.Policy.LearnerChangeCutoff
	}
	if command.Package != nil {
		return time.Duration(command.Package.Policy.LearnerChangeCutoffHours) * time.Hour
	}
	return 24 * time.Hour
}

func lifecycleError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, lessondomain.ErrUnauthorized):
		return handleError(e, errForbidden)
	case errors.Is(err, lessondomain.ErrStarted), errors.Is(err, lessondomain.ErrCancelled):
		return respond(e, http.StatusConflict, "started_lesson", "A started or closed lesson cannot change.")
	case errors.Is(err, commercial.ErrLearnerChangeTooLate), errors.Is(err, regularcontract.ErrCutoff):
		return respond(e, http.StatusBadRequest, "learner_change_cutoff", "Learner rescheduling is unavailable inside the change cutoff.")
	case errors.Is(err, regularcontract.ErrAllowanceExhausted), errors.Is(err, regularcontract.ErrAlreadyRescheduled):
		return respond(e, http.StatusConflict, "contract_allowance_exhausted", "The contract allowance is exhausted.")
	case errors.Is(err, regularcontract.ErrReplacementDeadline):
		return respond(e, http.StatusBadRequest, "contract_replacement_deadline", "The replacement exceeds the contract deadline.")
	case errors.Is(err, regularcontract.ErrUnavailable), errors.Is(err, scheduling.ErrConflict), errors.Is(err, errConflict):
		return respond(e, http.StatusConflict, "lesson_conflict", "The replacement conflicts with availability or a participant lesson.")
	case errors.Is(err, lessondomain.ErrOutcomeUnavailable), errors.Is(err, lessondomain.ErrOutcomeAlreadyStored):
		return respond(e, http.StatusConflict, "invalid_outcome", "The lesson outcome cannot change in its current state.")
	case errors.Is(err, lessondomain.ErrInvalidCommand), errors.Is(err, lessondomain.ErrReplacementRequired):
		return handleError(e, errInvalid)
	case errors.Is(err, lessondomain.ErrCorrectionTarget):
		return respond(e, http.StatusBadRequest, "correction_not_allowed", "The lesson transition cannot be corrected.")
	default:
		return handleError(e, err)
	}
}
