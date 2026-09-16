// Package businesspolicyapi translates the validated global policy into role-scoped HTTP responses without owning policy rules or landing routes.
package businesspolicyapi

import (
	"errors"
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// BusinessPolicy is the English DTO returned by both authenticated policy routes.
type BusinessPolicy struct {
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
	ContractReplacementDeadlineDays  int    `json:"contract_replacement_deadline_days"`
	MonthlyPaymentDueDay             int    `json:"monthly_payment_due_day"`
	ContractEndMonth                 int    `json:"contract_end_month"`
	ContractEndDay                   int    `json:"contract_end_day"`
}

var (
	errUnauthenticated = errors.New("business policy authentication is required")
	errUnauthorized    = errors.New("business policy account is not allowed")
	errInvalidPolicy   = errors.New("business policy is invalid")
)

// RegisterRoutes binds authenticated teacher and learner policy reads.
func RegisterRoutes(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/teachers/business-policy", func(event *core.RequestEvent) error {
			return servePolicy(event, teacherRealm)
		})
		e.Router.GET("/api/learners/business-policy", func(event *core.RequestEvent) error {
			return servePolicy(event, learnerRealm)
		})
		return e.Next()
	})
}

type realm string

const (
	teacherRealm realm = "teacher"
	learnerRealm realm = "learner"
)

func servePolicy(e *core.RequestEvent, want realm) error {
	if _, err := authenticatedCaller(e, want); err != nil {
		return policyError(e, err)
	}
	policy := businesspolicy.Current()
	if err := policy.Validate(); err != nil {
		return policyError(e, errInvalidPolicy)
	}
	return e.JSON(http.StatusOK, policyDTO(policy))
}

func authenticatedCaller(e *core.RequestEvent, want realm) (*core.Record, error) {
	if e.Auth == nil || e.Auth.Collection() == nil {
		return nil, missingIdentityError(e, want)
	}
	collection, cookieName := authRealm(want)
	cookie, err := e.Request.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return nil, missingIdentityError(e, want)
	}
	fromCookie, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if !validIdentity(e.Auth, fromCookie, collection, tokenErr) {
		return nil, errUnauthorized
	}
	return e.Auth, nil
}

func missingIdentityError(e *core.RequestEvent, want realm) error {
	if hasCookie(e, oppositeCookie(want)) {
		return errUnauthorized
	}
	return errUnauthenticated
}

func validIdentity(auth, fromCookie *core.Record, collection string, tokenErr error) bool {
	return tokenErr == nil && fromCookie != nil && fromCookie.Id == auth.Id && auth.Collection().Name == collection && auth.Verified()
}

func hasCookie(e *core.RequestEvent, name string) bool {
	cookie, err := e.Request.Cookie(name)
	return err == nil && cookie.Value != ""
}

func oppositeCookie(want realm) string {
	if want == teacherRealm {
		return sessioncookie.LearnerName
	}
	return sessioncookie.TeacherName
}

func authRealm(want realm) (string, string) {
	if want == teacherRealm {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName
}

func policyDTO(policy businesspolicy.Policy) BusinessPolicy {
	snapshot := policy.Snapshot()
	return BusinessPolicy{
		Version: snapshot.Version, Currency: snapshot.Currency,
		AdHocPriceMinor: snapshot.AdHocPriceMinor, PackagePriceMinor: snapshot.PackagePriceMinor, RegularLessonPriceMinor: snapshot.RegularLessonPriceMinor,
		LessonDurationMinutes: snapshot.LessonDurationMinutes, StartGridMinutes: snapshot.StartGridMinutes, ParticipantBufferMinutes: snapshot.ParticipantBufferMinutes,
		LearnerBookingMinimumHours: snapshot.LearnerBookingMinimumHours, LearnerChangeCutoffHours: snapshot.LearnerChangeCutoffHours, BookingHorizonDays: snapshot.BookingHorizonDays,
		PackageTokenCount: snapshot.PackageTokenCount, PackageValidityDays: snapshot.PackageValidityDays, TeacherCancellationExtensionDays: snapshot.TeacherCancellationExtensionDays,
		ContractMonthlyReschedules: snapshot.ContractMonthlyReschedules, ContractFreeCancellations: snapshot.ContractFreeCancellations, ContractReplacementDeadlineDays: snapshot.ContractReplacementDays,
		MonthlyPaymentDueDay: snapshot.MonthlyPaymentDueDay, ContractEndMonth: snapshot.ContractEndMonth, ContractEndDay: snapshot.ContractEndDay,
	}
}

func policyError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, errUnauthenticated):
		return e.JSON(http.StatusUnauthorized, map[string]string{"code": "unauthenticated", "message": "Authentication is required."})
	case errors.Is(err, errUnauthorized):
		return e.JSON(http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "The account is not allowed to access this resource."})
	default:
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The business policy is unavailable."})
	}
}
