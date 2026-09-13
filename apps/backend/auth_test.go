package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/database"
	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingapi"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func newTestServer(t *testing.T) (*pocketbase.PocketBase, http.Handler) {
	return newTestServerWithClock(t, time.Now)
}

func newTestServerWithClock(t *testing.T, clock schedulingapi.Clock) (*pocketbase.PocketBase, http.Handler) {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: t.TempDir(),
		DefaultDev:     false,
	})
	database.RegisterMigrate(app)
	configureAppWithClock(app, clock)
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(serveEvent); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	mux, err := baseRouter.BuildMux()
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return app, mux
}

func seedTestLearner(t *testing.T, app *pocketbase.PocketBase, verified bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(authconfig.LearnersCollectionName)
	if err != nil {
		t.Fatalf("find learners: %v", err)
	}
	record := core.NewRecord(collection)
	record.SetEmail("learner@example.test")
	record.Set(authconfig.LearnerNameField, "Test Learner")
	record.SetPassword("local-password")
	record.SetVerified(verified)
	if err := app.Save(record); err != nil {
		t.Fatalf("save learner: %v", err)
	}
}

func seedTestTeacher(t *testing.T, app *pocketbase.PocketBase, verified bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(authconfig.TeachersCollectionName)
	if err != nil {
		t.Fatalf("find teachers: %v", err)
	}
	record := core.NewRecord(collection)
	record.SetEmail("teacher@example.test")
	record.Set(authconfig.TeacherNameField, "Test Teacher")
	record.SetPassword("local-password")
	record.SetVerified(verified)
	if err := app.Save(record); err != nil {
		t.Fatalf("save teacher: %v", err)
	}
}

func request(t *testing.T, server http.Handler, method, path, body string, cookie *http.Cookie, intent bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if intent {
		req.Header.Set(authconfig.AuthIntentHeader, authconfig.AuthIntentValue)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	return response
}

func requestWithAuthorization(t *testing.T, server http.Handler, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", token)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	return response
}

func requestWithCookies(t *testing.T, server http.Handler, method, path string, cookies []*http.Cookie, intent bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if intent {
		req.Header.Set(authconfig.AuthIntentHeader, authconfig.AuthIntentValue)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	return response
}

func TestLearnersMigrationIsClosedAndConfigured(t *testing.T) {
	app, server := newTestServer(t)
	collection, err := app.FindCollectionByNameOrId(authconfig.LearnersCollectionName)
	if err != nil {
		t.Fatal(err)
	}
	if !collection.IsAuth() || collection.AuthToken.Duration != int64(authconfig.SessionDuration/time.Second) {
		t.Fatalf("unexpected learner auth configuration")
	}
	if collection.CreateRule != nil || collection.UpdateRule != nil || collection.DeleteRule != nil {
		t.Fatalf("learner write rules must be closed")
	}
	for _, field := range []string{"name", "last_login_at", "login_metadata"} {
		if collection.Fields.GetByName(field) == nil {
			t.Fatalf("missing learner field %q", field)
		}
	}
	if !collection.Fields.GetByName("last_login_at").GetHidden() || !collection.Fields.GetByName("login_metadata").GetHidden() {
		t.Fatal("login metadata must be hidden")
	}
	if response := request(t, server, http.MethodPost, "/api/collections/learners/records", `{}`, nil, true); response.Code != http.StatusNotFound {
		t.Fatalf("learner self-service must remain disabled, got %d", response.Code)
	}
	if response := request(t, server, http.MethodPost, "/api/collections/teachers/records", `{}`, nil, true); response.Code != http.StatusNotFound {
		t.Fatalf("teacher self-service must remain disabled, got %d", response.Code)
	}
}

func TestPocketBaseSuperuserCanUseLearnerCollectionRoutes(t *testing.T) {
	app, server := newTestServer(t)
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatal(err)
	}
	superuser := core.NewRecord(superusers)
	superuser.SetEmail("admin@example.test")
	superuser.SetPassword("local-password")
	if err := app.Save(superuser); err != nil {
		t.Fatalf("save superuser: %v", err)
	}
	token, err := superuser.NewAuthToken()
	if err != nil {
		t.Fatalf("create superuser token: %v", err)
	}

	response := requestWithAuthorization(t, server, http.MethodGet, "/api/collections/learners/records?page=1&perPage=40", token)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items"`) {
		t.Fatalf("superuser learner collection request failed: %d %s", response.Code, response.Body.String())
	}
	teacherResponse := requestWithAuthorization(t, server, http.MethodGet, "/api/collections/teachers/records?page=1&perPage=40", token)
	if teacherResponse.Code != http.StatusOK || !strings.Contains(teacherResponse.Body.String(), `"items"`) {
		t.Fatalf("superuser teacher collection request failed: %d %s", teacherResponse.Code, teacherResponse.Body.String())
	}
}

