// Package historyapi exposes authenticated, assignment-scoped business history reads.
// Route wiring remains opt-in so the application can integrate it with its route plan.
package historyapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes adds only read routes for role-scoped business history.
// No route can update or delete the append-only event collection.
func RegisterRoutes(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/teachers/assignments/{id}/history", func(event *core.RequestEvent) error {
			return serveHistory(event, history.TeacherActor)
		})
		e.Router.GET("/api/learners/assignments/{id}/history", func(event *core.RequestEvent) error {
			return serveHistory(event, history.LearnerActor)
		})
		return e.Next()
	})
}

func serveHistory(e *core.RequestEvent, role history.ActorRole) error {
	account, err := authenticatedCaller(e, role)
	if err != nil {
		return historyError(e, err)
	}
	page, perPage, err := pagination(e)
	if err != nil {
		return historyError(e, err)
	}
	result, err := history.QueryPage(e.App, e.Request.PathValue("id"), history.Viewer{Role: role, ID: account.Id}, page, perPage)
	if err != nil {
		return historyError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

var (
	errUnauthorized = errors.New("history authentication is required")
	errForbidden    = errors.New("history account is not allowed")
	errPagination   = errors.New("history pagination is invalid")
)

func authenticatedCaller(e *core.RequestEvent, role history.ActorRole) (*core.Record, error) {
	if e.Auth == nil || e.Auth.Collection() == nil {
		return nil, errUnauthorized
	}
	collection, cookieName := authRealm(role)
	cookie, err := e.Request.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return nil, missingHistoryIdentity(e, role)
	}
	fromCookie, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if !validHistoryIdentity(e.Auth, fromCookie, collection, tokenErr) {
		return nil, errForbidden
	}
	return e.Auth, nil
}

func missingHistoryIdentity(e *core.RequestEvent, role history.ActorRole) error {
	if _, err := e.Request.Cookie(oppositeCookie(role)); err == nil {
		return errForbidden
	}
	return errUnauthorized
}

func validHistoryIdentity(auth, cookie *core.Record, collection string, tokenErr error) bool {
	return tokenErr == nil && cookie != nil && cookie.Id == auth.Id && auth.Collection().Name == collection && auth.Verified()
}

func oppositeCookie(role history.ActorRole) string {
	if role == history.TeacherActor {
		return sessioncookie.LearnerName
	}
	return sessioncookie.TeacherName
}

func authRealm(role history.ActorRole) (string, string) {
	if role == history.TeacherActor {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName
}

func pagination(e *core.RequestEvent) (int, int, error) {
	page, err := queryInt(e, "page", 1)
	if err != nil {
		return 0, 0, errPagination
	}
	perPage := 20
	perPageValue := e.Request.URL.Query().Get("per_page")
	if perPageValue == "" {
		perPageValue = e.Request.URL.Query().Get("perPage")
	}
	if perPageValue != "" {
		perPage, err = strconv.Atoi(perPageValue)
	}
	if err != nil || page < 1 || perPage < 1 || perPage > 100 {
		return 0, 0, errPagination
	}
	return page, perPage, nil
}

func queryInt(e *core.RequestEvent, name string, fallback int) (int, error) {
	value := e.Request.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func historyError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, errUnauthorized):
		return e.JSON(http.StatusUnauthorized, map[string]string{"code": "unauthenticated", "message": "Authentication is required."})
	case errors.Is(err, errForbidden), errors.Is(err, history.ErrHistoryUnauthorized):
		return e.JSON(http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "The account is not allowed to access this resource."})
	case errors.Is(err, errPagination), errors.Is(err, history.ErrHistoryPagination):
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_pagination", "message": "The history pagination is invalid."})
	default:
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The history request failed."})
	}
}
