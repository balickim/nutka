// This file selects the immutable contract policy snapshot for later lifecycle decisions.
package regularcontract

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/domain"
)

func (c RegularContract) lifecyclePolicy() Policy {
	snapshot := c.PolicySnapshot
	if snapshot.Version == "" {
		return c.Policy
	}
	policy := c.Policy
	policy.Version = snapshot.Version
	policy.Currency = snapshot.Currency
	policy.AdHocPrice = domain.Money{Minor: snapshot.AdHocPriceMinor, Currency: snapshot.Currency}
	policy.PackagePrice = domain.Money{Minor: snapshot.PackagePriceMinor, Currency: snapshot.Currency}
	policy.RegularLessonPrice = domain.Money{Minor: snapshot.RegularLessonPriceMinor, Currency: snapshot.Currency}
	policy.LessonDuration = time.Duration(snapshot.LessonDurationMinutes) * time.Minute
	policy.StartGrid = time.Duration(snapshot.StartGridMinutes) * time.Minute
	policy.ParticipantBuffer = time.Duration(snapshot.ParticipantBufferMinutes) * time.Minute
	policy.LearnerBookingMinimum = time.Duration(snapshot.LearnerBookingMinimumHours) * time.Hour
	policy.LearnerChangeCutoff = time.Duration(snapshot.LearnerChangeCutoffHours) * time.Hour
	policy.BookingHorizon = time.Duration(snapshot.BookingHorizonDays) * 24 * time.Hour
	policy.PackageTokenCount = snapshot.PackageTokenCount
	policy.PackageValidityDays = snapshot.PackageValidityDays
	policy.TeacherCancellationExtensionDays = snapshot.TeacherCancellationExtensionDays
	policy.ContractMonthlyReschedules = snapshot.ContractMonthlyReschedules
	policy.ContractFreeCancellations = snapshot.ContractFreeCancellations
	policy.ContractReplacementDays = snapshot.ContractReplacementDays
	policy.MonthlyPaymentDueDay = snapshot.MonthlyPaymentDueDay
	policy.ContractEndMonth = time.Month(snapshot.ContractEndMonth)
	policy.ContractEndDay = snapshot.ContractEndDay
	if policy.Validate() != nil {
		return c.Policy
	}
	return policy
}