func TestLearnerLoginRequiresIntentAndStripsToken(t *testing.T) {
	app, server := newTestServer(t)
	seedTestLearner(t, app, true)
	body := `{"identity":"learner@example.test","password":"local-password"}`
	withoutIntent := request(t, server, http.MethodPost, "/api/collections/learners/auth-with-password", body, nil, false)
	if withoutIntent.Code != http.StatusForbidden || !strings.Contains(withoutIntent.Body.String(), "missing intent header") {
		t.Fatalf("expected intent rejection, got %d %s", withoutIntent.Code, withoutIntent.Body.String())
	}

	response := request(t, server, http.MethodPost, "/api/collections/learners/auth-with-password", body, nil, true)
	if response.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Token != "" {
		t.Fatal("auth token must not be returned in the response")
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessioncookie.LearnerName || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" || cookie.MaxAge != int(authconfig.SessionDuration/time.Second) {
		t.Fatalf("unexpected session cookie: %#v", cookie)
	}
	invalid := request(t, server, http.MethodPost, "/api/collections/learners/auth-with-password", `{"identity":"learner@example.test","password":"wrong"}`, nil, true)
	if invalid.Code != http.StatusBadRequest || len(invalid.Result().Cookies()) != 0 {
		t.Fatalf("invalid credentials must not issue a cookie: %d %#v", invalid.Code, invalid.Result().Cookies())
	}
}

func TestLearnerMeLogoutAndRefresh(t *testing.T) {
	app, server := newTestServer(t)
	seedTestLearner(t, app, true)
	missing := request(t, server, http.MethodGet, "/api/learners/auth/me", "", nil, false)
	if missing.Code != http.StatusUnauthorized || len(missing.Result().Cookies()) != 1 || missing.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("missing /me session must clear its cookie: %d %#v", missing.Code, missing.Result().Cookies())
	}
	login := request(t, server, http.MethodPost, "/api/collections/learners/auth-with-password", `{"identity":"learner@example.test","password":"local-password"}`, nil, true)
	cookie := login.Result().Cookies()[0]

	me := request(t, server, http.MethodGet, "/api/learners/auth/me", "", cookie, false)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), "Test Learner") || !strings.Contains(me.Body.String(), "learner@example.test") || !strings.Contains(me.Body.String(), "session_expires_at") {
		t.Fatalf("unexpected /me response: %d %s", me.Code, me.Body.String())
	}

	refresh := request(t, server, http.MethodPost, "/api/collections/learners/auth-refresh", "", cookie, true)
	if refresh.Code != http.StatusUnauthorized || !strings.Contains(refresh.Body.String(), sessioncookie.AuthRefreshDisabled) {
		t.Fatalf("expected refresh rejection, got %d %s", refresh.Code, refresh.Body.String())
	}

	logout := request(t, server, http.MethodPost, "/api/learners/auth/logout", "", cookie, true)
	if logout.Code != http.StatusOK || len(logout.Result().Cookies()) != 1 || logout.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("unexpected logout response: %d %s", logout.Code, logout.Body.String())
	}
	logoutAgain := request(t, server, http.MethodPost, "/api/learners/auth/logout", "", nil, true)
	if logoutAgain.Code != http.StatusOK {
		t.Fatalf("logout must be idempotent, got %d", logoutAgain.Code)
	}
	withoutIntent := request(t, server, http.MethodPost, "/api/learners/auth/logout", "", cookie, false)
	if withoutIntent.Code != http.StatusForbidden {
		t.Fatalf("expected logout intent rejection, got %d", withoutIntent.Code)
	}
}

