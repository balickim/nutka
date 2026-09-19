// Package personaroute resolves the persona caller, mutation intent, and assignment ownership for custom persona routes.
// It owns the shared 401 and 403 bodies. Feature packages keep their own domain errors.
package personaroute

import (
	"errors"
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase/core"
)

// Role names a persona realm. Its value matches the assignment field and the route prefix.
type Role string

const (
	Teacher Role = "teacher"
	Learner Role = "learner"
)

var (
	ErrUnauthenticated = errors.New("persona authentication is required")
	ErrForbidden       = errors.New("persona account is not allowed")
	ErrIntent          = errors.New("persona mutation intent is missing")
)

// Caller checks the realm cookie and the resolved auth record together so request fields cannot alter identity.
func Caller(e *core.RequestEvent, who Role) (*core.Record, error) {
	collection, cookieName, oppositeCookie := realm(who)
	if e.Auth == nil || e.Auth.Collection() == nil {
		if hasCookie(e, oppositeCookie) {
			return nil, ErrForbidden
		}
		return nil, ErrUnauthenticated
	}
	if !hasCookie(e, cookieName) {
		return nil, ErrForbidden
	}
	cookie, _ := e.Request.Cookie(cookieName)
	fromCookie, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if tokenErr != nil || fromCookie == nil || fromCookie.Id != e.Auth.Id || e.Auth.Collection().Name != collection || !e.Auth.Verified() {
		return nil, ErrForbidden
	}
	return e.Auth, nil
}

func RequireIntent(e *core.RequestEvent) error {
	if e.Request.Header.Get(authconfig.AuthIntentHeader) != authconfig.AuthIntentValue {
		return ErrIntent
	}
	return nil
}

// OwnedAssignment hides the existence of assignments that belong to other accounts.
func OwnedAssignment(app core.App, id string, who Role, accountID string) (*core.Record, error) {
	row, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, id)
	if err != nil || row.GetString(string(who)) != accountID {
		return nil, ErrForbidden
	}
	return row, nil
}

// WriteError writes the shared body for a persona error and reports whether it handled err.
func WriteError(e *core.RequestEvent, err error) (bool, error) {
	switch {
	case errors.Is(err, ErrUnauthenticated):
		return true, e.JSON(http.StatusUnauthorized, map[string]string{"code": "unauthenticated", "message": "Authentication is required."})
	case errors.Is(err, ErrForbidden):
		return true, e.JSON(http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "The account is not allowed to access this resource."})
	case errors.Is(err, ErrIntent):
		return true, e.JSON(http.StatusForbidden, map[string]string{"code": "missing_intent", "message": "The mutation intent header is required."})
	default:
		return false, nil
	}
}

func hasCookie(e *core.RequestEvent, name string) bool {
	cookie, err := e.Request.Cookie(name)
	return err == nil && cookie.Value != ""
}

func realm(who Role) (string, string, string) {
	if who == Teacher {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName, sessioncookie.LearnerName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName, sessioncookie.TeacherName
}
