// This file lists practice tasks, applies the teacher practice plan in one transaction, and updates one task.
package practiceapi

import (
	"net/http"
	"slices"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/practice"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type newTask struct {
	Title            string `json:"title"`
	Details          string `json:"details"`
	SuggestedMinutes *int   `json:"suggested_minutes"`
	Piece            string `json:"piece"`
	Material         string `json:"material"`
}

type planInput struct {
	Lesson  string    `json:"lesson"`
	Keep    []string  `json:"keep"`
	Done    []string  `json:"done"`
	Archive []string  `json:"archive"`
	Create  []newTask `json:"create"`
}

type taskPatch struct {
	Title            *string `json:"title"`
	Details          *string `json:"details"`
	SuggestedMinutes *int    `json:"suggested_minutes"`
	Status           *string `json:"status"`
}

func listTasks(e *core.RequestEvent, who personaroute.Role) error {
	assignment, err := ownedAssignment(e, who)
	if err != nil {
		return respondError(e, err)
	}
	status := e.Request.URL.Query().Get("status")
	if status == "" {
		status = practice.StatusActive
	}
	filter, sort := "assignment = {:assignment} && status = {:status}", "position,created"
	if status == "all" {
		filter, sort = "assignment = {:assignment}", "status,position,-updated"
	} else if !practice.ValidStatus(status) {
		return respondError(e, errNotFound)
	} else if status != practice.StatusActive {
		sort = "-updated"
	}
	rows, err := e.App.FindRecordsByFilter(practice.TasksCollectionName, filter, sort, 0, 0, dbx.Params{"assignment": assignment.Id, "status": status})
	if err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, map[string]any{"items": taskDTOs(e.App, rows)})
}

func savePlan(e *core.RequestEvent) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	assignment, err := ownedAssignment(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	var input planInput
	if err := decode(e, &input, errInvalidPlan); err != nil {
		return respondError(e, err)
	}
	if err := checkPlan(e.App, assignment.Id, input); err != nil {
		return respondError(e, err)
	}
	if err := e.App.RunInTransaction(func(tx core.App) error { return applyPlan(tx, assignment.Id, input) }); err != nil {
		return respondError(e, err)
	}
	rows, err := e.App.FindRecordsByFilter(practice.TasksCollectionName, "assignment = {:assignment} && status = 'active'", "position,created", 0, 0, dbx.Params{"assignment": assignment.Id})
	if err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, map[string]any{"items": taskDTOs(e.App, rows)})
}

// checkPlan rejects a foreign or repeated task, a foreign lesson, piece, or material, and an invalid new task before any write.
func checkPlan(app core.App, assignmentID string, input planInput) error {
	named := slices.Concat(input.Keep, input.Done, input.Archive)
	if len(named)+len(input.Create) > practice.MaxPlanTasks {
		return errInvalidPlan
	}
	seen := map[string]bool{}
	for _, id := range named {
		if seen[id] || !belongs(app, practice.TasksCollectionName, practice.AssignmentField, id, assignmentID) {
			return errInvalidPlan
		}
		seen[id] = true
	}
	if input.Lesson != "" && !belongs(app, schedulingstore.LessonsCollectionName, schedulingstore.AssignmentField, input.Lesson, assignmentID) {
		return errInvalidPlan
	}
	for _, task := range input.Create {
		if practice.ValidateTask(practice.NormalizeTask(textOf(task))) != nil || !optionalLink(app, task, assignmentID) {
			return errInvalidPlan
		}
	}
	return nil
}

func optionalLink(app core.App, task newTask, assignmentID string) bool {
	pieceOK := task.Piece == "" || belongs(app, repertoire.CollectionName, repertoire.AssignmentField, task.Piece, assignmentID)
	materialOK := task.Material == "" || belongs(app, materials.CollectionName, materials.AssignmentField, task.Material, assignmentID)
	return pieceOK && materialOK
}

func belongs(app core.App, collection, field, id, assignmentID string) bool {
	record, err := app.FindRecordById(collection, id)
	return err == nil && record.GetString(field) == assignmentID
}

