// Package lessonnotes defines teacher notes on one lesson: storage names, limits, and the rules for eligible lessons and valid content.
// It has no HTTP or persistence side effects.
package lessonnotes

import (
	"errors"
	"time"
)

const (
	CollectionName  = "lesson_notes"
	LessonField     = "lesson"
	AssignmentField = "assignment"
	BodyField       = "body"
	MaterialsField  = "materials"

	BodyMaxBytes = 20 << 10
	MaxMaterials = 10
)

var (
	ErrNotStarted = errors.New("lesson has not started")
	ErrCancelled  = errors.New("lesson is cancelled")
	ErrInvalid    = errors.New("lesson note is invalid")
)

// CheckLesson allows a note only for a scheduled lesson whose start is not in the future.
func CheckLesson(startAt time.Time, scheduleState string, now time.Time) error {
	if scheduleState != "scheduled" {
		return ErrCancelled
	}
	if startAt.After(now) {
		return ErrNotStarted
	}
	return nil
}

// Validate checks a sanitized body and the material count.
func Validate(sanitizedBody string, materialCount int) error {
	if sanitizedBody == "" || len(sanitizedBody) > BodyMaxBytes || materialCount > MaxMaterials {
		return ErrInvalid
	}
	return nil
}
