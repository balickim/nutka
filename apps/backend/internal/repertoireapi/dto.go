// This file projects stored pieces into DTOs with the material count and the latest material instant of each piece.
package repertoireapi

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/materials"
	"github.com/balickim/nutka/apps/backend/internal/repertoire"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type pieceDTO struct {
	ID               string  `json:"id"`
	Assignment       string  `json:"assignment"`
	Title            string  `json:"title"`
	Artist           string  `json:"artist"`
	Status           string  `json:"status"`
	StatusChangedAt  string  `json:"status_changed_at"`
	ProposedBy       string  `json:"proposed_by"`
	MaterialCount    int     `json:"material_count"`
	LatestMaterialAt *string `json:"latest_material_at"`
	CreatedAt        string  `json:"created_at"`
}

type counters struct {
	count  int
	latest time.Time
}

func listDTOs(app core.App, assignmentID string, rows []*core.Record) ([]pieceDTO, error) {
	stats, err := materialCounters(app, assignmentID)
	if err != nil {
		return nil, err
	}
	items := make([]pieceDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDTO(row, stats[row.Id]))
	}
	return items, nil
}

// materialCounters reads the materials of one assignment once and counts them per piece.
func materialCounters(app core.App, assignmentID string) (map[string]counters, error) {
	rows, err := app.FindAllRecords(materials.CollectionName, dbx.HashExp{materials.AssignmentField: assignmentID})
	if err != nil {
		return nil, err
	}
	stats := map[string]counters{}
	for _, row := range rows {
		piece := row.GetString(repertoire.MaterialPieceField)
		if piece == "" {
			continue
		}
		current := stats[piece]
		current.count++
		if created := row.GetDateTime("created").Time(); created.After(current.latest) {
			current.latest = created
		}
		stats[piece] = current
	}
	return stats, nil
}

func toDTO(row *core.Record, stats counters) pieceDTO {
	var latest *string
	if stats.count > 0 {
		value := utc(stats.latest)
		latest = &value
	}
	return pieceDTO{
		ID:               row.Id,
		Assignment:       row.GetString(repertoire.AssignmentField),
		Title:            row.GetString(repertoire.TitleField),
		Artist:           row.GetString(repertoire.ArtistField),
		Status:           row.GetString(repertoire.StatusField),
		StatusChangedAt:  utc(row.GetDateTime(repertoire.StatusChangedAtField).Time()),
		ProposedBy:       row.GetString(repertoire.ProposedByField),
		MaterialCount:    stats.count,
		LatestMaterialAt: latest,
		CreatedAt:        utc(row.GetDateTime("created").Time()),
	}
}

func utc(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
