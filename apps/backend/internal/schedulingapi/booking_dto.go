// This file exposes the fixed-duration commercial fields returned by successful booking commands.
package schedulingapi

import (
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

type bookingLessonDTO struct {
	ID             string              `json:"id"`
	Teacher        string              `json:"teacher"`
	Learner        string              `json:"learner"`
	Assignment     string              `json:"assignment"`
	PlanType       string              `json:"plan_type"`
	PackageToken   *string             `json:"package_token"`
	Contract       *string             `json:"contract"`
	StartAt        string              `json:"start_at"`
	EndAt          string              `json:"end_at"`
	Duration       int                 `json:"duration_minutes"`
	UnitPriceMinor int64               `json:"unit_price_minor"`
	Currency       string              `json:"currency"`
	PolicyVersion  string              `json:"policy_version"`
	ScheduleState  string              `json:"schedule_state"`
	Outcome        *string             `json:"outcome"`
	Protected      bookingProtectedDTO `json:"protected_interval"`
}

type bookingProtectedDTO struct {
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
}

func bookingLessonValue(row *core.Record) bookingLessonDTO {
	start := row.GetDateTime(schedulingstore.StartAtField).Time().UTC()
	end := row.GetDateTime(schedulingstore.EndAtField).Time().UTC()
	return bookingLessonDTO{
		ID: row.Id, Teacher: row.GetString("teacher"), Learner: row.GetString("learner"), Assignment: row.GetString(schedulingstore.AssignmentField),
		PlanType: row.GetString(schedulingstore.PlanTypeField), PackageToken: optionalBookingValue(row.GetString(schedulingstore.PackageTokenField)), Contract: optionalBookingValue(row.GetString(schedulingstore.ContractField)),
		StartAt: utcString(start), EndAt: utcString(end), Duration: businesspolicy.LessonDurationMinutes, UnitPriceMinor: int64(row.GetInt(schedulingstore.UnitPriceMinorField)), Currency: row.GetString(schedulingstore.CurrencyField), PolicyVersion: row.GetString(schedulingstore.PolicyVersionField), ScheduleState: row.GetString(schedulingstore.ScheduleStateField), Outcome: optionalBookingValue(row.GetString(schedulingstore.OutcomeField)),
		Protected: bookingProtectedDTO{StartAt: utcString(start.Add(-scheduling.ProtectedBuffer)), EndAt: utcString(end.Add(scheduling.ProtectedBuffer))},
	}
}

func optionalBookingValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
