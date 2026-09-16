// This file resolves the matching authenticated persona for ledger routes.
// Cookies and auth records are checked together so request fields cannot alter identity.
package ledgerapi

import (
	"errors"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase/core"
)

var (
	errUnauthenticated = errors.New("ledger authentication is required")
	errForbidden       = errors.New("ledger account is not allowed")
	errIntent          = errors.New("ledger mutation intent is missing")
)

func authenticatedActor(e *core.RequestEvent, role string) (ledger.Actor, error) {
	if e.Auth == nil || e.Auth.Collection() == nil {
		return noAuthResult(e, role)
	}
	collection, cookieName := realm(role)
	cookie, err := e.Request.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return missingRealmResult(e, role, collection)
	}
	fromCookie, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if !validSession(fromCookie, e.Auth, tokenErr, collection) {
		return ledger.Actor{}, errForbidden
	}
	return ledger.Actor{Role: string(role), ID: e.Auth.Id}, nil
}

func noAuthResult(e *core.RequestEvent, role string) (ledger.Actor, error) {
	if hasCookie(e, oppositeCookie(role)) {
		return ledger.Actor{}, errForbidden
	}
	return ledger.Actor{}, errUnauthenticated
}

func missingRealmResult(e *core.RequestEvent, role, collection string) (ledger.Actor, error) {
	if e.Auth.Collection().Name != collection || hasCookie(e, oppositeCookie(role)) {
		return ledger.Actor{}, errForbidden
	}
	return ledger.Actor{}, errUnauthenticated
}

func validSession(fromCookie, auth *core.Record, tokenErr error, collection string) bool {
	return tokenErr == nil && fromCookie != nil && fromCookie.Id == auth.Id && auth.Collection().Name == collection && auth.Verified()
}

func hasCookie(e *core.RequestEvent, name string) bool {
	cookie, err := e.Request.Cookie(name)
	return err == nil && cookie.Value != ""
}

func realm(role string) (string, string) {
	if role == ledger.TeacherActor {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName
}

func oppositeCookie(role string) string {
	if role == ledger.TeacherActor {
		return sessioncookie.LearnerName
	}
	return sessioncookie.TeacherName
}

func requireIntent(e *core.RequestEvent) error {
	if e.Request.Header.Get(authconfig.AuthIntentHeader) != authconfig.AuthIntentValue {
		return errIntent
	}
	return nil
}