func TestUnverifiedLearnerDoesNotReceiveCookie(t *testing.T) {
	app, server := newTestServer(t)
	seedTestLearner(t, app, false)
	response := request(t, server, http.MethodPost, "/api/collections/learners/auth-with-password", `{"identity":"learner@example.test","password":"local-password"}`, nil, true)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid login credentials") || len(response.Result().Cookies()) != 0 {
		t.Fatalf("unverified login must not issue a cookie: %d %#v", response.Code, response.Result().Cookies())
	}
}

func TestTeacherSessionUsesIndependentCookieAndRoutes(t *testing.T) {
	app, server := newTestServer(t)
	seedTestLearner(t, app, true)
	seedTestTeacher(t, app, true)
	learnerLogin := request(t, server, http.MethodPost, "/api/collections/learners/auth-with-password", `{"identity":"learner@example.test","password":"local-password"}`, nil, true)
	teacherLogin := request(t, server, http.MethodPost, "/api/collections/teachers/auth-with-password", `{"identity":"teacher@example.test","password":"local-password"}`, nil, true)
	if learnerLogin.Code != http.StatusOK || teacherLogin.Code != http.StatusOK {
		t.Fatalf("persona login failed: learner=%d teacher=%d", learnerLogin.Code, teacherLogin.Code)
	}
	learnerCookie := learnerLogin.Result().Cookies()[0]
	teacherCookie := teacherLogin.Result().Cookies()[0]
	if learnerCookie.Name != sessioncookie.LearnerName || teacherCookie.Name != sessioncookie.TeacherName {
		t.Fatalf("persona cookies are not independent: learner=%q teacher=%q", learnerCookie.Name, teacherCookie.Name)
	}
	for _, login := range []*httptest.ResponseRecorder{teacherLogin, learnerLogin} {
		var payload struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(login.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Token != "" {
			t.Fatal("persona login must not return bearer tokens")
		}
	}
	if response := request(t, server, http.MethodGet, "/api/teachers/auth/me", "", teacherCookie, false); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Test Teacher") {
		t.Fatalf("teacher session bootstrap failed: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, server, http.MethodGet, "/api/teachers/auth/me", "", learnerCookie, false); response.Code != http.StatusUnauthorized || len(response.Result().Cookies()) != 1 || response.Result().Cookies()[0].Name != sessioncookie.TeacherName || response.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("learner cookie must not authorize teacher session: %d", response.Code)
	}
	if response := request(t, server, http.MethodGet, "/api/learners/auth/me", "", teacherCookie, false); response.Code != http.StatusUnauthorized || len(response.Result().Cookies()) != 1 || response.Result().Cookies()[0].Name != sessioncookie.LearnerName || response.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("teacher cookie must not authorize learner session: %d", response.Code)
	}
	if response := request(t, server, http.MethodGet, "/api/collections/teachers/records", "", learnerCookie, false); response.Code != http.StatusNotFound {
		t.Fatalf("learner cookie must not authorize native teacher routes: %d", response.Code)
	}
	if response := request(t, server, http.MethodGet, "/api/auth/me", "", learnerCookie, false); response.Code != http.StatusNotFound {
		t.Fatalf("legacy auth route must not exist: %d", response.Code)
	}
}

func TestTeacherAuthRejectsUnverifiedAndDisablesRefresh(t *testing.T) {
	app, server := newTestServer(t)
	seedTestTeacher(t, app, false)
	login := request(t, server, http.MethodPost, "/api/collections/teachers/auth-with-password", `{"identity":"teacher@example.test","password":"local-password"}`, nil, true)
	if login.Code != http.StatusBadRequest || len(login.Result().Cookies()) != 0 || !strings.Contains(login.Body.String(), "invalid login credentials") {
		t.Fatalf("unverified teacher must not receive cookie: %d %#v %s", login.Code, login.Result().Cookies(), login.Body.String())
	}
	record, err := app.FindFirstRecordByData(authconfig.TeachersCollectionName, "email", "teacher@example.test")
	if err != nil {
		t.Fatal(err)
	}
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	login = request(t, server, http.MethodPost, "/api/collections/teachers/auth-with-password", `{"identity":"teacher@example.test","password":"local-password"}`, nil, true)
	refresh := request(t, server, http.MethodPost, "/api/collections/teachers/auth-refresh", "", login.Result().Cookies()[0], true)
	if refresh.Code != http.StatusUnauthorized || !strings.Contains(refresh.Body.String(), sessioncookie.AuthRefreshDisabled) {
		t.Fatalf("teacher refresh must be disabled: %d %s", refresh.Code, refresh.Body.String())
	}
}

