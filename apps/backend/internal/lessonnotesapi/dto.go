// This file checks material links and projects stored notes into DTOs with lesson starts and material titles.
package lessonnotesapi

import (
	"sort"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/lessonnotes"
	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type materialLink struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type noteDTO struct {
	ID            string         `json:"id"`
	Lesson        string         `json:"lesson"`
	Assignment    string         `json:"assignment"`
	LessonStartAt string         `json:"lesson_start_at"`
	Body          string         `json:"body"`
	Materials     []materialLink `json:"materials"`
	UpdatedAt     string         `json:"updated_at"`
}

// ownedMaterials returns unique material identifiers and rejects any material of another assignment.
func ownedMaterials(app core.App, assignmentID string, ids []string) ([]string, error) {
	result := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		material, err := app.FindRecordById(materials.CollectionName, id)
		if err != nil || material.GetString(materials.AssignmentField) != assignmentID {
			return nil, lessonnotes.ErrInvalid
		}
		seen[id] = true
		result = append(result, id)
	}
	return result, nil
}

func sortedDTOs(app core.App, rows []*core.Record) []noteDTO {
	items := make([]noteDTO, 0, len(rows))
	for _, row := range rows {
		lesson, err := app.FindRecordById(schedulingstore.LessonsCollectionName, row.GetString(lessonnotes.LessonField))
		if err != nil {
			continue
		}
		items = append(items, toDTO(app, row, lesson))
	}
	sort.SliceStable(items, func(left, right int) bool { return items[left].LessonStartAt > items[right].LessonStartAt })
	return items
}

// toDTO skips links to deleted materials, so a removed material disappears from the note without a write.
func toDTO(app core.App, row, lesson *core.Record) noteDTO {
	links := make([]materialLink, 0)
	for _, id := range row.GetStringSlice(lessonnotes.MaterialsField) {
		if material, err := app.FindRecordById(materials.CollectionName, id); err == nil {
			links = append(links, materialLink{ID: material.Id, Title: material.GetString(materials.TitleField)})
		}
	}
	return noteDTO{
		ID:            row.Id,
		Lesson:        lesson.Id,
		Assignment:    row.GetString(lessonnotes.AssignmentField),
		LessonStartAt: utc(lesson.GetDateTime(schedulingstore.StartAtField).Time()),
		Body:          row.GetString(lessonnotes.BodyField),
		Materials:     links,
		UpdatedAt:     utc(row.GetDateTime("updated").Time()),
	}
}

func utc(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
