// This file connects the framework-neutral contract handler to PocketBase while preserving realm-specific session checks.
package regularcontractapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes binds all regular contract paths to a PocketBase serve router.
func RegisterRoutes(app *pocketbase.PocketBase, handler *Handler) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		r := e.Router
		r.GET("/api/teachers/assignments/{id}/contracts", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.GET("/api/learners/assignments/{id}/contracts", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.LearnerActor) })
		r.GET("/api/teachers/contracts/{id}/series", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.POST("/api/teachers/assignments/{id}/contracts", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.POST("/api/teachers/contracts/{id}/schedule", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.POST("/api/teachers/contracts/{id}/notice", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.POST("/api/learners/contracts/{id}/notice", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.LearnerActor) })
		r.POST("/api/teachers/contracts/{id}/renew", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.POST("/api/teachers/contracts/{id}/amendments", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.POST("/api/teachers/contracts/{id}/early-end", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		r.POST("/api/teachers/contracts/{id}/correction", func(event *core.RequestEvent) error { return servePocket(event, handler, domain.TeacherActor) })
		return e.Next()
	})
}

func servePocket(e *core.RequestEvent, handler *Handler, role domain.ActorRole) error {
	if e.Auth == nil || e.Auth.Collection() == nil || !e.Auth.Verified() {
		return e.JSON(http.StatusUnauthorized, map[string]string{"code": "unauthenticated", "message": "Authentication is required."})
	}
	collection, cookieName := authRealm(role)
	cookie, err := e.Request.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return missingContractIdentity(e, role)
	}
	account, tokenErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if !validContractIdentity(e.Auth, account, collection, tokenErr) {
		return e.JSON(http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "The account is not allowed to access this resource."})
	}
	request := e.Request.WithContext(WithPrincipal(e.Request.Context(), Principal{Role: role, ID: account.Id}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	var payload any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The contract request failed."})
	}
	return e.JSON(recorder.Code, payload)
}

func missingContractIdentity(e *core.RequestEvent, role domain.ActorRole) error {
	if _, err := e.Request.Cookie(oppositeCookie(role)); err == nil {
		return e.JSON(http.StatusForbidden, map[string]string{"code": "unauthorized", "message": "The account is not allowed to access this resource."})
	}
	return e.JSON(http.StatusUnauthorized, map[string]string{"code": "unauthenticated", "message": "Authentication is required."})
}

func validContractIdentity(auth, cookie *core.Record, collection string, tokenErr error) bool {
	return tokenErr == nil && cookie != nil && cookie.Id == auth.Id && auth.Collection().Name == collection
}

func authRealm(role domain.ActorRole) (string, string) {
	if role == domain.TeacherActor {
		return authconfig.TeachersCollectionName, sessioncookie.TeacherName
	}
	return authconfig.LearnersCollectionName, sessioncookie.LearnerName
}

func oppositeCookie(role domain.ActorRole) string {
	if role == domain.TeacherActor {
		return sessioncookie.LearnerName
	}
	return sessioncookie.TeacherName
}
