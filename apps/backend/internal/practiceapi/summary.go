// This file serves the practice summary of one assignment and the teacher summaries for the lessons of one local day.
package practiceapi

import (
	"net/http"
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/practice"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type taskCount struct {
	Task  string `json:"task"`
	Title string `json:"title"`
	Count int    `json:"count"`
}

type summaryDTO struct {
	practice.Summary
	Tasks []taskCount `json:"tasks"`
}

type assignmentSummaryDTO struct {
	summaryDTO
	Today      string   `json:"today"`
	RecentDays []string `json:"recent_days"`
}

type lessonSummaryDTO struct {
	Lesson     string     `json:"lesson"`
	Assignment string     `json:"assignment"`
	Summary    summaryDTO `json:"summary"`
}

func assignmentSummary(e *core.RequestEvent, who personaroute.Role, now time.Time) error {
	assignment, err := ownedAssignment(e, who)
	if err != nil {
		return respondError(e, err)
	}
	location := teacherLocation(e.App, assignment.GetString("teacher"))
	facts, starts, err := assignmentFacts(e.App, assignment.Id)
	if err != nil {
		return respondError(e, err)
	}
	summary := withTitles(e.App, practice.Summarize(facts, practice.WindowStart(starts, now, now, location)))
	return e.JSON(http.StatusOK, assignmentSummaryDTO{summaryDTO: summary, Today: practice.Today(now, location), RecentDays: practice.RecentDays(facts, now, location)})
}

// daySummaries returns one summary for each scheduled lesson of the teacher on the local date. Each window starts at the previous lesson of that assignment.
func daySummaries(e *core.RequestEvent, now time.Time) error {
	teacher, err := personaroute.Caller(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	location := teacherLocation(e.App, teacher.Id)
	date := e.Request.URL.Query().Get("date")
	if _, err := time.Parse(practice.DateLayout, date); err != nil {
		return respondError(e, errNotFound)
	}
	lessons, err := e.App.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{"teacher": teacher.Id, schedulingstore.ScheduleStateField: "scheduled"})
	if err != nil {
		return respondError(e, err)
	}
	items := []lessonSummaryDTO{}
	for _, lesson := range lessons {
		start := lesson.GetDateTime(schedulingstore.StartAtField).Time()
		if start.In(location).Format(practice.DateLayout) != date {
			continue
		}
		item, err := lessonSummary(e.App, lesson, start, now, location)
		if err != nil {
			return respondError(e, err)
		}
		items = append(items, item)
	}
	sort.Slice(items, func(left, right int) bool { return items[left].Lesson < items[right].Lesson })
	return e.JSON(http.StatusOK, map[string]any{"items": items})
}

func lessonSummary(app core.App, lesson *core.Record, start, now time.Time, location *time.Location) (lessonSummaryDTO, error) {
	assignmentID := lesson.GetString(schedulingstore.AssignmentField)
	facts, starts, err := assignmentFacts(app, assignmentID)
	if err != nil {
		return lessonSummaryDTO{}, err
	}
	summary := withTitles(app, practice.Summarize(facts, practice.WindowStart(starts, start, now, location)))
	return lessonSummaryDTO{Lesson: lesson.Id, Assignment: assignmentID, Summary: summary}, nil
}

// assignmentFacts reads the sessions and the scheduled lesson starts of one assignment.
func assignmentFacts(app core.App, assignmentID string) ([]practice.SessionFact, []time.Time, error) {
	rows, err := app.FindAllRecords(practice.SessionsCollectionName, dbx.HashExp{practice.AssignmentField: assignmentID})
	if err != nil {
		return nil, nil, err
	}
	facts := make([]practice.SessionFact, 0, len(rows))
	for _, row := range rows {
		facts = append(facts, practice.SessionFact{PracticedOn: row.GetString(practice.PracticedOnField), Minutes: row.GetInt(practice.MinutesField), Tasks: row.GetStringSlice(practice.TasksField), Comment: row.GetString(practice.CommentField)})
	}
	lessons, err := app.FindAllRecords(schedulingstore.LessonsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignmentID, schedulingstore.ScheduleStateField: "scheduled"})
	if err != nil {
		return nil, nil, err
	}
	starts := make([]time.Time, 0, len(lessons))
	for _, lesson := range lessons {
		starts = append(starts, lesson.GetDateTime(schedulingstore.StartAtField).Time())
	}
	return facts, starts, nil
}

// withTitles resolves task titles and skips deleted tasks. The most practiced task comes first.
func withTitles(app core.App, summary practice.Summary) summaryDTO {
	tasks := []taskCount{}
	for id, count := range summary.TaskCounts {
		if task, err := app.FindRecordById(practice.TasksCollectionName, id); err == nil {
			tasks = append(tasks, taskCount{Task: id, Title: task.GetString(practice.TitleField), Count: count})
		}
	}
	sort.Slice(tasks, func(left, right int) bool {
		if tasks[left].Count != tasks[right].Count {
			return tasks[left].Count > tasks[right].Count
		}
		return tasks[left].Title < tasks[right].Title
	})
	return summaryDTO{Summary: summary, Tasks: tasks}
}
