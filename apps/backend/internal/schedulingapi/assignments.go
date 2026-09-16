// This file serves private assignment lists and teacher-owned assignment configuration.
package schedulingapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type assignmentUpdate struct {
	Active    *bool  `json:"active"`
	Teacher   string `json:"teacher"`
	Learner   string `json:"learner"`
	ActorID   string `json:"actor_id"`
	ActorRole string `json:"actor_role"`
}

func assignments(e *core.RequestEvent) error {
	role := "learner"
	if e.Request.URL.Path == "/api/teachers/assignments" {
		role = "teacher"
	}
	account, err := caller(e, role)
	if err != nil {
		return handleError(e, err)
	}
	rows, err := ownedAssignments(e.App, role, account.Id)
	if err != nil {
		return handleError(e, err)
	}
	items := make([]assignmentDTO, 0, len(rows))
	for _, row := range rows {
		item, valueErr := assignmentValue(e.App, row)
		if valueErr != nil {
			return handleError(e, valueErr)
		}
		items = append(items, item)
	}
	return e.JSON(http.StatusOK, map[string]any{"assignments": items})
}

func teacherAssignmentUpdate(e *core.RequestEvent) error {
	return teacherAssignmentUpdateAt(e, time.Now)
}

func teacherAssignmentUpdateAt(e *core.RequestEvent, clock Clock) error {
	if err := requireMutation(e); err != nil {
		return handleError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	var input assignmentUpdate
	if bindErr := bindBody(e, &input); bindErr != nil {
		return handleError(e, bindErr)
	}
	if input.Active == nil {
		return handleError(e, errInvalid)
	}
	id := e.Request.PathValue("id")
	var updated *core.Record
	if clock == nil {
		clock = time.Now
	}
	now := clock().UTC()
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		row, lookupErr := ownedAssignment(tx, id, "teacher", teacher.Id)
		if lookupErr != nil {
			return lookupErr
		}
		if !*input.Active && row.GetBool(schedulingstore.ActiveField) {
			obligations, obligationErr := assignmentObligations(tx, row.Id, now)
			if obligationErr != nil {
				return obligationErr
			}
			if err := regularcontract.CanDeactivateAssignment(obligations); err != nil {
				return err
			}
		}
		row.Set(schedulingstore.ActiveField, *input.Active)
		if saveErr := tx.Save(row); saveErr != nil {
			return errInvalid
		}
		updated = row
		return nil
	})
	if err != nil {
		if errors.Is(err, regularcontract.ErrAssignmentBlocked) {
			return respond(e, http.StatusConflict, "unresolved_obligation", "A contract, token, package-backed lesson, or future lesson blocks deactivation.")
		}
		return handleError(e, err)
	}
	item, valueErr := assignmentValue(e.App, updated)
	if valueErr != nil {
		return handleError(e, valueErr)
	}
	return e.JSON(http.StatusOK, item)
}
