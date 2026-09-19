// This file projects stored tasks and sessions into DTOs with linked titles.
package practiceapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/balickim/nutka/apps/backend/internal/practice"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/pocketbase/pocketbase/core"
)

type link struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type taskDTO struct {
	ID               string  `json:"id"`
	Assignment       string  `json:"assignment"`
	Lesson           *string `json:"lesson"`
	Title            string  `json:"title"`
	Details          string  `json:"details"`
	SuggestedMinutes *int    `json:"suggested_minutes"`
	Piece            *link   `json:"piece"`
	Material         *link   `json:"material"`
	Status           string  `json:"status"`
	Position         int     `json:"position"`
	CreatedAt        string  `json:"created_at"`
}

type sessionDTO struct {
	ID          string `json:"id"`
	PracticedOn string `json:"practiced_on"`
	Minutes     *int   `json:"minutes"`
	Tasks       []link `json:"tasks"`
	Comment     string `json:"comment"`
	CreatedAt   string `json:"created_at"`
	Deletable   bool   `json:"deletable"`
}

func taskDTOs(app core.App, rows []*core.Record) []taskDTO {
	items := make([]taskDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, taskDTO{
			ID:               row.Id,
			Assignment:       row.GetString(practice.AssignmentField),
			Lesson:           optional(row.GetString(practice.LessonField)),
			Title:            row.GetString(practice.TitleField),
			Details:          row.GetString(practice.DetailsField),
			SuggestedMinutes: positive(row.GetInt(practice.SuggestedMinutesField)),
			Piece:            linked(app, repertoire.CollectionName, row.GetString(practice.PieceField), repertoire.TitleField),
			Material:         linked(app, materials.CollectionName, row.GetString(practice.MaterialField), materials.TitleField),
			Status:           row.GetString(practice.StatusField),
			Position:         row.GetInt(practice.PositionField),
			CreatedAt:        utc(row.GetDateTime("created").Time()),
		})
	}
	return items
}

// sessionDTOs marks a session deletable only for the learner within the delete window.
func sessionDTOs(app core.App, rows []*core.Record, who personaroute.Role, now time.Time) []sessionDTO {
	items := make([]sessionDTO, 0, len(rows))
	for _, row := range rows {
		tasks := []link{}
		for _, id := range row.GetStringSlice(practice.TasksField) {
			if task := linked(app, practice.TasksCollectionName, id, practice.TitleField); task != nil {
				tasks = append(tasks, *task)
			}
		}
		created := row.GetDateTime("created").Time()
		items = append(items, sessionDTO{
			ID:          row.Id,
			PracticedOn: row.GetString(practice.PracticedOnField),
			Minutes:     positive(row.GetInt(practice.MinutesField)),
			Tasks:       tasks,
			Comment:     row.GetString(practice.CommentField),
			CreatedAt:   utc(created),
			Deletable:   who == personaroute.Learner && practice.LearnerMayDelete(created, now) == nil,
		})
	}
	return items
}

func linked(app core.App, collection, id, titleField string) *link {
	if id == "" {
		return nil
	}
	record, err := app.FindRecordById(collection, id)
	if err != nil {
		return nil
	}
	return &link{ID: record.Id, Title: record.GetString(titleField)}
}

func optional(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}

func positive(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func utc(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
