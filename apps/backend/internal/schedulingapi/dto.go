// This file defines scheduling DTOs and converts PocketBase records to safe API values.
package schedulingapi

import (
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type assignmentDTO struct {
	ID              string `json:"id"`
	Teacher         string `json:"teacher"`
	TeacherName     string `json:"teacher_name"`
	Learner         string `json:"learner"`
	LearnerName     string `json:"learner_name"`
	Active          bool   `json:"active"`
	DefaultDuration int    `json:"default_duration_minutes"`
}

type ruleDTO struct {
	ID        string `json:"id"`
	Teacher   string `json:"teacher"`
	Weekday   int    `json:"weekday"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Enabled   bool   `json:"enabled"`
}

type exceptionDTO struct {
	ID      string `json:"id"`
	Teacher string `json:"teacher"`
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
	Kind    string `json:"kind"`
	Note    string `json:"note,omitempty"`
}

type lessonDTO struct {
	ID               string        `json:"id"`
	Teacher          string        `json:"teacher"`
	Learner          string        `json:"learner"`
	Assignment       string        `json:"assignment"`
	StartAt          string        `json:"start_at"`
	EndAt            string        `json:"end_at"`
	Duration         int           `json:"duration_minutes"`
	Status           string        `json:"status"`
	Protected        *protectedDTO `json:"protected_interval,omitempty"`
	CancellationRole string        `json:"cancellation_initiator_role,omitempty"`
	CancellationID   string        `json:"cancellation_initiator_id,omitempty"`
	CancelledAt      string        `json:"cancelled_at,omitempty"`
}

type protectedDTO struct {
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
}
type slotDTO struct {
	StartAt   string        `json:"start_at"`
	EndAt     string        `json:"end_at"`
	Duration  int           `json:"duration_minutes"`
	Protected *protectedDTO `json:"protected_interval,omitempty"`
}
type calendarDTO struct {
	Assignments []assignmentDTO `json:"assignments"`
	Rules       []ruleDTO       `json:"availability_rules"`
	Exceptions  []exceptionDTO  `json:"availability_exceptions"`
	Lessons     []lessonDTO     `json:"lessons"`
	Counters    countersDTO     `json:"cancellation_counters"`
}
type countersDTO struct {
	Teacher int `json:"teacher"`
	Learner int `json:"learner"`
}

func assignmentValue(app core.App, r *core.Record) (assignmentDTO, error) {
	teacherID, learnerID := r.GetString("teacher"), r.GetString("learner")
	teacherName, err := verifiedName(app, authconfig.TeachersCollectionName, teacherID)
	if err != nil {
		return assignmentDTO{}, err
	}
	learnerName, err := verifiedName(app, authconfig.LearnersCollectionName, learnerID)
	if err != nil {
		return assignmentDTO{}, err
	}
	return assignmentDTO{ID: r.Id, Teacher: teacherID, TeacherName: teacherName, Learner: learnerID, LearnerName: learnerName, Active: r.GetBool(schedulingstore.ActiveField), DefaultDuration: r.GetInt(schedulingstore.DefaultDurationMinutesField)}, nil
}

func verifiedName(app core.App, collection, id string) (string, error) {
	if id == "" {
		return "", errRelatedIdentity
	}
	record, err := app.FindRecordById(collection, id)
	if err != nil || record.Collection() == nil || record.Collection().Name != collection || !record.Verified() {
		return "", errRelatedIdentity
	}
	name := record.GetString(authconfig.TeacherNameField)
	if name == "" {
		return "", errRelatedIdentity
	}
	return name, nil
}
func ruleValue(r *core.Record) ruleDTO {
	return ruleDTO{ID: r.Id, Teacher: r.GetString("teacher"), Weekday: r.GetInt(schedulingstore.WeekdayField), StartTime: r.GetString(schedulingstore.StartTimeField), EndTime: r.GetString(schedulingstore.EndTimeField), Enabled: r.GetBool(schedulingstore.EnabledField)}
}
func exceptionValue(r *core.Record) exceptionDTO {
	return exceptionDTO{ID: r.Id, Teacher: r.GetString("teacher"), StartAt: utcString(r.GetDateTime(schedulingstore.StartAtField).Time()), EndAt: utcString(r.GetDateTime(schedulingstore.EndAtField).Time()), Kind: r.GetString(schedulingstore.KindField), Note: r.GetString(schedulingstore.NoteField)}
}
func lessonValue(r *core.Record) lessonDTO {
	start, end := r.GetDateTime(schedulingstore.StartAtField).Time().UTC(), r.GetDateTime(schedulingstore.EndAtField).Time().UTC()
	protected := &protectedDTO{StartAt: utcString(start.Add(-scheduling.ProtectedBuffer)), EndAt: utcString(end.Add(scheduling.ProtectedBuffer))}
	return lessonDTO{ID: r.Id, Teacher: r.GetString("teacher"), Learner: r.GetString("learner"), Assignment: r.GetString(schedulingstore.AssignmentField), StartAt: utcString(start), EndAt: utcString(end), Duration: r.GetInt(schedulingstore.DurationMinutesField), Status: r.GetString(schedulingstore.StatusField), Protected: protected, CancellationRole: r.GetString(schedulingstore.CancellationInitiatorRoleField), CancellationID: r.GetString(schedulingstore.CancellationInitiatorIDField), CancelledAt: utcString(r.GetDateTime(schedulingstore.CancelledAtField).Time())}
}
func utcString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
func parseInstant(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil || t.IsZero() {
		return time.Time{}, errInvalid
	}
	return t.UTC(), nil
}
