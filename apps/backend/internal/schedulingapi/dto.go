// This file defines scheduling DTOs and converts PocketBase records to safe API values.
package schedulingapi

import (
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type assignmentDTO struct {
	ID          string `json:"id"`
	Teacher     string `json:"teacher"`
	TeacherName string `json:"teacher_name"`
	Learner     string `json:"learner"`
	LearnerName string `json:"learner_name"`
	Active      bool   `json:"active"`
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
	Enabled bool   `json:"enabled"`
}

type lessonDTO struct {
	ID                string        `json:"id"`
	Teacher           string        `json:"teacher"`
	Learner           string        `json:"learner"`
	Assignment        string        `json:"assignment"`
	PlanType          string        `json:"plan_type"`
	PackageToken      *string       `json:"package_token"`
	Contract          *string       `json:"contract"`
	OriginalLocalDate string        `json:"original_local_date,omitempty"`
	OriginalStartAt   string        `json:"original_start_at,omitempty"`
	StartAt           string        `json:"start_at"`
	EndAt             string        `json:"end_at"`
	Duration          int           `json:"duration_minutes"`
	UnitPriceMinor    int64         `json:"unit_price_minor"`
	Currency          string        `json:"currency"`
	PolicyVersion     string        `json:"policy_version"`
	ScheduleState     string        `json:"schedule_state"`
	Outcome           *string       `json:"outcome"`
	Protected         *protectedDTO `json:"protected_interval"`
	CancellationRole  string        `json:"cancellation_initiator_role,omitempty"`
	CancellationID    string        `json:"cancellation_initiator_id,omitempty"`
	CancelledAt       string        `json:"cancelled_at,omitempty"`
	Cutoff            string        `json:"cutoff,omitempty"`
	PlanEffect        string        `json:"plan_effect,omitempty"`
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
	Assignments          []assignmentDTO `json:"assignments"`
	Rules                []ruleDTO       `json:"availability_rules"`
	Exceptions           []exceptionDTO  `json:"availability_exceptions"`
	CommercialSummaries  []any           `json:"commercial_summaries"`
	NearTermLessons      []lessonDTO     `json:"near_term_lessons"`
	LaterContractLessons []lessonDTO     `json:"later_contract_lessons,omitempty"`
	PaymentSummary       []any           `json:"payment_summary,omitempty"`
	HistorySummary       []historyCount  `json:"history_summary,omitempty"`
	UnresolvedWork       *unresolvedDTO  `json:"unresolved_work,omitempty"`
}

type historyCount struct {
	Assignment string `json:"assignment"`
	EventCount int    `json:"event_count"`
}

type unresolvedDTO struct {
	AwaitingOutcome   int `json:"awaiting_outcome"`
	PendingSettlement int `json:"pending_settlement"`
	UnpaidCharges     int `json:"unpaid_charges"`
	OverdueCharges    int `json:"overdue_charges"`
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
	return assignmentDTO{ID: r.Id, Teacher: teacherID, TeacherName: teacherName, Learner: learnerID, LearnerName: learnerName, Active: r.GetBool(schedulingstore.ActiveField)}, nil
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
	return exceptionDTO{ID: r.Id, Teacher: r.GetString("teacher"), StartAt: utcString(r.GetDateTime(schedulingstore.StartAtField).Time()), EndAt: utcString(r.GetDateTime(schedulingstore.EndAtField).Time()), Kind: r.GetString(schedulingstore.KindField), Note: r.GetString(schedulingstore.NoteField), Enabled: r.GetBool(schedulingstore.EnabledField)}
}
func lessonValueAt(r *core.Record, now time.Time) lessonDTO {
	start, end := r.GetDateTime(schedulingstore.StartAtField).Time().UTC(), r.GetDateTime(schedulingstore.EndAtField).Time().UTC()
	buffer := businesspolicy.Current().ParticipantBuffer
	protected := &protectedDTO{StartAt: utcString(start.Add(-buffer)), EndAt: utcString(end.Add(buffer))}
	scheduleState := r.GetString(schedulingstore.ScheduleStateField)
	if scheduleState == "" {
		scheduleState = r.GetString(schedulingstore.StatusField)
	}
	outcome := r.GetString(schedulingstore.OutcomeField)
	if outcome == "" && scheduleState == "scheduled" && !end.After(now) {
		outcome = "awaiting_outcome"
	}
	return lessonDTO{ID: r.Id, Teacher: r.GetString("teacher"), Learner: r.GetString("learner"), Assignment: r.GetString(schedulingstore.AssignmentField), PlanType: r.GetString(schedulingstore.PlanTypeField), PackageToken: optionalString(r.GetString(schedulingstore.PackageTokenField)), Contract: optionalString(r.GetString(schedulingstore.ContractField)), OriginalLocalDate: r.GetString(schedulingstore.OriginalLocalDateField), OriginalStartAt: utcString(r.GetDateTime(schedulingstore.OriginalStartAtField).Time()), StartAt: utcString(start), EndAt: utcString(end), Duration: r.GetInt(schedulingstore.DurationMinutesField), UnitPriceMinor: int64(r.GetInt(schedulingstore.UnitPriceMinorField)), Currency: r.GetString(schedulingstore.CurrencyField), PolicyVersion: r.GetString(schedulingstore.PolicyVersionField), ScheduleState: scheduleState, Outcome: optionalString(outcome), Protected: protected, CancellationRole: r.GetString(schedulingstore.CancellationInitiatorRoleField), CancellationID: r.GetString(schedulingstore.CancellationInitiatorIDField), CancelledAt: utcString(r.GetDateTime(schedulingstore.CancelledAtField).Time())}
}

func lessonValue(r *core.Record) lessonDTO { return lessonValueAt(r, time.Now().UTC()) }

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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
