// This file provides framework-neutral routing, strict JSON decoding, and stable error responses for contract endpoints.
package regularcontractapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/domain"
)

type contextKey string

const principalKey contextKey = "regular-contract-principal"

// Principal is the already authenticated persona for a request.
type Principal struct {
	Role domain.ActorRole
	ID   string
}

// WithPrincipal makes a verified persona available to the standalone HTTP handler.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

func principal(ctx context.Context) (Principal, bool) {
	value, ok := ctx.Value(principalKey).(Principal)
	return value, ok && value.ID != "" && (value.Role == domain.TeacherActor || value.Role == domain.LearnerActor)
}

// Handler serves the regular contract API without coupling it to an authentication framework.
type Handler struct {
	Store Repository
	Clock Clock
}

// New returns a contract API handler with an injectable repository and clock.
func New(store Repository, clock Clock) *Handler {
	if clock == nil {
		clock = time.Now
	}
	return &Handler{Store: store, Clock: clock}
}

// ServeHTTP dispatches the canonical regular contract paths.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if !validContractPath(parts) {
		writeError(w, http.StatusNotFound, "not_found", "The contract resource was not found.")
		return
	}
	p, ok := principal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
		return
	}
	if h.dispatchRead(w, r, p, parts) {
		return
	}
	if r.Method == http.MethodGet {
		writeError(w, http.StatusNotFound, "not_found", "The contract resource was not found.")
		return
	}
	if !contractResourcePath(parts) {
		writeError(w, http.StatusNotFound, "not_found", "The contract resource was not found.")
		return
	}
	if r.Method != http.MethodPost || !hasIntent(r) {
		writeMutationMethodError(w, r.Method)
		return
	}
	h.mutate(w, r, p, parts)
}

func validContractPath(parts []string) bool {
	return len(parts) >= 4 && parts[0] == "api"
}

func contractResourcePath(parts []string) bool {
	return len(parts) >= 4 && (parts[2] == "contracts" || len(parts) == 5 && parts[2] == "assignments" && parts[4] == "contracts")
}

func (h *Handler) dispatchRead(w http.ResponseWriter, r *http.Request, p Principal, parts []string) bool {
	if r.Method != http.MethodGet || len(parts) != 5 {
		return false
	}
	if parts[2] == "assignments" && parts[4] == "contracts" && (parts[1] == "teachers" || parts[1] == "learners") {
		h.list(w, r, p, parts[3])
		return true
	}
	if parts[1] == "teachers" && parts[2] == "contracts" && parts[4] == "series" {
		h.series(w, r, p, parts[3])
		return true
	}
	return false
}

func writeMutationMethodError(w http.ResponseWriter, method string) {
	if method == http.MethodPost {
		writeError(w, http.StatusForbidden, "missing_intent", "The mutation intent header is required.")
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "The HTTP method is not supported.")
}

func pathParts(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func hasIntent(r *http.Request) bool { return r.Header.Get("X-Requested-With") == "fetch" }

func decode(r *http.Request, target any) error {
	if r.Body == nil {
		return errInvalidRequest
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errInvalidRequest
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errInvalidRequest
	}
	return nil
}

func decodeEmptyOrObject(r *http.Request, target any) error {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	return decode(r, target)
}

func decodeEmptyObject(r *http.Request) error {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil || fields == nil || len(fields) != 0 {
		return errInvalidRequest
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errInvalidRequest
	}
	return nil
}

var (
	errInvalidRequest = errors.New("invalid contract request")
	errNotFound       = errors.New("contract resource not found")
	errUnauthorized   = errors.New("contract caller is not a participant")
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}
