package ledgerapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/sessioncookie"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

type fakeService struct {
	workActor        ledger.Actor
	readActor        ledger.Actor
	settleState      ledger.SettlementState
	called           string
	err              error
	refundAmount     int64
	correctionReason string
}

func (f *fakeService) FinancialWork(_ context.Context, actor ledger.Actor) (FinancialWork, error) {
	f.workActor = actor
	return FinancialWork{}, f.err
}
func (f *fakeService) FinancialSummary(_ context.Context, actor ledger.Actor) (FinancialSummary, error) {
	f.readActor = actor
	return FinancialSummary{}, f.err
}
func (f *fakeService) FinancialHistory(_ context.Context, actor ledger.Actor, _ string, page, perPage int) (FinancialHistoryPage, error) {
	f.readActor = actor
	return FinancialHistoryPage{Page: page, PerPage: perPage, Items: []FinancialEntryView{{ActorRole: ledger.TeacherActor}}}, f.err
}
func (f *fakeService) ContractMonths(_ context.Context, actor ledger.Actor, _ string, _, _ int) (ContractMonthPage, error) {
	f.readActor = actor
	return ContractMonthPage{}, f.err
}
func (f *fakeService) PaymentDue(_ context.Context, actor ledger.Actor, _ string) (PaymentDue, error) {
	f.readActor = actor
	return PaymentDue{}, f.err
}
func (f *fakeService) UnresolvedWork(context.Context, ledger.Actor) (UnresolvedWork, error) {
	return UnresolvedWork{}, f.err
}
func (f *fakeService) SettleAdHoc(_ context.Context, actor ledger.Actor, _ string, state ledger.SettlementState) (LessonPayment, error) {
	f.workActor, f.settleState = actor, state
	return LessonPayment{SettlementState: string(state)}, f.err
}
func (f *fakeService) PayCharge(_ context.Context, actor ledger.Actor, _ string, state ledger.SettlementState) (ChargeView, error) {
	f.workActor, f.settleState = actor, state
	return ChargeView{SettlementState: string(state)}, f.err
}

func (f *fakeService) RefundCharge(_ context.Context, _ ledger.Actor, _ string, amount int64, _ string) (ChargeView, error) {
	f.refundAmount = amount
	return ChargeView{}, f.err
}
func (f *fakeService) CorrectCharge(_ context.Context, _ ledger.Actor, _ string, reason, _ string) (ChargeView, error) {
	f.correctionReason = reason
	return ChargeView{}, f.err
}

func TestTeacherFinancialWorkUsesMatchingTeacherSession(t *testing.T) {
	e, teacher := authEvent(t, "teacher")
	service := &fakeService{}
	if err := serveFinancialWork(e, service); err != nil {
		t.Fatal(err)
	}
	response := e.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusOK || service.workActor.ID != teacher.Id || service.workActor.Role != ledger.TeacherActor {
		t.Fatalf("unexpected response or actor: status=%d actor=%+v", response.Code, service.workActor)
	}
}

func TestLearnerSessionCannotUseTeacherFinancialWork(t *testing.T) {
	e, _ := authEvent(t, "learner")
	service := &fakeService{}
	if err := serveFinancialWork(e, service); err != nil {
		t.Fatal(err)
	}
	response := e.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", response.Code)
	}
}

func TestLearnerHistoryRedactsActorIdentity(t *testing.T) {
	e, _ := authEvent(t, "learner")
	service := &fakeService{}
	if err := serveFinancialHistory(e, service, ledger.LearnerActor); err != nil {
		t.Fatal(err)
	}
	if service.readActor.Role != ledger.LearnerActor || service.readActor.ID == "" {
		t.Fatalf("unexpected learner actor: %+v", service.readActor)
	}
	response := e.Response.(*httptest.ResponseRecorder)
	if strings.Contains(response.Body.String(), "actor_id") || strings.Contains(response.Body.String(), "reason") {
		t.Fatal("learner financial history exposed protected fields")
	}
}

