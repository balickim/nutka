// This file decodes activation requests, loads transaction state, and maps regular-contract domain outcomes to HTTP errors.
package regularcontractapi

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

type activationInput struct {
	StartOn          string            `json:"start_on"`
	Weekday          int               `json:"weekday"`
	StartTime        string            `json:"start_time"`
	ConvertLessonIDs []string          `json:"convert_lesson_ids"`
	BackdateReason   string            `json:"backdate_reason"`
	PastOutcomes     map[string]string `json:"past_outcomes"`
}

func (h *Handler) activate(w http.ResponseWriter, r *http.Request, p Principal, assignmentID string) {
	var input activationInput
	if err := decode(r, &input); err != nil {
		writeDomainError(w, err)
		return
	}
	now := h.Clock().UTC()
	var result regularcontract.RegularContract
	err := h.Store.RunInTransaction(r.Context(), func(tx Transaction) error {
		var activationErr error
		result, activationErr = activateContractTransaction(r.Context(), tx, assignmentID, p.ID, input, now)
		return activationErr
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contractValue(result, p.Role, now))
}

func activateContractTransaction(ctx context.Context, tx Transaction, assignmentID, teacherID string, input activationInput, now time.Time) (regularcontract.RegularContract, error) {
	state, err := tx.ActivationContext(ctx, assignmentID)
	if !authorizedActivationState(state, assignmentID, teacherID, err) {
		return regularcontract.RegularContract{}, errUnauthorized
	}
	start, minute, err := activationSchedule(input, state.TeacherTimezone)
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	if !validActivationPolicy(state.Policy) {
		return regularcontract.RegularContract{}, errInvalidRequest
	}
	conversion, err := commercial.DecideContractActivation(state.CommercialState, input.ConvertLessonIDs)
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	contractID := state.ContractID
	if contractID == "" {
		contractID = newID(assignmentID, now)
	}
	result, err := regularcontract.Activate(regularcontract.ActivationRequest{ID: contractID, AssignmentID: assignmentID, TeacherID: state.Assignment.TeacherID, LearnerID: state.Assignment.LearnerID, TeacherTimezone: state.TeacherTimezone, Actor: regularcontract.Actor{Role: regularcontract.Teacher, ID: teacherID}, StartOn: start, Weekday: time.Weekday(input.Weekday), StartMinute: minute, Now: now, Policy: state.Policy, Availability: state.Availability, Unavailable: state.Unavailable, ExistingLessons: state.Lessons, ExistingCommercialObligation: false, BackdateReason: input.BackdateReason, PastOutcomes: input.PastOutcomes})
	if err != nil {
		return regularcontract.RegularContract{}, err
	}
	if err := tx.ConvertContractLessons(ctx, result.ID, conversion.ConvertedLessons); err != nil {
		return regularcontract.RegularContract{}, err
	}
	return result, tx.SaveContract(ctx, result)
}

func authorizedActivationState(state ActivationContext, assignmentID, teacherID string, err error) bool {
	return err == nil && state.Assignment.ID == assignmentID && state.Assignment.TeacherID == teacherID
}

func validActivationPolicy(policy businesspolicy.Policy) bool {
	return policy.Validate() == nil && policy.Version == businesspolicy.CurrentVersion
}

func activationSchedule(input activationInput, timezone string) (time.Time, int, error) {
	start, err := parseDate(input.StartOn, timezone)
	if err != nil {
		return time.Time{}, 0, errInvalidRequest
	}
	minute, err := parseMinute(input.StartTime)
	if err != nil {
		return time.Time{}, 0, errInvalidRequest
	}
	return start, minute, nil
}

var idSequence uint64

func newID(assignment string, now time.Time) string {
	value := fmt.Sprintf("%s:%d:%d", assignment, now.UnixNano(), atomic.AddUint64(&idSequence, 1))
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", digest[:])[:15]
}

func parseMinute(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil || parsed.Second() != 0 {
		return 0, errInvalidRequest
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func writeDomainError(w http.ResponseWriter, err error) {
	status, code, message := domainErrorResponse(err)
	writeError(w, status, code, message)
}

func domainErrorResponse(err error) (int, string, string) {
	if status, code, message, ok := requestErrorResponse(err); ok {
		return status, code, message
	}
	if status, code, message, ok := conflictErrorResponse(err); ok {
		return status, code, message
	}
	if status, code, message, ok := contractRuleErrorResponse(err); ok {
		return status, code, message
	}
	return http.StatusInternalServerError, "internal_error", "The contract request failed."
}

func requestErrorResponse(err error) (int, string, string, bool) {
	if errors.Is(err, errUnauthorized) || errors.Is(err, regularcontract.ErrUnauthorized) {
		return http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.", true
	}
	invalid := errors.Is(err, errInvalidRequest) || errors.Is(err, regularcontract.ErrInvalidContract) || errors.Is(err, scheduling.ErrInvalidGrid) || errors.Is(err, scheduling.ErrInvalidDuration) || errors.Is(err, commercial.ErrInvalidAssignment) || errors.Is(err, commercial.ErrInvalidTimezone) || errors.Is(err, commercial.ErrInvalidPolicy)
	if invalid {
		return http.StatusBadRequest, "invalid_request", "The contract request is invalid.", true
	}
	return 0, "", "", false
}

func conflictErrorResponse(err error) (int, string, string, bool) {
	if errors.Is(err, regularcontract.ErrUnavailable) {
		return http.StatusConflict, "lesson_conflict", "The requested interval is not available.", true
	}
	if errors.Is(err, regularcontract.ErrParticipantConflict) {
		return http.StatusConflict, "lesson_conflict", "The requested interval conflicts with a participant lesson.", true
	}
	overlap := errors.Is(err, regularcontract.ErrCommercialOverlap) || errors.Is(err, commercial.ErrActiveContractOverlap) || errors.Is(err, commercial.ErrContractActive) || errors.Is(err, commercial.ErrAdHocOverlap)
	if overlap {
		return http.StatusConflict, "contract_overlap", "The contract overlaps an existing obligation.", true
	}
	if errors.Is(err, commercial.ErrConversionNotEligible) {
		return http.StatusConflict, "contract_overlap", "Every future ad hoc lesson must be explicitly converted.", true
	}
	return 0, "", "", false
}

func contractRuleErrorResponse(err error) (int, string, string, bool) {
	if errors.Is(err, regularcontract.ErrPastStart) || errors.Is(err, regularcontract.ErrMissingPastOutcome) || errors.Is(err, regularcontract.ErrInvalidEarlyEnd) {
		return http.StatusBadRequest, "correction_reason_required", "A correction reason and required past outcomes are missing.", true
	}
	if errors.Is(err, regularcontract.ErrInvalidAmendment) {
		return http.StatusBadRequest, "invalid_request", "The amendment must start on a future month boundary.", true
	}
	if errors.Is(err, regularcontract.ErrReplacementDeadline) {
		return http.StatusBadRequest, "contract_replacement_deadline", "The replacement exceeds the contract deadline.", true
	}
	return 0, "", "", false
}
