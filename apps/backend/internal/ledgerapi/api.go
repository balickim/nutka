// Package ledgerapi exposes role-scoped payment reads and teacher ledger commands.
// Domain decisions and transaction persistence remain behind the injected Service.
package ledgerapi

import (
	"context"
	"strconv"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Service is the application boundary for ledger reads and mutations.
// Every mutation must persist its domain decision and history in one transaction.
type Service interface {
	FinancialWork(context.Context, ledger.Actor) (FinancialWork, error)
	FinancialSummary(context.Context, ledger.Actor) (FinancialSummary, error)
	FinancialHistory(context.Context, ledger.Actor, string, int, int) (FinancialHistoryPage, error)
	ContractMonths(context.Context, ledger.Actor, string, int, int) (ContractMonthPage, error)
	UnresolvedWork(context.Context, ledger.Actor) (UnresolvedWork, error)
	SettleAdHoc(context.Context, ledger.Actor, string, ledger.SettlementState) (LessonPayment, error)
	PayCharge(context.Context, ledger.Actor, string, ledger.SettlementState) (ChargeView, error)
	RefundCharge(context.Context, ledger.Actor, string, int64, string) (ChargeView, error)
	CorrectCharge(context.Context, ledger.Actor, string, string, string) (ChargeView, error)
}

// Clock supplies the instant used only for response projection such as overdue state.
type Clock func() time.Time

// RegisterRoutes binds ledger routes. Callers provide the transaction-aware service.
func RegisterRoutes(app *pocketbase.PocketBase, service Service) {
	RegisterRoutesWithClock(app, service, time.Now)
}

// RegisterRoutesWithClock binds routes with an injectable clock for deterministic tests.
func RegisterRoutesWithClock(app *pocketbase.PocketBase, service Service, clock Clock) {
	if clock == nil {
		clock = time.Now
	}
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		r := e.Router
		r.GET("/api/teachers/financial-work", func(event *core.RequestEvent) error {
			return serveFinancialWork(event, service)
		})
		r.GET("/api/learners/financial-summary", func(event *core.RequestEvent) error {
			return serveFinancialSummary(event, service)
		})
		r.GET("/api/teachers/assignments/{id}/financial-history", func(event *core.RequestEvent) error {
			return serveFinancialHistory(event, service, ledger.TeacherActor)
		})
		r.GET("/api/learners/assignments/{id}/financial-history", func(event *core.RequestEvent) error {
			return serveFinancialHistory(event, service, ledger.LearnerActor)
		})
		r.GET("/api/teachers/contracts/{id}/months", func(event *core.RequestEvent) error {
			return serveContractMonths(event, service, ledger.TeacherActor)
		})
		r.GET("/api/learners/contracts/{id}/months", func(event *core.RequestEvent) error {
			return serveContractMonths(event, service, ledger.LearnerActor)
		})
		r.GET("/api/teachers/unresolved-work", func(event *core.RequestEvent) error {
			return serveUnresolvedWork(event, service)
		})
		r.POST("/api/teachers/lessons/{id}/settlement", func(event *core.RequestEvent) error {
			return serveSettlement(event, service)
		})
		r.POST("/api/teachers/charges/{id}/payment", func(event *core.RequestEvent) error {
			return serveChargePayment(event, service)
		})
		r.POST("/api/teachers/charges/{id}/refund", func(event *core.RequestEvent) error {
			return serveChargeRefund(event, service)
		})
		r.POST("/api/teachers/charges/{id}/correction", func(event *core.RequestEvent) error {
			return serveChargeCorrection(event, service)
		})
		return e.Next()
	})
}

func pageValues(e *core.RequestEvent) (int, int, error) {
	page, perPage := 1, 20
	var err error
	if value := e.Request.URL.Query().Get("page"); value != "" {
		page, err = strconv.Atoi(value)
	}
	if value := e.Request.URL.Query().Get("per_page"); value != "" {
		perPage, err = strconv.Atoi(value)
	}
	if err != nil || page < 1 || perPage < 1 || perPage > 100 {
		return 0, 0, errPagination
	}
	return page, perPage, nil
}