func TestSettlementRejectsUnknownAndIdentityFields(t *testing.T) {
	e, _ := authEvent(t, "teacher")
	e.Request.Method = http.MethodPost
	e.Request.Header.Set("X-Requested-With", "fetch")
	e.Request.Body = ioBody(`{"settlement":"paid","teacher":"spoofed"}`)
	e.Request.ContentLength = int64(len(`{"settlement":"paid","teacher":"spoofed"}`))
	service := &fakeService{}
	if err := serveSettlement(e, service); err != nil {
		t.Fatal(err)
	}
	response := e.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusForbidden || service.settleState != "" {
		t.Fatalf("expected identity rejection, status=%d state=%q", response.Code, service.settleState)
	}
}

func TestChargePaymentRequiresExplicitSettlement(t *testing.T) {
	e, _ := authEvent(t, "teacher")
	e.Request.Method = http.MethodPost
	e.Request.Header.Set("X-Requested-With", "fetch")
	body := `{"settlement":"intentionally_unpaid"}`
	e.Request.Body, e.Request.ContentLength = ioBody(body), int64(len(body))
	service := &fakeService{}
	if err := serveChargePayment(e, service); err != nil {
		t.Fatal(err)
	}
	response := e.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusOK || service.settleState != ledger.IntentionallyUnpaid {
		t.Fatalf("unexpected payment result: status=%d state=%q", response.Code, service.settleState)
	}
}

func TestFinancialHistoryRejectsUnboundedPage(t *testing.T) {
	e, _ := authEvent(t, "teacher")
	e.Request.URL.RawQuery = "page=1&per_page=101"
	if err := serveFinancialHistory(e, &fakeService{}, ledger.TeacherActor); err != nil {
		t.Fatal(err)
	}
	response := e.Response.(*httptest.ResponseRecorder)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad pagination, got %d", response.Code)
	}
}

func TestLearnerSummaryAndContractMonthsUseLearnerRealm(t *testing.T) {
	e, _ := authEvent(t, "learner")
	service := &fakeService{}
	if err := serveFinancialSummary(e, service); err != nil {
		t.Fatal(err)
	}
	if service.readActor.Role != ledger.LearnerActor {
		t.Fatalf("summary used wrong actor: %+v", service.readActor)
	}
	e, _ = authEvent(t, "learner")
	if err := serveContractMonths(e, service, ledger.LearnerActor); err != nil {
		t.Fatal(err)
	}
	if service.readActor.Role != ledger.LearnerActor {
		t.Fatalf("month read used wrong actor: %+v", service.readActor)
	}
}

func TestTeacherMonthsAndUnresolvedWorkUseTeacherRealm(t *testing.T) {
	e, _ := authEvent(t, "teacher")
	service := &fakeService{}
	if err := serveContractMonths(e, service, ledger.TeacherActor); err != nil {
		t.Fatal(err)
	}
	if service.readActor.Role != ledger.TeacherActor {
		t.Fatalf("month read used wrong actor: %+v", service.readActor)
	}
	e, _ = authEvent(t, "teacher")
	if err := serveUnresolvedWork(e, service); err != nil {
		t.Fatal(err)
	}
}

