// This file translates authenticated HTTP requests into injected ledger operations.
// It contains no payment, refund, correction, forecast, or authorization business rules.
package ledgerapi

import (
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/pocketbase/pocketbase/core"
)

func serveFinancialWork(e *core.RequestEvent, service Service) error {
	actor, err := authenticatedActor(e, ledger.TeacherActor)
	if err != nil {
		return handleError(e, err)
	}
	result, err := service.FinancialWork(e.Request.Context(), actor)
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func serveFinancialSummary(e *core.RequestEvent, service Service) error {
	actor, err := authenticatedActor(e, ledger.LearnerActor)
	if err != nil {
		return handleError(e, err)
	}
	result, err := service.FinancialSummary(e.Request.Context(), actor)
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func serveFinancialHistory(e *core.RequestEvent, service Service, role string) error {
	actor, err := authenticatedActor(e, role)
	if err != nil {
		return handleError(e, err)
	}
	page, perPage, err := pageValues(e)
	if err != nil {
		return handleError(e, err)
	}
	result, err := service.FinancialHistory(e.Request.Context(), actor, e.Request.PathValue("id"), page, perPage)
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func serveContractMonths(e *core.RequestEvent, service Service, role string) error {
	actor, err := authenticatedActor(e, role)
	if err != nil {
		return handleError(e, err)
	}
	page, perPage, err := pageValues(e)
	if err != nil {
		return handleError(e, err)
	}
	result, err := service.ContractMonths(e.Request.Context(), actor, e.Request.PathValue("id"), page, perPage)
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func serveUnresolvedWork(e *core.RequestEvent, service Service) error {
	actor, err := authenticatedActor(e, ledger.TeacherActor)
	if err != nil {
		return handleError(e, err)
	}
	result, err := service.UnresolvedWork(e.Request.Context(), actor)
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func serveSettlement(e *core.RequestEvent, service Service) error {
	return mutate(e, ledger.TeacherActor, func(actor ledger.Actor) (any, error) {
		var request settlementRequest
		if err := decodeBody(e, &request); err != nil {
			return nil, err
		}
		return service.SettleAdHoc(e.Request.Context(), actor, e.Request.PathValue("id"), ledger.SettlementState(request.Settlement))
	})
}

func serveChargePayment(e *core.RequestEvent, service Service) error {
	return mutate(e, ledger.TeacherActor, func(actor ledger.Actor) (any, error) {
		var request settlementRequest
		if err := decodeBody(e, &request); err != nil {
			return nil, err
		}
		if request.Settlement != string(ledger.Paid) && request.Settlement != string(ledger.IntentionallyUnpaid) {
			return nil, ledger.ErrInvalidState
		}
		return service.PayCharge(e.Request.Context(), actor, e.Request.PathValue("id"), ledger.SettlementState(request.Settlement))
	})
}

func serveChargeRefund(e *core.RequestEvent, service Service) error {
	return mutate(e, ledger.TeacherActor, func(actor ledger.Actor) (any, error) {
		var request refundRequest
		if err := decodeBody(e, &request); err != nil {
			return nil, err
		}
		return service.RefundCharge(e.Request.Context(), actor, e.Request.PathValue("id"), request.AmountMinor, request.Reason)
	})
}

func serveChargeCorrection(e *core.RequestEvent, service Service) error {
	return mutate(e, ledger.TeacherActor, func(actor ledger.Actor) (any, error) {
		var request correctionRequest
		if err := decodeBody(e, &request); err != nil {
			return nil, err
		}
		return service.CorrectCharge(e.Request.Context(), actor, e.Request.PathValue("id"), request.Reason, request.Note)
	})
}

func mutate(e *core.RequestEvent, role string, operation func(ledger.Actor) (any, error)) error {
	if err := requireIntent(e); err != nil {
		return handleError(e, err)
	}
	actor, err := authenticatedActor(e, role)
	if err != nil {
		return handleError(e, err)
	}
	result, err := operation(actor)
	if err != nil {
		return handleError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}
