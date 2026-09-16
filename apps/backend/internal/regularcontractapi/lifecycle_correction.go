// This file handles contract corrections and validates persisted correction event references.
package regularcontractapi

import (
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
)

type correctionInput struct {
	EventID string `json:"event_id"`
	Reason  string `json:"reason"`
}

func (h *Handler) correction(w http.ResponseWriter, r *http.Request, p Principal, id string) {
	var input correctionInput
	if err := decode(r, &input); err != nil {
		writeDomainError(w, err)
		return
	}
	now := h.Clock().UTC()
	var result regularcontract.RegularContract
	err := h.Store.RunInTransaction(r.Context(), func(tx Transaction) error {
		state, err := tx.ContractContext(r.Context(), id)
		if err != nil || state.Contract.TeacherID != p.ID {
			return errUnauthorized
		}
		eventID, err := correctionEventID(tx, id, input.EventID)
		if err != nil {
			return err
		}
		if err := state.Contract.CorrectEvent(eventID, input.Reason, regularcontract.Actor{Role: regularcontract.Teacher, ID: p.ID}, now); err != nil {
			return err
		}
		result = state.Contract
		return tx.SaveContract(r.Context(), result)
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contractValue(result, p.Role, now))
}

func correctionEventID(tx Transaction, contractID, requested string) (string, error) {
	pocketBase, ok := tx.(*pocketBaseTransaction)
	if !ok {
		return requested, nil
	}
	row, err := pocketBase.app.FindRecordById(schedulingstore.BusinessEventsCollectionName, requested)
	if err != nil {
		return requested, nil
	}
	if row.GetString(schedulingstore.AggregateTypeField) != "contract" || row.GetString(schedulingstore.AggregateIDField) != contractID {
		return "", errUnauthorized
	}
	legacyID := row.GetString(schedulingstore.LegacyEventIDField)
	if legacyID == "" {
		return "", errInvalidRequest
	}
	return legacyID, nil
}

func parseEffectiveMonth(value string) (string, error) {
	if len(value) == 7 {
		parsed, err := time.Parse("2006-01", value)
		if err == nil && parsed.Format("2006-01") == value {
			return value, nil
		}
	}
	if len(value) == 10 {
		parsed, err := time.Parse("2006-01-02", value)
		if err == nil && parsed.Day() == 1 && parsed.Format("2006-01-02") == value {
			return value[:7], nil
		}
	}
	return "", errInvalidRequest
}
