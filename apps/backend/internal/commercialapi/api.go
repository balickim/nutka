// Package commercialapi exposes assignment-scoped package commands and commercial summaries.
// It authenticates persona sessions, decodes strict English requests, and delegates state changes to transactions.
package commercialapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Clock supplies the current instant to commands and makes boundary tests deterministic.
type Clock func() time.Time

// RegisterRoutes binds package and assignment commercial routes to PocketBase.
func RegisterRoutes(app *pocketbase.PocketBase) { RegisterRoutesWithClock(app, time.Now) }

// RegisterRoutesWithClock binds routes with an injectable clock for deterministic tests.
func RegisterRoutesWithClock(app *pocketbase.PocketBase, clock Clock) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/teachers/assignments/{id}/packages", func(event *core.RequestEvent) error {
			return listPackages(event, "teacher")
		})
		e.Router.GET("/api/learners/assignments/{id}/packages", func(event *core.RequestEvent) error {
			return listPackages(event, "learner")
		})
		e.Router.POST("/api/teachers/assignments/{id}/packages", func(event *core.RequestEvent) error {
			return purchasePackage(event, clock)
		})
		e.Router.POST("/api/teachers/packages/{id}/close", func(event *core.RequestEvent) error {
			return closePackage(event, clock)
		})
		e.Router.POST("/api/teachers/packages/{id}/correction", func(event *core.RequestEvent) error {
			return correctPackage(event, clock)
		})
		e.Router.GET("/api/teachers/assignments/{id}/commercial-summary", func(event *core.RequestEvent) error {
			return commercialSummary(event, "teacher", clock)
		})
		e.Router.GET("/api/learners/assignments/{id}/commercial-summary", func(event *core.RequestEvent) error {
			return commercialSummary(event, "learner", clock)
		})
		return e.Next()
	})
}

var (
	errUnauthorized = errors.New("commercial authentication is required")
	errForbidden    = errors.New("commercial account is not allowed")
	errIntent       = errors.New("commercial mutation intent is missing")
	errInvalid      = errors.New("commercial request is invalid")
	errConflict     = errors.New("commercial request conflicts with current state")
)

func commercialError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, errUnauthorized):
		return e.JSON(http.StatusUnauthorized, map[string]string{"code": "unauthenticated", "message": "Authentication is required."})
	case errors.Is(err, errForbidden):
		return e.JSON(http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "The account is not allowed to access this resource."})
	case errors.Is(err, errIntent):
		return e.JSON(http.StatusForbidden, map[string]string{"code": "missing_intent", "message": "The mutation intent header is required."})
	case errors.Is(err, errConflict):
		return e.JSON(http.StatusConflict, map[string]string{"code": "package_conflict", "message": "The package state conflicts with this request."})
	case errors.Is(err, errInvalid):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_request", "message": "The commercial request is invalid."})
	default:
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The commercial request failed."})
	}
}

func requireIntent(e *core.RequestEvent) error {
	if e.Request.Header.Get(authconfig.AuthIntentHeader) != authconfig.AuthIntentValue {
		return errIntent
	}
	return nil
}

func oppositePersona(role string) string {
	if role == "teacher" {
		return sessioncookie.LearnerName
	}
	return sessioncookie.TeacherName
}

func personaRealm(role string) (string, string) {
	if role == "teacher" {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName
}

func authenticatedCaller(e *core.RequestEvent, role string) (*core.Record, error) {
	if e.Auth == nil || e.Auth.Collection() == nil {
		return nil, missingCallerError(e, role)
	}
	realm, cookieName := personaRealm(role)
	cookie, err := e.Request.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return nil, missingCallerError(e, role)
	}
	fromCookie, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if !validCallerIdentity(e.Auth, fromCookie, realm, tokenErr) {
		return nil, errForbidden
	}
	return e.Auth, nil
}

func missingCallerError(e *core.RequestEvent, role string) error {
	if _, err := e.Request.Cookie(oppositePersona(role)); err == nil {
		return errForbidden
	}
	return errUnauthorized
}

func validCallerIdentity(auth, fromCookie *core.Record, realm string, tokenErr error) bool {
	return tokenErr == nil && fromCookie != nil && fromCookie.Id == auth.Id && auth.Collection().Name == realm && auth.Verified()
}
