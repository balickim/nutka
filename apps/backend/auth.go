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

func registerLearnerAuth(app *pocketbase.PocketBase) {
	app.OnRecordAuthWithPasswordRequest().BindFunc(func(e *core.RecordAuthWithPasswordRequestEvent) error {
		if e.Collection == nil || e.Collection.Name != authconfig.LearnersCollectionName {
			return e.Next()
		}
		if err := requireIntent(e.RequestEvent); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordAuthRequest().BindFunc(func(e *core.RecordAuthRequestEvent) error {
		if e.Record == nil || e.Record.Collection().Name != authconfig.LearnersCollectionName {
			return e.Next()
		}
		if e.AuthMethod == "" {
			return e.Next()
		}
		if err := requireIntent(e.RequestEvent); err != nil {
			return err
		}
		if !e.Record.Verified() {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"message": "Failed to authenticate.",
				"error":   "invalid login credentials",
			})
		}

		token := e.Token
		e.Token = ""
		e.Response = &sessioncookie.InjectingWriter{
			ResponseWriter: e.Response,
			Cookie:         sessioncookie.Build(token),
		}

		err := e.Next()
		if err == nil {
			e.Record.Set("last_login_at", time.Now().UTC().Format(time.RFC3339))
			if saveErr := e.App.Save(e.Record); saveErr != nil {
				e.App.Logger().Warn("failed to record learner login", "learner", e.Record.Id, "error", saveErr)
			}
		}
		return err
	})

	app.OnRecordAuthRefreshRequest().BindFunc(func(e *core.RecordAuthRefreshRequestEvent) error {
		if e.Collection == nil || e.Collection.Name != authconfig.LearnersCollectionName {
			return e.Next()
		}
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"message": "Auth refresh is disabled.",
			"code":    sessioncookie.AuthRefreshDisabled,
		})
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/auth/me", learnerMe)
		e.Router.POST("/api/auth/logout", learnerLogout)
		e.Router.Bind(&hook.Handler[*core.RequestEvent]{
			Id:       "nutkaLearnerSessionCookieBridge",
			Priority: apis.DefaultLoadAuthTokenMiddlewarePriority - 1,
			Func: func(e *core.RequestEvent) error {
				if e.Auth != nil || e.Request.Header.Get("Authorization") != "" {
					return e.Next()
				}
				cookie, err := e.Request.Cookie(authconfig.SessionCookieName)
				if err == nil && cookie.Value != "" {
					record, findErr := e.App.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
					if findErr == nil && isVerifiedLearner(record) {
						e.Auth = record
					}
				}
				return e.Next()
			},
		})
		// PocketBase's admin UI uses the native collection routes, including
		// /api/collections/learners/records. Run this guard after PB resolves
		// Authorization so valid superusers can use those routes while the
		// deferred learner self-service surface remains hidden.
		e.Router.Bind(&hook.Handler[*core.RequestEvent]{
			Id:       "nutkaDeferredLearnerEndpointGuard",
			Priority: apis.DefaultLoadAuthTokenMiddlewarePriority + 1,
			Func: func(e *core.RequestEvent) error {
				if !isDeferredLearnerEndpoint(e.Request) || (e.Auth != nil && e.Auth.IsSuperuser()) {
					return e.Next()
				}
				return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
			},
		})
		return e.Next()
	})
}

func isDeferredLearnerEndpoint(request *http.Request) bool {
	const prefix = "/api/collections/learners/"
	if !strings.HasPrefix(request.URL.Path, prefix) {
		return false
	}
	return request.URL.Path != prefix+"auth-with-password" && request.URL.Path != prefix+"auth-refresh"
}

func requireIntent(e *core.RequestEvent) error {
	if e.Request.Header.Get(authconfig.AuthIntentHeader) == authconfig.AuthIntentValue {
		return nil
	}
	return e.JSON(http.StatusForbidden, map[string]string{"error": "missing intent header"})
}

func learnerMe(e *core.RequestEvent) error {
	record, expiresAt, err := learnerFromCookie(e.App, e.Request)
	if err != nil {
		e.SetCookie(sessioncookie.Clear())
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
	}
	// The learner is authenticated for this request, so the home view may show
	// the account identity even when email visibility is disabled publicly.
	record.IgnoreEmailVisibility(true)

	response := map[string]any{"record": record}
	if !expiresAt.IsZero() {
		response["session_expires_at"] = expiresAt.UTC().Format(time.RFC3339)
	}
	return e.JSON(http.StatusOK, response)
}

func learnerLogout(e *core.RequestEvent) error {
	if err := requireIntent(e); err != nil {
		return err
	}
	e.SetCookie(sessioncookie.Clear())
	return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func learnerFromCookie(app core.App, request *http.Request) (*core.Record, time.Time, error) {
	cookie, err := request.Cookie(authconfig.SessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, time.Time{}, errUnauthenticated
	}
	record, err := app.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
	if err != nil || !isVerifiedLearner(record) {
		return nil, time.Time{}, errUnauthenticated
	}

	claims, err := security.ParseUnverifiedJWT(cookie.Value)
	if err != nil {
		return record, time.Time{}, nil
	}
	return record, jwtExpiry(claims), nil
}

func isVerifiedLearner(record *core.Record) bool {
	return record != nil && record.Collection() != nil &&
		record.Collection().Name == authconfig.LearnersCollectionName && record.Verified()
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
