// This file defines stable English contract DTOs and converts domain values without exposing persistence records.
package regularcontractapi

import (
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
)

type contractDTO struct {
	ID                          string `json:"id"`
	Assignment                  string `json:"assignment"`
	Status                      string `json:"status"`
	StartOn                     string `json:"start_on"`
	EndOn                       string `json:"end_on"`
	Weekday                     int    `json:"weekday"`
	StartTime                   string `json:"start_time"`
	PriceMinor                  int64  `json:"price_minor"`
	Currency                    string `json:"currency"`
	PolicyVersion               string `json:"policy_version"`
	EffectiveEndOn              string `json:"effective_end_on"`
	NoticeAt                    string `json:"notice_at,omitempty"`
	RemainingMonthlyReschedules int    `json:"remaining_monthly_reschedules"`
	RemainingFreeCancellations  int    `json:"remaining_free_cancellations"`
}

type occurrenceDTO struct {
	ID                      string `json:"id"`
	Contract                string `json:"contract"`
	Assignment              string `json:"assignment"`
	OriginalLocalDate       string `json:"original_local_date"`
	OriginalStartAt         string `json:"original_start_at"`
	StartAt                 string `json:"start_at"`
	EndAt                   string `json:"end_at"`
	ScheduleState           string `json:"schedule_state"`
	Outcome                 string `json:"outcome,omitempty"`
	BillingOutcome          string `json:"billing_outcome"`
	UnitPriceMinor          int64  `json:"unit_price_minor"`
	Currency                string `json:"currency"`
	IndividuallyRescheduled bool   `json:"individually_rescheduled"`
}

func contractValue(c regularcontract.RegularContract, role domain.ActorRole, now time.Time) contractDTO {
	allowance := c.Allowances(monthKeyForDTO(c, now))
	return contractDTO{
		ID: c.ID, Assignment: c.AssignmentID, Status: string(c.Status), StartOn: localDateText(c.StartOn), EndOn: localDateText(c.EndOn),
		Weekday: int(c.Weekday), StartTime: minuteText(c.StartMinute), PriceMinor: c.PriceMinor, Currency: c.Currency, PolicyVersion: c.PolicySnapshot.Version,
		EffectiveEndOn: localDateText(c.EffectiveEndOn), NoticeAt: instantText(c.NoticeAt), RemainingMonthlyReschedules: allowance.MonthlyReschedulesRemaining,
		RemainingFreeCancellations: allowance.FreeCancellationsRemaining,
	}
}

func occurrenceValues(values []regularcontract.Occurrence) []occurrenceDTO {
	result := make([]occurrenceDTO, 0, len(values))
	for _, value := range values {
		result = append(result, occurrenceValue(value))
	}
	return result
}

func occurrenceValue(o regularcontract.Occurrence) occurrenceDTO {
	return occurrenceDTO{ID: o.ID, Contract: o.ContractID, Assignment: o.AssignmentID, OriginalLocalDate: o.OriginalLocalDate,
		OriginalStartAt: instantText(o.OriginalStartAt), StartAt: instantText(o.Interval.Start), EndAt: instantText(o.Interval.End), ScheduleState: string(o.ScheduleState),
		Outcome: o.Outcome, BillingOutcome: string(o.BillingOutcome), UnitPriceMinor: o.UnitPriceMinor, Currency: o.Currency, IndividuallyRescheduled: o.IndividuallyRescheduled}
}

func monthKeyForDTO(c regularcontract.RegularContract, now time.Time) string {
	location, err := time.LoadLocation(c.TeacherTimezone)
	if err == nil {
		year, month, _ := now.In(location).Date()
		return fmt.Sprintf("%04d-%02d", year, month)
	}
	return ""
}

func localDateText(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func instantText(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func minuteText(minutes int) string {
	return time.Date(0, 1, 1, minutes/60, minutes%60, 0, 0, time.UTC).Format("15:04")
}
