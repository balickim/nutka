// Provides isolated teacher and learner authentication with server-managed sessions and closed native collection routes.
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/security"
)

var errUnauthenticated = errors.New("unauthenticated")

type authRealm struct {
	collection string
	cookieName string
	mePath     string
	logoutPath string
}

var learnerRealm = authRealm{
	collection: authconfig.LearnersCollectionName,
	cookieName: sessioncookie.LearnerName,
	mePath:     "/api/learners/auth/me",
	logoutPath: "/api/learners/auth/logout",
}

var teacherRealm = authRealm{
	collection: authconfig.TeachersCollectionName,
	cookieName: sessioncookie.TeacherName,
	mePath:     "/api/teachers/auth/me",
	logoutPath: "/api/teachers/auth/logout",
}

func registerDualPersonaAuth(app *pocketbase.PocketBase) {
	registerAuthRealms(app, []authRealm{teacherRealm, learnerRealm})
}

func registerAuthRealms(app *pocketbase.PocketBase, realms []authRealm) {
	app.OnRecordAuthWithPasswordRequest().BindFunc(func(e *core.RecordAuthWithPasswordRequestEvent) error {
		_, ok := realmForCollection(realms, e.Collection)
		if !ok {
			return e.Next()
		}
		if err := requireIntent(e.RequestEvent); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordAuthRequest().BindFunc(func(e *core.RecordAuthRequestEvent) error {
		var collection *core.Collection
		if e.Record != nil {
			collection = e.Record.Collection()
		}
		realm, ok := realmForCollection(realms, collection)
		if !ok || e.AuthMethod == "" {
			return e.Next()
		}
		if err := requireIntent(e.RequestEvent); err != nil {
			return err
		}
		if !isVerifiedRecord(e.Record, realm) {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"message": "Failed to authenticate.",
				"error":   "invalid login credentials",
			})
		}

		token := e.Token
		e.Token = ""
		e.Response = &sessioncookie.InjectingWriter{
			ResponseWriter: e.Response,
			Cookie:         sessioncookie.Build(realm.cookieName, token),
		}
		if err := e.Next(); err != nil {
			return err
		}
		e.Record.Set("last_login_at", time.Now().UTC().Format(time.RFC3339))
		if err := e.App.Save(e.Record); err != nil {
			e.App.Logger().Warn("failed to record auth login", "collection", realm.collection, "record", e.Record.Id, "error", err)
		}
		return nil
	})

	app.OnRecordAuthRefreshRequest().BindFunc(func(e *core.RecordAuthRefreshRequestEvent) error {
		if _, ok := realmForCollection(realms, e.Collection); !ok {
			return e.Next()
		}
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"message": "Auth refresh is disabled.",
			"code":    sessioncookie.AuthRefreshDisabled,
		})
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		for _, realm := range realms {
			registerRealmRoutes(e, realm)
		}
		registerSessionCookieBridge(e, realms)
		registerNativeRealmGuard(e)
		return e.Next()
	})
}

func realmForCollection(realms []authRealm, collection *core.Collection) (authRealm, bool) {
	if collection == nil {
		return authRealm{}, false
	}
	for _, realm := range realms {
		if collection.Name == realm.collection {
			return realm, true
		}
	}
	return authRealm{}, false
}

func registerRealmRoutes(e *core.ServeEvent, realm authRealm) {
	e.Router.GET(realm.mePath, func(event *core.RequestEvent) error {
		return realmMe(event, realm)
	})
	e.Router.POST(realm.logoutPath, func(event *core.RequestEvent) error {
		return realmLogout(event, realm)
	})
}

