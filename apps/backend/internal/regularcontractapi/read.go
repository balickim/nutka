// This file serves participant-scoped contract and series reads and removes later occurrences from learner responses.
package regularcontractapi

import (
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
)

func (h *Handler) list(w http.ResponseWriter, r *http.Request, p Principal, assignmentID string) {
	role := expectedRole(r)
	if p.Role != role {
		writeError(w, http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.")
		return
	}
	assignment, err := h.Store.Assignment(r.Context(), assignmentID)
	if err != nil || assignment.ID != assignmentID || !owns(assignment, p) {
		writeError(w, http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.")
		return
	}
	contracts, err := h.Store.Contracts(r.Context(), assignmentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "The contract request failed.")
		return
	}
	items := make([]contractDTO, 0, len(contracts))
	for _, contract := range contracts {
		if contract.AssignmentID != assignmentID || !ownsContract(contract, p) {
			continue
		}
		items = append(items, contractValue(contract, p.Role, h.Clock()))
	}
	writeJSON(w, http.StatusOK, map[string]any{"contracts": items})
}

func (h *Handler) series(w http.ResponseWriter, r *http.Request, p Principal, contractID string) {
	if p.Role != expectedRole(r) {
		writeError(w, http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.")
		return
	}
	contract, err := h.Store.Contract(r.Context(), contractID)
	if err != nil || !ownsContract(contract, p) {
		writeError(w, http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.")
		return
	}
	partition, err := contract.Partition(h.Clock())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "The contract series request failed.")
		return
	}
	if p.Role == domain.LearnerActor {
		writeJSON(w, http.StatusOK, map[string]any{"near_term": occurrenceValues(partition.Near), "later": []occurrenceDTO{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"near_term": occurrenceValues(partition.Near), "later": occurrenceValues(partition.Later)})
}

func expectedRole(r *http.Request) domain.ActorRole {
	if len(pathParts(r.URL.Path)) > 1 && pathParts(r.URL.Path)[1] == "learners" {
		return domain.LearnerActor
	}
	return domain.TeacherActor
}

func owns(assignment Assignment, p Principal) bool {
	return (p.Role == domain.TeacherActor && assignment.TeacherID == p.ID) || (p.Role == domain.LearnerActor && assignment.LearnerID == p.ID)
}

func ownsContract(contract regularcontract.RegularContract, p Principal) bool {
	return (p.Role == domain.TeacherActor && contract.TeacherID == p.ID) || (p.Role == domain.LearnerActor && contract.LearnerID == p.ID)
}
