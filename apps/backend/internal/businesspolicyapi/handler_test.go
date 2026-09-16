package businesspolicyapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestPolicyHandlerReturnsEnglishPolicyForEachRealm(t *testing.T) {
	for _, test := range []struct {
		name string
		want realm
		role string
	}{
		{name: "teacher", want: teacherRealm, role: authconfig.TeachersCollectionName},
		{name: "learner", want: learnerRealm, role: authconfig.LearnersCollectionName},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, account, token := policyAuth(t, test.role)
			request := httptest.NewRequest(http.MethodGet, "/api/"+test.name+"s/business-policy", nil)
			request.AddCookie(&http.Cookie{Name: policyCookie(test.want), Value: token})
			response := httptest.NewRecorder()
			event := policyEvent(app, account, request, response)

			if err := servePolicy(event, test.want); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK {
				t.Fatalf("policy status = %d, body = %s", response.Code, response.Body.String())
			}
			assertPolicyDocument(t, response.Body.Bytes())
		})
	}
}

func TestPolicyHandlerRejectsGuestWithoutPolicyData(t *testing.T) {
	app := newPolicyApp(t)
	request := httptest.NewRequest(http.MethodGet, "/api/learners/business-policy", nil)
	response := httptest.NewRecorder()

	if err := servePolicy(policyEvent(app, nil, request, response), learnerRealm); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("guest policy response: %d %q", response.Code, response.Body.String())
	}
	var errorBody map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &errorBody); err != nil || errorBody["code"] != "unauthenticated" {
		t.Fatalf("guest error response: %s", response.Body.String())
	}
	if body := response.Body.String(); body == "" || containsPolicyField(body) {
		t.Fatalf("guest response exposed policy data: %s", body)
	}
}

func TestPolicyHandlerRejectsOppositeRealm(t *testing.T) {
	app, learner, token := policyAuth(t, authconfig.LearnersCollectionName)
	request := httptest.NewRequest(http.MethodGet, "/api/teachers/business-policy", nil)
	request.AddCookie(&http.Cookie{Name: sessioncookie.LearnerName, Value: token})
	response := httptest.NewRecorder()

	if err := servePolicy(policyEvent(app, learner, request, response), teacherRealm); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusForbidden || response.Body.String() == "" || containsPolicyField(response.Body.String()) {
		t.Fatalf("opposite realm policy response: %d %s", response.Code, response.Body.String())
	}
}

func assertPolicyDocument(t *testing.T, body []byte) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{
		"version": "v1", "currency": "PLN", "ad_hoc_price_minor": float64(8000), "package_price_minor": float64(26000), "regular_lesson_price_minor": float64(5000),
		"lesson_duration_minutes": float64(45), "start_grid_minutes": float64(15), "participant_buffer_minutes": float64(5), "learner_booking_minimum_hours": float64(24), "learner_change_cutoff_hours": float64(24), "booking_horizon_days": float64(14),
		"package_token_count": float64(4), "package_validity_days": float64(60), "teacher_cancellation_extension_days": float64(7), "contract_monthly_reschedules": float64(1), "contract_free_cancellations": float64(2), "contract_replacement_deadline_days": float64(30),
		"monthly_payment_due_day": float64(5), "contract_end_month": float64(6), "contract_end_day": float64(30),
	}
	if len(got) != len(expected) {
		t.Fatalf("policy field count = %d, want %d: %#v", len(got), len(expected), got)
	}
	for key, want := range expected {
		if got[key] != want {
			t.Errorf("policy[%q] = %#v, want %#v", key, got[key], want)
		}
	}
}

func containsPolicyField(body string) bool {
	return len(body) > 0 && (strings.Contains(body, "ad_hoc_price_minor") || strings.Contains(body, "lesson_duration_minutes"))
}

func policyAuth(t *testing.T, collectionName string) (*pocketbase.PocketBase, *core.Record, string) {
	t.Helper()
	app := newPolicyApp(t)
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatal(err)
	}
	account := core.NewRecord(collection)
	account.SetEmail(collectionName + "@example.test")
	account.SetPassword("local-password")
	account.SetVerified(true)
	if err := app.Save(account); err != nil {
		t.Fatal(err)
	}
	token, err := account.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return app, account, token
}

func newPolicyApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir(), DefaultDev: false})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{authconfig.TeachersCollectionName, authconfig.LearnersCollectionName} {
		if err := app.Save(core.NewAuthCollection(name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}

func policyEvent(app *pocketbase.PocketBase, account *core.Record, request *http.Request, response *httptest.ResponseRecorder) *core.RequestEvent {
	return &core.RequestEvent{App: app, Auth: account, Event: router.Event{Request: request, Response: response}}
}

func policyCookie(want realm) string {
	if want == teacherRealm {
		return sessioncookie.TeacherName
	}
	return sessioncookie.LearnerName
}
