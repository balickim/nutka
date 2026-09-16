// This file maps domain and transport failures to stable English API errors.
// Responses never expose persistence details or cross-assignment existence.
package ledgerapi

import (
	"errors"
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/pocketbase/pocketbase/core"
)

var errPagination = errors.New("ledger pagination is invalid")

func handleError(e *core.RequestEvent, err error) error {
	status, code, message := errorStatus(err)
	return e.JSON(status, map[string]string{"code": code, "message": message})
}

func errorStatus(err error) (int, string, string) {
	if errors.Is(err, errUnauthenticated) {
		return http.StatusUnauthorized, "unauthenticated", "Authentication is required."
	}
	if errors.Is(err, errForbidden) || errors.Is(err, errOwnership) {
		return http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource."
	}
	if errors.Is(err, errIntent) {
		return http.StatusForbidden, "missing_intent", "The mutation intent header is required."
	}
	return domainErrorStatus(err)
}

func domainErrorStatus(err error) (int, string, string) {
	if errors.Is(err, ledger.ErrInvalidReason) {
		return http.StatusBadRequest, "correction_reason_required", "A correction reason is required."
	}
	if invalidDomainError(err) {
		return http.StatusBadRequest, "invalid_request", "The ledger request is invalid."
	}
	return conflictOrInternal(err)
}

func invalidDomainError(err error) bool {
	return errors.Is(err, errInvalidRequest) || errors.Is(err, errPagination) || errors.Is(err, errNotAdHoc) || errors.Is(err, ledger.ErrInvalidState) || errors.Is(err, ledger.ErrInvalidMoney) || errors.Is(err, ledger.ErrNegativeAmount) || errors.Is(err, ledger.ErrInvalidDate) || errors.Is(err, ledger.ErrCurrencyMismatch)
}

func conflictOrInternal(err error) (int, string, string) {
	if errors.Is(err, ledger.ErrLessonNotEnded) {
		return http.StatusConflict, "lesson_not_ended", "The lesson has not ended."
	}
	if errors.Is(err, ledger.ErrInvalidTransition) {
		return http.StatusConflict, "invalid_transition", "The payment transition is not allowed."
	}
	if errors.Is(err, ledger.ErrInsufficientCredit) || errors.Is(err, ledger.ErrAdjustmentExceedsCharge) {
		return http.StatusConflict, "insufficient_credit", "The credit is not available."
	}
	return http.StatusInternalServerError, "internal_error", "The ledger request failed."
}
