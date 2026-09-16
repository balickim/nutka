// Package businesspolicy defines the validated global policy used by scheduling and commercial domains.
package businesspolicy

import (
	"errors"
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/domain"
)

const (
	CurrentVersion                         = "v1"
	CurrencyPLN                            = "PLN"
	AdHocPriceMinor                  int64 = 8000
	PackagePriceMinor                int64 = 26000
	RegularLessonPriceMinor          int64 = 5000
	LessonDurationMinutes                  = 45
	StartGridMinutes                       = 15
	ParticipantBufferMinutes               = 5
	LearnerBookingMinimumHours             = 24
	LearnerChangeCutoffHours               = 24
	BookingHorizonDays                     = 14
	PackageTokenCount                      = 4
	PackageValidityDays                    = 60
	TeacherCancellationExtensionDays       = 7
	ContractMonthlyReschedules             = 1
	ContractFreeCancellations              = 2
	ContractReplacementDays                = 30
	MonthlyPaymentDueDay                   = 5
	ContractEndMonth                       = time.June
	ContractEndDay                         = 30
)

var (
	ErrInvalidPolicy = errors.New("business policy is invalid")
	currentPolicy    = Policy{
		Version:                          CurrentVersion,
		Currency:                         CurrencyPLN,
		AdHocPrice:                       domain.Money{Minor: AdHocPriceMinor, Currency: CurrencyPLN},
		PackagePrice:                     domain.Money{Minor: PackagePriceMinor, Currency: CurrencyPLN},
		RegularLessonPrice:               domain.Money{Minor: RegularLessonPriceMinor, Currency: CurrencyPLN},
		LessonDuration:                   LessonDurationMinutes * time.Minute,
		StartGrid:                        StartGridMinutes * time.Minute,
		ParticipantBuffer:                ParticipantBufferMinutes * time.Minute,
		LearnerBookingMinimum:            LearnerBookingMinimumHours * time.Hour,
		LearnerChangeCutoff:              LearnerChangeCutoffHours * time.Hour,
		BookingHorizon:                   BookingHorizonDays * 24 * time.Hour,
		PackageTokenCount:                PackageTokenCount,
		PackageValidityDays:              PackageValidityDays,
		TeacherCancellationExtensionDays: TeacherCancellationExtensionDays,
		ContractMonthlyReschedules:       ContractMonthlyReschedules,
		ContractFreeCancellations:        ContractFreeCancellations,
		ContractReplacementDays:          ContractReplacementDays,
		MonthlyPaymentDueDay:             MonthlyPaymentDueDay,
		ContractEndMonth:                 ContractEndMonth,
		ContractEndDay:                   ContractEndDay,
	}
)

// Policy contains all current values that affect a commercial or scheduling decision.
type Policy struct {
	Version                          string
	Currency                         string
	AdHocPrice                       domain.Money
	PackagePrice                     domain.Money
	RegularLessonPrice               domain.Money
	LessonDuration                   time.Duration
	StartGrid                        time.Duration
	ParticipantBuffer                time.Duration
	LearnerBookingMinimum            time.Duration
	LearnerChangeCutoff              time.Duration
	BookingHorizon                   time.Duration
	PackageTokenCount                int
	PackageValidityDays              int
	TeacherCancellationExtensionDays int
	ContractMonthlyReschedules       int
	ContractFreeCancellations        int
	ContractReplacementDays          int
	MonthlyPaymentDueDay             int
	ContractEndMonth                 time.Month
	ContractEndDay                   int
}

// PolicySnapshot stores the compact immutable values needed by later lifecycle decisions.
type PolicySnapshot struct {
	Version                          string `json:"version"`
	Currency                         string `json:"currency"`
	AdHocPriceMinor                  int64  `json:"ad_hoc_price_minor"`
	PackagePriceMinor                int64  `json:"package_price_minor"`
	RegularLessonPriceMinor          int64  `json:"regular_lesson_price_minor"`
	LessonDurationMinutes            int    `json:"lesson_duration_minutes"`
	StartGridMinutes                 int    `json:"start_grid_minutes"`
	ParticipantBufferMinutes         int    `json:"participant_buffer_minutes"`
	LearnerBookingMinimumHours       int    `json:"learner_booking_minimum_hours"`
	LearnerChangeCutoffHours         int    `json:"learner_change_cutoff_hours"`
	BookingHorizonDays               int    `json:"booking_horizon_days"`
	PackageTokenCount                int    `json:"package_token_count"`
	PackageValidityDays              int    `json:"package_validity_days"`
	TeacherCancellationExtensionDays int    `json:"teacher_cancellation_extension_days"`
	ContractMonthlyReschedules       int    `json:"contract_monthly_reschedules"`
	ContractFreeCancellations        int    `json:"contract_free_cancellations"`
	ContractReplacementDays          int    `json:"contract_replacement_days"`
	MonthlyPaymentDueDay             int    `json:"monthly_payment_due_day"`
	ContractEndMonth                 int    `json:"contract_end_month"`
	ContractEndDay                   int    `json:"contract_end_day"`
}

func Current() Policy { return currentPolicy }

