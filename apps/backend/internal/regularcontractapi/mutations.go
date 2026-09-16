// This file routes authenticated contract mutations to one command handler per domain operation.
package regularcontractapi

import (
	"net/http"
	"strings"

	"github.com/balickim/nutka/apps/backend/internal/domain"
)

func (h *Handler) mutate(w http.ResponseWriter, r *http.Request, p Principal, parts []string) {
	role := roleForPath(parts[1])
	if !authorizedMutationRole(role, p.Role) {
		writeError(w, http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.")
		return
	}
	if assignmentActivationPath(parts, role) {
		h.activate(w, r, p, parts[3])
		return
	}
	if !contractMutationPath(parts) {
		writeError(w, http.StatusNotFound, "not_found", "The contract resource was not found.")
		return
	}
	action := strings.ToLower(parts[4])
	if action == "notice" {
		h.notice(w, r, p, parts[3])
		return
	}
	if role != domain.TeacherActor {
		h.forbidden(w)
		return
	}
	h.mutateTeacher(w, r, p, parts[3], action)
}

func authorizedMutationRole(routeRole, principalRole domain.ActorRole) bool {
	known := routeRole == domain.TeacherActor || routeRole == domain.LearnerActor
	return known && routeRole == principalRole
}

func assignmentActivationPath(parts []string, role domain.ActorRole) bool {
	return len(parts) == 5 && parts[2] == "assignments" && parts[4] == "contracts" && role == domain.TeacherActor
}

func contractMutationPath(parts []string) bool {
	return len(parts) == 5 && parts[2] == "contracts"
}

func (h *Handler) mutateTeacher(w http.ResponseWriter, r *http.Request, p Principal, id, action string) {
	switch action {
	case "schedule":
		h.changeSchedule(w, r, p, id)
	case "renew":
		h.renew(w, r, p, id)
	case "amendments":
		h.amend(w, r, p, id)
	case "early-end":
		h.earlyEnd(w, r, p, id)
	case "correction":
		h.correction(w, r, p, id)
	default:
		writeError(w, http.StatusNotFound, "not_found", "The contract resource was not found.")
	}
}

func roleForPath(value string) domain.ActorRole {
	if value == "teachers" {
		return domain.TeacherActor
	}
	if value == "learners" {
		return domain.LearnerActor
	}
	return domain.ActorRole(value)
}

func (h *Handler) forbidden(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource.")
}