func TestMutationsRequireIntentAndRejectUnknownBodies(t *testing.T) {
	cases := []struct {
		name  string
		serve func(*core.RequestEvent, Service) error
		body  string
	}{
		{name: "settlement", serve: serveSettlement, body: `{"settlement":"paid"}`},
		{name: "payment", serve: serveChargePayment, body: `{"settlement":"paid"}`},
		{name: "refund", serve: serveChargeRefund, body: `{"amount_minor":1,"reason":"r"}`},
		{name: "correction", serve: serveChargeCorrection, body: `{"reason":"r"}`},
	}
	for _, test := range cases {
		t.Run(test.name+" missing intent", func(t *testing.T) {
			e, _ := authEvent(t, "teacher")
			setBody(e, test.body)
			if err := test.serve(e, &fakeService{}); err != nil {
				t.Fatal(err)
			}
			if e.Response.(*httptest.ResponseRecorder).Code != http.StatusForbidden {
				t.Fatalf("expected missing intent rejection, got %d", e.Response.(*httptest.ResponseRecorder).Code)
			}
		})
		t.Run(test.name+" unknown field", func(t *testing.T) {
			e, _ := authEvent(t, "teacher")
			e.Request.Header.Set("X-Requested-With", "fetch")
			setBody(e, test.body[:len(test.body)-1]+`,"unknown":true}`)
			if err := test.serve(e, &fakeService{}); err != nil {
				t.Fatal(err)
			}
			if e.Response.(*httptest.ResponseRecorder).Code != http.StatusBadRequest {
				t.Fatalf("expected unknown field rejection, got %d", e.Response.(*httptest.ResponseRecorder).Code)
			}
		})
	}
}

func TestReadServiceErrorUsesStableInternalCode(t *testing.T) {
	e, _ := authEvent(t, "teacher")
	if err := serveFinancialWork(e, &fakeService{err: errors.New("storage failed")}); err != nil {
		t.Fatal(err)
	}
	if e.Response.(*httptest.ResponseRecorder).Code != http.StatusInternalServerError || !strings.Contains(e.Response.(*httptest.ResponseRecorder).Body.String(), `"code":"internal_error"`) {
		t.Fatalf("unexpected service error response: %s", e.Response.(*httptest.ResponseRecorder).Body.String())
	}
}

func TestServiceOwnershipErrorIsForbidden(t *testing.T) {
	e, _ := authEvent(t, "teacher")
	if err := serveFinancialWork(e, &fakeService{err: errOwnership}); err != nil {
		t.Fatal(err)
	}
	if e.Response.(*httptest.ResponseRecorder).Code != http.StatusForbidden {
		t.Fatalf("expected forbidden ownership error, got %d", e.Response.(*httptest.ResponseRecorder).Code)
	}
}

func TestTeacherSettlementRefundAndCorrectionSuccess(t *testing.T) {
	cases := []struct {
		name  string
		serve func(*core.RequestEvent, Service) error
		body  string
	}{
		{name: "settlement", serve: serveSettlement, body: `{"settlement":"paid"}`},
		{name: "refund", serve: serveChargeRefund, body: `{"amount_minor":500,"reason":"cash returned"}`},
		{name: "correction", serve: serveChargeCorrection, body: `{"reason":"duplicate entry","note":"teacher only"}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			e, _ := authEvent(t, "teacher")
			e.Request.Header.Set("X-Requested-With", "fetch")
			setBody(e, test.body)
			service := &fakeService{}
			if err := test.serve(e, service); err != nil {
				t.Fatal(err)
			}
			if e.Response.(*httptest.ResponseRecorder).Code != http.StatusOK {
				t.Fatalf("expected success, got %d", e.Response.(*httptest.ResponseRecorder).Code)
			}
		})
	}
}

func setBody(e *core.RequestEvent, body string) {
	e.Request.Method = http.MethodPost
	e.Request.Body, e.Request.ContentLength = ioBody(body), int64(len(body))
}

type requestBody struct{ *strings.Reader }

func (b requestBody) Close() error { return nil }

func ioBody(value string) requestBody { return requestBody{Reader: strings.NewReader(value)} }

func authEvent(t *testing.T, realm string) (*core.RequestEvent, *core.Record) {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir(), DefaultDev: false})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	collection := core.NewAuthCollection(realm + "s")
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.SetEmail(realm + "@example.test")
	record.SetPassword("local-password")
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	token, err := record.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/teachers/financial-work", nil)
	cookieName := sessioncookie.TeacherName
	if realm == "learner" {
		cookieName = sessioncookie.LearnerName
	}
	request.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: app, Auth: record, Event: router.Event{Request: request, Response: response}}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return event, record
}