func (p Policy) Validate() error {
	validators := []func() error{p.validateIdentity, p.validatePrices, p.validateIntervals, p.validateHorizons, p.validateLimits, p.validateDates}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func (p Policy) validateIdentity() error {
	if p.Version == "" || len(p.Currency) != 3 || p.Currency != CurrencyPLN {
		return fmt.Errorf("%w: version or currency", ErrInvalidPolicy)
	}
	return nil
}

func (p Policy) validatePrices() error {
	for _, price := range []domain.Money{p.AdHocPrice, p.PackagePrice, p.RegularLessonPrice} {
		if price.Minor <= 0 || price.Currency != p.Currency || price.Validate() != nil {
			return fmt.Errorf("%w: money", ErrInvalidPolicy)
		}
	}
	return nil
}

func (p Policy) validateIntervals() error {
	if p.LessonDuration <= 0 || p.StartGrid <= 0 || p.ParticipantBuffer < 0 || p.LessonDuration%p.StartGrid != 0 {
		return fmt.Errorf("%w: intervals", ErrInvalidPolicy)
	}
	return nil
}

func (p Policy) validateHorizons() error {
	if p.LearnerBookingMinimum <= 0 || p.LearnerChangeCutoff <= 0 || p.BookingHorizon <= 0 || p.LearnerBookingMinimum > p.BookingHorizon || p.LearnerChangeCutoff > p.BookingHorizon {
		return fmt.Errorf("%w: horizons", ErrInvalidPolicy)
	}
	return nil
}

func (p Policy) validateLimits() error {
	if p.PackageTokenCount <= 0 || p.PackageValidityDays <= 0 || p.TeacherCancellationExtensionDays <= 0 || p.ContractMonthlyReschedules < 0 || p.ContractFreeCancellations < 0 || p.ContractReplacementDays <= 0 {
		return fmt.Errorf("%w: limits", ErrInvalidPolicy)
	}
	return nil
}

func (p Policy) validateDates() error {
	if p.MonthlyPaymentDueDay < 1 || p.MonthlyPaymentDueDay > 31 || p.ContractEndMonth < time.January || p.ContractEndMonth > time.December || p.ContractEndDay < 1 || p.ContractEndDay > 31 {
		return fmt.Errorf("%w: dates", ErrInvalidPolicy)
	}
	date := time.Date(2024, p.ContractEndMonth, p.ContractEndDay, 0, 0, 0, 0, time.UTC)
	if date.Month() != p.ContractEndMonth || date.Day() != p.ContractEndDay {
		return fmt.Errorf("%w: end date", ErrInvalidPolicy)
	}
	return nil
}

func (p Policy) Snapshot() PolicySnapshot {
	return PolicySnapshot{
		Version: p.Version, Currency: p.Currency,
		AdHocPriceMinor: p.AdHocPrice.Minor, PackagePriceMinor: p.PackagePrice.Minor, RegularLessonPriceMinor: p.RegularLessonPrice.Minor,
		LessonDurationMinutes: int(p.LessonDuration / time.Minute), StartGridMinutes: int(p.StartGrid / time.Minute), ParticipantBufferMinutes: int(p.ParticipantBuffer / time.Minute),
		LearnerBookingMinimumHours: int(p.LearnerBookingMinimum / time.Hour), LearnerChangeCutoffHours: int(p.LearnerChangeCutoff / time.Hour), BookingHorizonDays: int(p.BookingHorizon / (24 * time.Hour)),
		PackageTokenCount: p.PackageTokenCount, PackageValidityDays: p.PackageValidityDays, TeacherCancellationExtensionDays: p.TeacherCancellationExtensionDays,
		ContractMonthlyReschedules: p.ContractMonthlyReschedules, ContractFreeCancellations: p.ContractFreeCancellations, ContractReplacementDays: p.ContractReplacementDays,
		MonthlyPaymentDueDay: p.MonthlyPaymentDueDay, ContractEndMonth: int(p.ContractEndMonth), ContractEndDay: p.ContractEndDay,
	}
}

func CurrentSnapshot() PolicySnapshot { return Current().Snapshot() }

func (s PolicySnapshot) Validate() error {
	return Policy{
		Version: s.Version, Currency: s.Currency,
		AdHocPrice: domain.Money{Minor: s.AdHocPriceMinor, Currency: s.Currency}, PackagePrice: domain.Money{Minor: s.PackagePriceMinor, Currency: s.Currency}, RegularLessonPrice: domain.Money{Minor: s.RegularLessonPriceMinor, Currency: s.Currency},
		LessonDuration: time.Duration(s.LessonDurationMinutes) * time.Minute, StartGrid: time.Duration(s.StartGridMinutes) * time.Minute, ParticipantBuffer: time.Duration(s.ParticipantBufferMinutes) * time.Minute,
		LearnerBookingMinimum: time.Duration(s.LearnerBookingMinimumHours) * time.Hour, LearnerChangeCutoff: time.Duration(s.LearnerChangeCutoffHours) * time.Hour, BookingHorizon: time.Duration(s.BookingHorizonDays) * 24 * time.Hour,
		PackageTokenCount: s.PackageTokenCount, PackageValidityDays: s.PackageValidityDays, TeacherCancellationExtensionDays: s.TeacherCancellationExtensionDays,
		ContractMonthlyReschedules: s.ContractMonthlyReschedules, ContractFreeCancellations: s.ContractFreeCancellations, ContractReplacementDays: s.ContractReplacementDays,
		MonthlyPaymentDueDay: s.MonthlyPaymentDueDay, ContractEndMonth: time.Month(s.ContractEndMonth), ContractEndDay: s.ContractEndDay,
	}.Validate()
}

// ValidateCurrent protects application startup from an internally inconsistent source policy.
func ValidateCurrent() error { return Current().Validate() }
