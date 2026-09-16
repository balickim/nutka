// This file exposes authenticated availability preview and resolved-commit routes with strict DTO decoding.
package schedulingapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/availabilityimpact"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type availabilityRequest struct {
	Operation string                        `json:"operation"`
	Target    string                        `json:"target"`
	ID        string                        `json:"id"`
	Rule      *availabilityRuleRequest      `json:"rule"`
	Exception *availabilityExceptionRequest `json:"exception"`
	Proposal  *availabilityProposalRequest  `json:"proposal"`
}

type availabilityProposalRequest struct {
	Operation string                        `json:"operation"`
	Target    string                        `json:"target"`
	ID        string                        `json:"id"`
	Rule      *availabilityRuleRequest      `json:"rule"`
	Exception *availabilityExceptionRequest `json:"exception"`
}

type availabilityRuleRequest struct {
	Weekday   *int    `json:"weekday"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Enabled   *bool   `json:"enabled"`
}

type availabilityExceptionRequest struct {
	StartAt *string `json:"start_at"`
	EndAt   *string `json:"end_at"`
	Kind    *string `json:"kind"`
	Note    *string `json:"note"`
	Enabled *bool   `json:"enabled"`
}

type availabilityCommitRequest struct {
	PreviewVersion string                          `json:"preview_version"`
	Proposal       availabilityProposalRequest     `json:"proposal"`
	Resolutions    []availabilityResolutionRequest `json:"resolutions"`
}

type availabilityResolutionRequest struct {
	Lesson             string  `json:"lesson"`
	Action             string  `json:"action"`
	ReplacementStartAt *string `json:"replacement_start_at"`
}

// RegisterAvailabilityRoutes binds only the preview and resolved-commit endpoints.
// The function is separate so application assembly can control route order explicitly.
func RegisterAvailabilityRoutes(app *pocketbase.PocketBase) {
	RegisterAvailabilityRoutesWithClock(app, time.Now)
}

// RegisterAvailabilityRoutesWithClock binds availability routes with an injectable UTC clock.
func RegisterAvailabilityRoutesWithClock(app *pocketbase.PocketBase, clock func() time.Time) {
	service, err := NewAvailabilityService(businesspolicy.Current(), clock, nil)
	if err != nil {
		return
	}
	RegisterAvailabilityRoutesWithService(app, service)
}

// RegisterAvailabilityRoutesWithService binds availability routes to an explicit application service.
func RegisterAvailabilityRoutesWithService(app *pocketbase.PocketBase, service *AvailabilityService) {
	if service == nil {
		return
	}
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/teachers/availability/preview", func(event *core.RequestEvent) error { return previewAvailability(event, service) })
		e.Router.POST("/api/teachers/availability/commit", func(event *core.RequestEvent) error { return commitAvailability(event, service) })
		return e.Next()
	})
}

func previewAvailability(e *core.RequestEvent, service *AvailabilityService) error {
	if err := requireMutation(e); err != nil {
		return availabilityError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return availabilityError(e, err)
	}
	var request availabilityRequest
	if err := bindBody(e, &request); err != nil {
		return availabilityError(e, err)
	}
	mutation, err := requestMutation(request)
	if err != nil {
		return availabilityError(e, err)
	}
	preview, err := service.Preview(e.App, teacher.Id, mutation)
	if err != nil {
		return availabilityError(e, err)
	}
	return e.JSON(http.StatusOK, preview)
}

func commitAvailability(e *core.RequestEvent, service *AvailabilityService) error {
	if err := requireMutation(e); err != nil {
		return availabilityError(e, err)
	}
	teacher, err := caller(e, "teacher")
	if err != nil {
		return availabilityError(e, err)
	}
	var request availabilityCommitRequest
	if err := bindBody(e, &request); err != nil {
		return availabilityError(e, err)
	}
	mutation, err := proposalMutation(request.Proposal)
	if err != nil {
		return availabilityError(e, err)
	}
	resolutions, err := requestResolutions(request.Resolutions)
	if err != nil {
		return availabilityError(e, err)
	}
	result, err := service.Commit(e.App, teacher.Id, mutation, request.PreviewVersion, resolutions)
	if err != nil {
		return availabilityError(e, err)
	}
	response := map[string]any{"preview": result.Preview}
	if result.Rule != nil {
		response["availability_rule"] = ruleValue(result.Rule)
	}
	if result.Exception != nil {
		response["availability_exception"] = exceptionValue(result.Exception)
	}
	return e.JSON(http.StatusOK, response)
}

func requestMutation(request availabilityRequest) (AvailabilityMutation, error) {
	if request.Proposal != nil {
		if request.Operation != "" || request.Target != "" || request.ID != "" || request.Rule != nil || request.Exception != nil {
			return AvailabilityMutation{}, errInvalid
		}
		return proposalMutation(*request.Proposal)
	}
	return AvailabilityMutation{Operation: normalizedOperation(request.Operation), Target: normalizedTarget(request.Target, request.Operation), ID: request.ID, Rule: requestRule(request.Rule), Exception: requestException(request.Exception)}, nil
}

func proposalMutation(request availabilityProposalRequest) (AvailabilityMutation, error) {
	if request.Operation == "" {
		return AvailabilityMutation{}, errInvalid
	}
	return AvailabilityMutation{Operation: normalizedOperation(request.Operation), Target: normalizedTarget(request.Target, request.Operation), ID: request.ID, Rule: requestRule(request.Rule), Exception: requestException(request.Exception)}, nil
}

func normalizedOperation(value string) availabilityimpact.Operation {
	for _, operation := range []availabilityimpact.Operation{availabilityimpact.Create, availabilityimpact.Update, availabilityimpact.Enable, availabilityimpact.Disable, availabilityimpact.Delete} {
		if value == string(operation) || value == string(operation)+"_rule" || value == string(operation)+"_exception" {
			return operation
		}
	}
	return availabilityimpact.Operation(value)
}

func normalizedTarget(target, operation string) availabilityimpact.Target {
	if target == "recurring_rule" {
		return availabilityimpact.RecurringRule
	}
	if target == string(availabilityimpact.RecurringRule) || target == string(availabilityimpact.Exception) {
		return availabilityimpact.Target(target)
	}
	if len(operation) > 5 && operation[len(operation)-5:] == "rule" {
		return availabilityimpact.RecurringRule
	}
	if len(operation) > 9 && operation[len(operation)-9:] == "exception" {
		return availabilityimpact.Exception
	}
	return availabilityimpact.Target(target)
}

func requestRule(value *availabilityRuleRequest) *RuleMutation {
	if value == nil {
		return nil
	}
	return &RuleMutation{Weekday: value.Weekday, StartTime: value.StartTime, EndTime: value.EndTime, Enabled: value.Enabled}
}

func requestException(value *availabilityExceptionRequest) *ExceptionMutation {
	if value == nil {
		return nil
	}
	return &ExceptionMutation{StartAt: value.StartAt, EndAt: value.EndAt, Kind: value.Kind, Note: value.Note, Enabled: value.Enabled}
}

func requestResolutions(values []availabilityResolutionRequest) ([]Resolution, error) {
	result := make([]Resolution, 0, len(values))
	for _, value := range values {
		if value.Lesson == "" || (value.Action != string(availabilityimpact.Cancel) && value.Action != string(availabilityimpact.Reschedule)) {
			return nil, errInvalidResolution
		}
		resolution := Resolution{LessonID: value.Lesson, Action: availabilityimpact.ResolutionAction(value.Action)}
		if value.ReplacementStartAt != nil {
			parsed, err := parseInstant(*value.ReplacementStartAt)
			if err != nil {
				return nil, errInvalidResolution
			}
			resolution.ReplacementStart = parsed
		}
		result = append(result, resolution)
	}
	return result, nil
}

func availabilityError(e *core.RequestEvent, err error) error {
	status, code, message := http.StatusInternalServerError, "internal_error", "The availability request failed."
	switch {
	case errors.Is(err, errUnauthorized):
		status, code, message = http.StatusUnauthorized, "unauthenticated", "Authentication is required."
	case errors.Is(err, errForbidden):
		status, code, message = http.StatusForbidden, "unauthorized", "The account is not allowed to access this resource."
	case errors.Is(err, errIntent):
		status, code, message = http.StatusForbidden, "missing_intent", "The mutation intent header is required."
	case errors.Is(err, errStalePreview):
		status, code, message = http.StatusConflict, "stale_preview", "The availability preview is no longer current."
	case errors.Is(err, errIncompletePreview):
		status, code, message = http.StatusConflict, "unresolved_obligations", "Every near-term lesson conflict requires a resolution."
	case errors.Is(err, errInvalidResolution):
		status, code, message = http.StatusBadRequest, "invalid_resolution", "The availability resolution is invalid."
	case errors.Is(err, errConflict):
		status, code, message = http.StatusConflict, "conflict", "The requested availability change conflicts with a scheduled lesson."
	case errors.Is(err, errGrid):
		status, code, message = http.StatusBadRequest, "invalid_grid", "Availability times must align to the policy grid."
	case errors.Is(err, errInvalid), errors.Is(err, availabilityimpact.ErrInvalidProposal), errors.Is(err, availabilityimpact.ErrInvalidInput), errors.Is(err, availabilityimpact.ErrInvalidAvailability):
		status, code, message = http.StatusBadRequest, "invalid_request", "The availability request is invalid."
	}
	return e.JSON(status, map[string]string{"code": code, "message": message})
}
