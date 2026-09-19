// Package repertoire defines the pieces of one assignment: storage names, statuses, limits, and the rules for valid pieces and learner deletes.
// It has no HTTP or persistence side effects.
package repertoire

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	CollectionName       = "pieces"
	AssignmentField      = "assignment"
	TitleField           = "title"
	ArtistField          = "artist"
	StatusField          = "status"
	StatusChangedAtField = "status_changed_at"
	ProposedByField      = "proposed_by"
	MaterialPieceField   = "piece"
	TextMaxLength        = 200
	StatusWish           = "wish"
	StatusLearning       = "learning"
	StatusPlaying        = "playing"
	StatusRepertoire     = "repertoire"
	ProposedByTeacher    = "teacher"
	ProposedByLearner    = "learner"
)

// Statuses lists every piece status. The order is not a required sequence.
var Statuses = []string{StatusWish, StatusLearning, StatusPlaying, StatusRepertoire}

var (
	ErrInvalid = errors.New("piece is invalid")
	ErrLocked  = errors.New("piece cannot be deleted by the learner")
)

// Input holds the editable fields of a piece.
type Input struct {
	Title  string
	Artist string
	Status string
}

// Normalize trims the text fields.
func Normalize(input Input) Input {
	return Input{Title: strings.TrimSpace(input.Title), Artist: strings.TrimSpace(input.Artist), Status: strings.TrimSpace(input.Status)}
}

// Validate checks the title, artist, and status of a normalized piece.
func Validate(input Input) error {
	if input.Title == "" || utf8.RuneCountInString(input.Title) > TextMaxLength || utf8.RuneCountInString(input.Artist) > TextMaxLength || !slices.Contains(Statuses, input.Status) {
		return ErrInvalid
	}
	return nil
}

// LearnerMayDelete allows a learner to delete only their own piece while it is a wish.
func LearnerMayDelete(proposedBy, status string) error {
	if proposedBy != ProposedByLearner || status != StatusWish {
		return ErrLocked
	}
	return nil
}
