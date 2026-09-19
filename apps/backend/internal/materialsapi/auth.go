// This file resolves the persona session and the owned assignment for learner materials routes.
// Cookies and auth records are checked together so request fields cannot alter identity.
package materialsapi

import (
	"errors"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase/core"
)

type role string

const (
	teacherRole role = "teacher"
	learnerRole role = "learner"
)

var (
	errUnauthenticated = errors.New("materials authentication is required")
	errForbidden       = errors.New("materials account is not allowed")
	errIntent          = errors.New("materials mutation intent is missing")
	errNotFound        = errors.New("material file is unavailable")
	errInvalid         = errors.New("material is invalid")
)

func authenticatedCaller(e *core.RequestEvent, who role) (*core.Record, error) {
	collection, cookieName, oppositeCookie := realm(who)
	if e.Auth == nil || e.Auth.Collection() == nil {
		if hasCookie(e, oppositeCookie) {
			return nil, errForbidden
		}
		return nil, errUnauthenticated
	}
	if !hasCookie(e, cookieName) {
		return nil, errForbidden
	}
	cookie, _ := e.Request.Cookie(cookieName)
	fromCookie, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if tokenErr != nil || fromCookie == nil || fromCookie.Id != e.Auth.Id || e.Auth.Collection().Name != collection || !e.Auth.Verified() {
		return nil, errForbidden
	}
	return e.Auth, nil
}

func requireIntent(e *core.RequestEvent) error {
	if e.Request.Header.Get(authconfig.AuthIntentHeader) != authconfig.AuthIntentValue {
		return errIntent
	}
	return nil
}

// ownedAssignment hides the existence of assignments that belong to other accounts.
func ownedAssignment(app core.App, id string, who role, accountID string) (*core.Record, error) {
	row, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, id)
	if err != nil || row.GetString(string(who)) != accountID {
		return nil, errForbidden
	}
	return row, nil
}

func hasCookie(e *core.RequestEvent, name string) bool {
	cookie, err := e.Request.Cookie(name)
	return err == nil && cookie.Value != ""
}

func realm(who role) (string, string, string) {
	if who == teacherRole {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName, sessioncookie.LearnerName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName, sessioncookie.TeacherName
}