func textOf(task newTask) practice.Task {
	minutes := 0
	if task.SuggestedMinutes != nil {
		minutes = *task.SuggestedMinutes
	}
	return practice.Task{Title: task.Title, Details: task.Details, SuggestedMinutes: minutes}
}

// applyPlan sets the statuses, then numbers kept tasks, other active tasks, and new tasks in that order.
func applyPlan(tx core.App, assignmentID string, input planInput) error {
	for status, ids := range map[string][]string{practice.StatusDone: input.Done, practice.StatusArchived: input.Archive} {
		if err := setStatus(tx, ids, status); err != nil {
			return err
		}
	}
	collection, err := tx.FindCachedCollectionByNameOrId(practice.TasksCollectionName)
	if err != nil {
		return err
	}
	created := []string{}
	for _, task := range input.Create {
		record := core.NewRecord(collection)
		fields := practice.NormalizeTask(textOf(task))
		record.Load(map[string]any{practice.AssignmentField: assignmentID, practice.LessonField: input.Lesson, practice.TitleField: fields.Title, practice.DetailsField: fields.Details, practice.SuggestedMinutesField: fields.SuggestedMinutes, practice.PieceField: task.Piece, practice.MaterialField: task.Material, practice.StatusField: practice.StatusActive})
		if err := tx.Save(record); err != nil {
			return err
		}
		created = append(created, record.Id)
	}
	return renumber(tx, assignmentID, input.Keep, created)
}

func setStatus(tx core.App, ids []string, status string) error {
	for _, id := range ids {
		record, err := tx.FindRecordById(practice.TasksCollectionName, id)
		if err != nil {
			return err
		}
		record.Set(practice.StatusField, status)
		if err := tx.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func renumber(tx core.App, assignmentID string, kept, created []string) error {
	rows, err := tx.FindRecordsByFilter(practice.TasksCollectionName, "assignment = {:assignment} && status = 'active'", "position,created", 0, 0, dbx.Params{"assignment": assignmentID})
	if err != nil {
		return err
	}
	slices.SortStableFunc(rows, func(left, right *core.Record) int {
		return rank(kept, created, left.Id) - rank(kept, created, right.Id)
	})
	for index, row := range rows {
		row.Set(practice.PositionField, index+1)
		if err := tx.Save(row); err != nil {
			return err
		}
	}
	return nil
}

// rank puts kept tasks first in request order, other active tasks next in their stored order, and new tasks last.
func rank(kept, created []string, id string) int {
	if index := slices.Index(kept, id); index >= 0 {
		return index
	}
	if index := slices.Index(created, id); index >= 0 {
		return len(kept) + 1 + index
	}
	return len(kept)
}

func updateTask(e *core.RequestEvent) error {
	record, _, err := ownedRecord(e, practice.TasksCollectionName, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	var input taskPatch
	if err := decode(e, &input, practice.ErrInvalidTask); err != nil {
		return respondError(e, err)
	}
	fields := practice.NormalizeTask(practice.Task{Title: or(input.Title, record.GetString(practice.TitleField)), Details: or(input.Details, record.GetString(practice.DetailsField)), SuggestedMinutes: record.GetInt(practice.SuggestedMinutesField)})
	if input.SuggestedMinutes != nil {
		fields.SuggestedMinutes = *input.SuggestedMinutes
	}
	status := or(input.Status, record.GetString(practice.StatusField))
	if err := practice.ValidateTask(fields); err != nil || !practice.ValidStatus(status) {
		return respondError(e, practice.ErrInvalidTask)
	}
	record.Load(map[string]any{practice.TitleField: fields.Title, practice.DetailsField: fields.Details, practice.SuggestedMinutesField: fields.SuggestedMinutes, practice.StatusField: status})
	if err := e.App.Save(record); err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, taskDTOs(e.App, []*core.Record{record})[0])
}

func or(field *string, fallback string) string {
	if field == nil {
		return fallback
	}
	return *field
}