func registerSessionCookieBridge(e *core.ServeEvent, realms []authRealm) {
	e.Router.Bind(&hook.Handler[*core.RequestEvent]{
		Id:       "nutkaPersonaSessionCookieBridge",
		Priority: apis.DefaultLoadAuthTokenMiddlewarePriority - 1,
		Func: func(event *core.RequestEvent) error {
			if event.Auth != nil || event.Request.Header.Get("Authorization") != "" {
				return event.Next()
			}
			realm, ok := realmForRequest(realms, event.Request)
			if !ok {
				return event.Next()
			}
			record, _, err := recordFromCookie(event.App, event.Request, realm)
			if err == nil {
				event.Auth = record
			}
			return event.Next()
		},
	})
}

func registerNativeRealmGuard(e *core.ServeEvent) {
	e.Router.Bind(&hook.Handler[*core.RequestEvent]{
		Id:       "nutkaDeferredPersonaEndpointGuard",
		Priority: apis.DefaultLoadAuthTokenMiddlewarePriority + 1,
		Func: func(event *core.RequestEvent) error {
			if !isDeferredAuthEndpoint(event.Request) || (event.Auth != nil && event.Auth.IsSuperuser()) {
				return event.Next()
			}
			return event.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		},
	})
}

func isDeferredAuthEndpoint(request *http.Request) bool {
	path := request.URL.Path
	for _, collection := range []string{authconfig.LearnersCollectionName, authconfig.TeachersCollectionName} {
		prefix := "/api/collections/" + collection + "/"
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		return path != prefix+"auth-with-password" && path != prefix+"auth-refresh"
	}
	return false
}

func requireIntent(e *core.RequestEvent) error {
	if e.Request.Header.Get(authconfig.AuthIntentHeader) == authconfig.AuthIntentValue {
		return nil
	}
	return e.JSON(http.StatusForbidden, map[string]string{"error": "missing intent header"})
}

func realmMe(e *core.RequestEvent, realm authRealm) error {
	record, expiresAt, err := recordFromCookie(e.App, e.Request, realm)
	if err != nil {
		e.SetCookie(sessioncookie.Clear(realm.cookieName))
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
	}
	record.IgnoreEmailVisibility(true)
	response := map[string]any{"record": record}
	if !expiresAt.IsZero() {
		response["session_expires_at"] = expiresAt.UTC().Format(time.RFC3339)
	}
	return e.JSON(http.StatusOK, response)
}

func realmLogout(e *core.RequestEvent, realm authRealm) error {
	if err := requireIntent(e); err != nil {
		return err
	}
	e.SetCookie(sessioncookie.Clear(realm.cookieName))
	return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func recordFromCookie(app core.App, request *http.Request, realm authRealm) (*core.Record, time.Time, error) {
	cookie, err := request.Cookie(realm.cookieName)
	if err != nil || cookie.Value == "" {
		return nil, time.Time{}, errUnauthenticated
	}
	record, err := app.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if err != nil || !isVerifiedRecord(record, realm) {
		return nil, time.Time{}, errUnauthenticated
	}
	claims, err := security.ParseUnverifiedJWT(cookie.Value)
	if err != nil {
		return record, time.Time{}, nil
	}
	return record, jwtExpiry(claims), nil
}

func realmForRequest(realms []authRealm, request *http.Request) (authRealm, bool) {
	path := request.URL.Path
	for _, realm := range realms {
		if strings.HasPrefix(path, "/api/"+realm.collection+"/") || strings.HasPrefix(path, "/api/collections/"+realm.collection+"/") {
			return realm, true
		}
	}
	return authRealm{}, false
}

func isVerifiedRecord(record *core.Record, realm authRealm) bool {
	return record != nil && record.Collection() != nil && record.Collection().Name == realm.collection && record.Verified()
}

func jwtExpiry(claims map[string]any) time.Time {
	value, ok := claims["exp"]
	if !ok {
		return time.Time{}
	}
	switch value := value.(type) {
	case float64:
		return time.Unix(int64(value), 0)
	case json.Number:
		seconds, err := value.Int64()
		if err == nil {
			return time.Unix(seconds, 0)
		}
	}
	return time.Time{}
}