func TestPersonaLogoutClearsOnlyMatchingSession(t *testing.T) {
	app, server := newTestServer(t)
	seedTestLearner(t, app, true)
	seedTestTeacher(t, app, true)
	learnerLogin := request(t, server, http.MethodPost, "/api/collections/learners/auth-with-password", `{"identity":"learner@example.test","password":"local-password"}`, nil, true)
	teacherLogin := request(t, server, http.MethodPost, "/api/collections/teachers/auth-with-password", `{"identity":"teacher@example.test","password":"local-password"}`, nil, true)
	learnerCookie := learnerLogin.Result().Cookies()[0]
	teacherCookie := teacherLogin.Result().Cookies()[0]
	logout := requestWithCookies(t, server, http.MethodPost, "/api/teachers/auth/logout", []*http.Cookie{learnerCookie, teacherCookie}, true)
	if logout.Code != http.StatusOK || len(logout.Result().Cookies()) != 1 || logout.Result().Cookies()[0].Name != sessioncookie.TeacherName {
		t.Fatalf("teacher logout must clear only teacher cookie: %d %#v", logout.Code, logout.Result().Cookies())
	}
	if learnerMe := request(t, server, http.MethodGet, "/api/learners/auth/me", "", learnerCookie, false); learnerMe.Code != http.StatusOK {
		t.Fatalf("teacher logout must preserve learner session: %d %s", learnerMe.Code, learnerMe.Body.String())
	}
}

func TestSeedCommandOnlyExistsInDevelopment(t *testing.T) {
	t.Setenv("NUTKA_ENV", "production")
	production := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	registerSeedLearnerCommand(production)
	registerSeedTeacherCommand(production)
	for _, child := range production.RootCmd.Commands() {
		if child.Name() == "seed-learner" || child.Name() == "seed-teacher" {
			t.Fatal("seed command must not be registered in production")
		}
	}
}

func TestSeedLearnerCreatesAndUpdatesVerifiedRecord(t *testing.T) {
	t.Setenv("NUTKA_ENV", "development")
	app, _ := newTestServer(t)
	if err := seedLearner(app, "seed@example.test", "local-password", "Seeded Learner"); err != nil {
		t.Fatal(err)
	}
	if err := seedLearner(app, "seed@example.test", "updated-password", "Updated Learner"); err != nil {
		t.Fatal(err)
	}
	record, err := app.FindFirstRecordByData(authconfig.LearnersCollectionName, "email", "seed@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if !record.Verified() || record.GetString(authconfig.LearnerNameField) != "Updated Learner" || !record.ValidatePassword("updated-password") {
		t.Fatal("seed learner did not create/update a verified record")
	}
}

func TestSeedTeacherCreatesAndUpdatesVerifiedRecord(t *testing.T) {
	t.Setenv("NUTKA_ENV", "development")
	app, _ := newTestServer(t)
	if err := seedTeacher(app, "seed-teacher@example.test", "local-password", "Seeded Teacher"); err != nil {
		t.Fatal(err)
	}
	if err := seedTeacher(app, "seed-teacher@example.test", "updated-password", "Updated Teacher"); err != nil {
		t.Fatal(err)
	}
	record, err := app.FindFirstRecordByData(authconfig.TeachersCollectionName, "email", "seed-teacher@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if !record.Verified() || record.GetString(authconfig.TeacherNameField) != "Updated Teacher" || !record.ValidatePassword("updated-password") {
		t.Fatal("seed teacher did not create/update a verified record")
	}
}
