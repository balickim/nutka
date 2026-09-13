// Package schedulingapi exposes authorized scheduling resources without exposing PocketBase collections.
package schedulingapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var (
	errUnauthorized    = errors.New("scheduling authorization failed")
	errForbidden       = errors.New("scheduling caller is not allowed")
	errIntent          = errors.New("mutation intent is missing")
	errInvalid         = errors.New("scheduling request is invalid")
	errDuration        = errors.New("scheduling duration is invalid")
	errGrid            = errors.New("scheduling start grid is invalid")
	errConflict        = errors.New("scheduling conflict")
	errHorizon         = errors.New("scheduling horizon failed")
	errRelatedIdentity = errors.New("scheduling assignment identity is unavailable")
)

type Clock func() time.Time

// RegisterRoutes binds all scheduling routes to the PocketBase serve router.
func RegisterRoutes(app *pocketbase.PocketBase) {
	RegisterRoutesWithClock(app, time.Now)
}

// RegisterRoutesWithClock binds routes with an injectable UTC clock for deterministic horizon tests.
func RegisterRoutesWithClock(app *pocketbase.PocketBase, clock Clock) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		r := e.Router
		r.GET("/api/teachers/assignments", assignments)
		r.PATCH("/api/teachers/assignments/{id}", teacherAssignmentUpdate)
		r.GET("/api/learners/assignments", assignments)
		r.GET("/api/learners/assignments/{id}/slots", func(event *core.RequestEvent) error { return learnerSlotsAt(event, clock) })
		r.POST("/api/learners/assignments/{id}/book", func(event *core.RequestEvent) error { return learnerBookingAt(event, clock) })
		r.PATCH("/api/teachers/lessons/{id}/reschedule", func(event *core.RequestEvent) error { return rescheduleLesson(event, "teacher", clock) })
		r.PATCH("/api/learners/lessons/{id}/reschedule", func(event *core.RequestEvent) error { return rescheduleLesson(event, "learner", clock) })
		r.POST("/api/teachers/lessons/{id}/cancel", func(event *core.RequestEvent) error { return cancelLesson(event, "teacher", clock) })
		r.POST("/api/learners/lessons/{id}/cancel", func(event *core.RequestEvent) error { return cancelLesson(event, "learner", clock) })
		r.GET("/api/teachers/cancellation-counters", teacherCancellationCounters)
		r.GET("/api/learners/cancellation-counters", learnerCancellationCounters)

		r.GET("/api/teachers/availability/rules", teacherRules)
		r.POST("/api/teachers/availability/rules", createRule)
		r.PATCH("/api/teachers/availability/rules/{id}", updateRule)
		r.DELETE("/api/teachers/availability/rules/{id}", deleteRule)
		r.GET("/api/teachers/availability/exceptions", teacherExceptions)
		r.POST("/api/teachers/availability/exceptions", createException)
		r.PATCH("/api/teachers/availability/exceptions/{id}", updateException)
		r.DELETE("/api/teachers/availability/exceptions/{id}", deleteException)

		r.GET("/api/teachers/calendar", teacherCalendar)
		r.GET("/api/learners/calendar", learnerCalendar)
		return e.Next()
	})
}

func caller(e *core.RequestEvent, role string) (*core.Record, error) {
	if e.Auth == nil || e.Auth.Collection() == nil {
		if _, err := e.Request.Cookie(oppositeCookie(role)); err == nil {
			return nil, errForbidden
		}
		return nil, errUnauthorized
	}
	want, cookieName := realm(role)
	cookie, cookieErr := e.Request.Cookie(cookieName)
	if cookieErr != nil || cookie.Value == "" {
		return nil, errUnauthorized
	}
	fromCookie, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if !validCookieCaller(fromCookie, e.Auth, tokenErr, want) {
		return nil, errForbidden
	}
	return e.Auth, nil
}

func oppositeCookie(role string) string {
	if role == "teacher" {
		return sessioncookie.LearnerName
	}
	return sessioncookie.TeacherName
}

func realm(role string) (string, string) {
	if role == "teacher" {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName
}
func validCookieCaller(cookie, auth *core.Record, tokenErr error, collection string) bool {
	return tokenErr == nil && cookie != nil && auth != nil && cookie.Id == auth.Id && auth.Collection() != nil && auth.Collection().Name == collection && auth.Verified()
}

func requireMutation(e *core.RequestEvent) error {
	if e.Request.Header.Get(authconfig.AuthIntentHeader) != authconfig.AuthIntentValue {
		return errIntent
	}
	return nil
}

func respond(e *core.RequestEvent, status int, code string, message string) error {
	return e.JSON(status, map[string]string{"code": code, "message": message})
}

func handleError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, errUnauthorized):
		return respond(e, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
	case errors.Is(err, errForbidden):
		return respond(e, http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.")
	case errors.Is(err, errIntent):
		return respond(e, http.StatusForbidden, "missing_intent", "The mutation intent header is required.")
	case errors.Is(err, errDuration), errors.Is(err, scheduling.ErrInvalidDuration):
		return respond(e, http.StatusBadRequest, "invalid_duration", "The duration must be a positive multiple of 15 minutes.")
	case errors.Is(err, errGrid), errors.Is(err, scheduling.ErrInvalidGrid):
		return respond(e, http.StatusBadRequest, "invalid_grid", "The start must be aligned to a 15-minute boundary.")
	case errors.Is(err, errInvalid):
		return respond(e, http.StatusBadRequest, "invalid_request", "The scheduling request is invalid.")
	case errors.Is(err, errHorizon):
		return respond(e, http.StatusBadRequest, "horizon", "The requested time is outside the rolling horizon.")
	case errors.Is(err, errConflict):
		return respond(e, http.StatusConflict, "conflict", "The requested interval conflicts with a scheduled lesson.")
	default:
		return respond(e, http.StatusInternalServerError, "internal_error", "The scheduling request failed.")
	}
}
