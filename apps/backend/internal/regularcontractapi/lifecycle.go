// This file adapts schedule, notice, renewal, amendment, early-end, and correction commands to the contract aggregate.
package regularcontractapi

import (
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
)

type scheduleInput struct {
	Weekday     int    `json:"weekday"`
	StartTime   string `json:"start_time"`
	EffectiveOn string `json:"effective_on"`
}
type renewalInput struct {
	StartOn string `json:"start_on"`
}
type amendmentInput struct {
	EffectiveOn string `json:"effective_on"`
	PriceMinor  int64  `json:"price_minor"`
	Currency    string `json:"currency"`
}
type earlyEndInput struct {
	EndOn  string `json:"end_on"`
	Reason string `json:"reason"`
}

func (h *Handler) changeSchedule(w http.ResponseWriter, r *http.Request, p Principal, id string) {
	var input scheduleInput
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
		minute, err := parseMinute(input.StartTime)
		if err != nil {
			return errInvalidRequest
		}
		effective, err := parseDate(input.EffectiveOn, state.Contract.TeacherTimezone)
		if err != nil {
			return errInvalidRequest
		}
		err = state.Contract.ChangeSchedule(regularcontract.ScheduleChangeRequest{Weekday: time.Weekday(input.Weekday), StartMinute: minute, EffectiveOn: effective, Now: now, Actor: regularcontract.Actor{Role: regularcontract.Teacher, ID: p.ID}, Availability: state.Availability, Unavailable: state.Unavailable, Lessons: state.Lessons})
		if err != nil {
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

func (h *Handler) notice(w http.ResponseWriter, r *http.Request, p Principal, id string) {
	if err := decodeEmptyObject(r); err != nil {
		writeDomainError(w, err)
		return
	}
	now := h.Clock().UTC()
	var result regularcontract.RegularContract
	err := h.Store.RunInTransaction(r.Context(), func(tx Transaction) error {
		state, err := tx.ContractContext(r.Context(), id)
		if err != nil || !ownsContract(state.Contract, p) {
			return errUnauthorized
		}
		err = state.Contract.SubmitNotice(regularcontract.Actor{Role: p.Role, ID: p.ID}, now)
		if err != nil {
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

func (h *Handler) earlyEnd(w http.ResponseWriter, r *http.Request, p Principal, id string) {
	var input earlyEndInput
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
		end, err := parseDate(input.EndOn, state.Contract.TeacherTimezone)
		if err != nil {
			return errInvalidRequest
		}
		if err = state.Contract.EndEarly(end, input.Reason, regularcontract.Actor{Role: regularcontract.Teacher, ID: p.ID}, now); err != nil {
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

func (h *Handler) amend(w http.ResponseWriter, r *http.Request, p Principal, id string) {
	var input amendmentInput
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
		effective, err := parseEffectiveMonth(input.EffectiveOn)
		if err != nil {
			return errInvalidRequest
		}
		err = state.Contract.AmendPrice(regularcontract.PriceAmendmentRequest{EffectiveMonth: effective, PriceMinor: input.PriceMinor, Currency: input.Currency, Now: now, Actor: regularcontract.Actor{Role: regularcontract.Teacher, ID: p.ID}})
		if err != nil {
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

func (h *Handler) renew(w http.ResponseWriter, r *http.Request, p Principal, id string) {
	var input renewalInput
	if err := decodeEmptyOrObject(r, &input); err != nil {
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
		if state.Policy.Validate() != nil || state.Policy.Version != businesspolicy.CurrentVersion {
			return errInvalidRequest
		}
		var start time.Time
		if input.StartOn != "" {
			start, err = parseDate(input.StartOn, state.Contract.TeacherTimezone)
			if err != nil {
				return errInvalidRequest
			}
		}
		contractID, idErr := tx.NewContractID(r.Context())
		if idErr != nil {
			return idErr
		}
		result, err = state.Contract.Renew(contractID, start, now, state.Policy, state.Availability, state.Lessons, regularcontract.Actor{Role: regularcontract.Teacher, ID: p.ID})
		if err != nil {
			return err
		}
		return tx.SaveContract(r.Context(), result)
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contractValue(result, p.Role, now))
}

func parseDate(value, timezone string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errInvalidRequest
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, errInvalidRequest
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil || parsed.Format("2006-01-02") != value {
		return time.Time{}, errInvalidRequest
	}
	return parsed, nil
}
