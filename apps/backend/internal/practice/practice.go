// Package practice defines practice tasks and learner practice sessions: storage names, limits, and the rules for valid tasks, days, sessions, and deletes.
// It has no HTTP or persistence side effects.
package practice

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	TasksCollectionName    = "practice_tasks"
	SessionsCollectionName = "practice_sessions"

	AssignmentField       = "assignment"
	LessonField           = "lesson"
	TitleField            = "title"
	DetailsField          = "details"
	SuggestedMinutesField = "suggested_minutes"
	PieceField            = "piece"
	MaterialField         = "material"
	StatusField           = "status"
	PositionField         = "position"

	PracticedOnField = "practiced_on"
	MinutesField     = "minutes"
	TasksField       = "tasks"
	CommentField     = "comment"

	StatusActive   = "active"
	StatusDone     = "done"
	StatusArchived = "archived"

	TitleMaxLength      = 200
	DetailsMaxLength    = 1000
	CommentMaxLength    = 500
	MaxSuggestedMinutes = 120
	MaxSessionMinutes   = 240
	MaxSessionTasks     = 20
	MaxPlanTasks        = 50
	BackfillDays        = 14
	DeleteWindow        = 7 * 24 * time.Hour
	DateLayout          = "2006-01-02"
)

// Statuses lists every task status.
var Statuses = []string{StatusActive, StatusDone, StatusArchived}

var (
	ErrInvalidTask    = errors.New("practice task is invalid")
	ErrInvalidSession = errors.New("practice session is invalid")
	ErrLocked         = errors.New("practice session can no longer be deleted")
)

// Task holds the editable text fields of a task. Zero suggested minutes means no suggestion.
type Task struct {
	Title            string
	Details          string
	SuggestedMinutes int
}

// NormalizeTask trims the text fields.
func NormalizeTask(task Task) Task {
	return Task{Title: strings.TrimSpace(task.Title), Details: strings.TrimSpace(task.Details), SuggestedMinutes: task.SuggestedMinutes}
}

// ValidateTask checks a normalized task.
func ValidateTask(task Task) error {
	if task.Title == "" || utf8.RuneCountInString(task.Title) > TitleMaxLength || utf8.RuneCountInString(task.Details) > DetailsMaxLength {
		return ErrInvalidTask
	}
	if task.SuggestedMinutes < 0 || task.SuggestedMinutes > MaxSuggestedMinutes {
		return ErrInvalidTask
	}
	return nil
}

// ValidStatus reports whether status is a task status.
func ValidStatus(status string) bool {
	return slices.Contains(Statuses, status)
}

// Today returns the local date of now in location.
func Today(now time.Time, location *time.Location) string {
	return now.In(location).Format(DateLayout)
}

// AddDays moves a local date by whole days. Calendar arithmetic in UTC avoids daylight saving shifts.
func AddDays(date string, days int) string {
	day, err := time.Parse(DateLayout, date)
	if err != nil {
		return ""
	}
	return day.AddDate(0, 0, days).Format(DateLayout)
}

// CheckDay accepts a local date from today back to BackfillDays before today.
func CheckDay(date string, now time.Time, location *time.Location) error {
	if _, err := time.Parse(DateLayout, date); err != nil {
		return ErrInvalidSession
	}
	today := Today(now, location)
	if date > today || date < AddDays(today, -BackfillDays) {
		return ErrInvalidSession
	}
	return nil
}

// Session holds the learner input of one practice session. Zero minutes means no value.
type Session struct {
	PracticedOn string
	Minutes     int
	Tasks       []string
	Comment     string
}

// ValidateSession checks the day, minutes, task count, and comment of a session with a trimmed comment.
func ValidateSession(session Session, now time.Time, location *time.Location) error {
	if err := CheckDay(session.PracticedOn, now, location); err != nil {
		return err
	}
	if session.Minutes < 0 || session.Minutes > MaxSessionMinutes || len(session.Tasks) > MaxSessionTasks || utf8.RuneCountInString(session.Comment) > CommentMaxLength {
		return ErrInvalidSession
	}
	return nil
}

// LearnerMayDelete allows a delete within DeleteWindow after creation.
func LearnerMayDelete(created, now time.Time) error {
	if now.Sub(created) > DeleteWindow {
		return ErrLocked
	}
	return nil
}
