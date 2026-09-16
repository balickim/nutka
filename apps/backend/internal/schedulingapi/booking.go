// This file translates learner and teacher flexible-booking commands into one atomic commercial lesson mutation.
package schedulingapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type bookingInput struct {
	StartAt            *string         `json:"start_at"`
	ConfirmShortNotice *bool           `json:"confirm_short_notice,omitempty"`
	DurationMinutes    json.RawMessage `json:"duration_minutes,omitempty"`
}

// RegisterBookingRoutes adds the teacher route. The learner route is part of the existing scheduling route set.
func RegisterBookingRoutes(app *pocketbase.PocketBase, clock Clock) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/teachers/assignments/{id}/book", func(event *core.RequestEvent) error { return flexibleBookingAt(event, "teacher", clock) })
		return e.Next()
	})
}

func learnerBookingAt(e *core.RequestEvent, clock Clock) error {
	return flexibleBookingAt(e, "learner", clock)
}
func teacherBookingAt(e *core.RequestEvent, clock Clock) error {
	return flexibleBookingAt(e, "teacher", clock)
}

func flexibleBookingAt(e *core.RequestEvent, role string, clock Clock) error {
	command, err := bookingCommandFromRequest(e, role, clock)
	if err != nil {
		return bookingError(e, err)
	}
	lesson, err := createFlexibleBooking(e.App, command)
	if err != nil {
		return bookingError(e, err)
	}
	return e.JSON(http.StatusCreated, bookingLessonValue(lesson))
}

func bookingCommandFromRequest(e *core.RequestEvent, role string, clock Clock) (bookingCommand, error) {
	if err := requireMutation(e); err != nil {
		return bookingCommand{}, err
	}
	account, err := caller(e, role)
	if err != nil {
		return bookingCommand{}, err
	}
	var input bookingInput
	if err := bindBody(e, &input); err != nil {
		return bookingCommand{}, err
	}
	now := clock().UTC()
	start, err := validatedBookingStart(input, role, now)
	if err != nil {
		return bookingCommand{}, err
	}
	return bookingCommand{AssignmentID: e.Request.PathValue("id"), Actor: history.Actor{Role: history.ActorRole(role), ID: account.Id}, Role: role, StartAt: start, Now: now}, nil
}

func validatedBookingStart(input bookingInput, role string, now time.Time) (time.Time, error) {
	if input.DurationMinutes != nil {
		return time.Time{}, errDurationOverride
	}
	if input.StartAt == nil || (role == "learner" && input.ConfirmShortNotice != nil) {
		return time.Time{}, errInvalid
	}
	start, err := parseInstant(*input.StartAt)
	if err != nil {
		return time.Time{}, err
	}
	if role == "teacher" && start.Sub(now) < businesspolicy.Current().LearnerBookingMinimum && (input.ConfirmShortNotice == nil || !*input.ConfirmShortNotice) {
		return time.Time{}, errShortNotice
	}
	return start, nil
}

type bookingCommand struct {
	AssignmentID string
	Actor        history.Actor
	Role         string
	StartAt      time.Time
	Now          time.Time
}

var (
	errShortNotice         = errors.New("teacher short-notice confirmation is required")
	errPlanPrecedence      = errors.New("the requested flexible lesson is blocked by plan precedence")
	errPackageUnavailable  = errors.New("no eligible package token is available")
	errPackageExpired      = errors.New("the package is expired for this lesson start")
	errBookingNotAvailable = errors.New("the requested interval is not available")
	errBookingConflict     = errors.New("the requested interval conflicts with a participant lesson")
	errDurationOverride    = errors.New("booking duration override is not supported")
)

func createFlexibleBooking(app core.App, command bookingCommand) (*core.Record, error) {
	lessonMutationMu.Lock()
	defer lessonMutationMu.Unlock()
	var result *core.Record
	err := app.RunInTransaction(func(tx core.App) error {
		assignment, err := ownedAssignment(tx, command.AssignmentID, command.Role, command.Actor.ID)
		if err != nil || !assignment.GetBool(schedulingstore.ActiveField) {
			return errForbidden
		}
		teacherID, learnerID := assignment.GetString("teacher"), assignment.GetString("learner")
		policy := businesspolicy.Current()
		interval, err := bookingInterval(command.StartAt, command.Now, command.Role, policy)
		if err != nil {
			return err
		}
		state, err := loadBookingState(tx, assignment, command.Now, command.StartAt)
		if err != nil {
			return err
		}
		decision, err := commercial.EvaluateEligibility(commercial.EligibilityInput{Assignment: commercial.Assignment{ID: assignment.Id, TeacherID: teacherID, LearnerID: learnerID}, Now: command.Now, LessonStart: command.StartAt, Contract: state.contract, Packages: state.packages})
		if err != nil {
			return mapEligibilityError(err)
		}
		if err := validateBookingAvailability(tx, command.Now, interval, teacherID); err != nil {
			return err
		}
		if err := validateParticipantConflicts(tx, interval, teacherID, learnerID, ""); err != nil {
			return errBookingConflict
		}
		result, err = persistBooking(tx, assignment, command, interval, decision, state)
		return err
	})
	return result, err
}

func bookingInterval(start, now time.Time, role string, policy businesspolicy.Policy) (scheduling.Interval, error) {
	intervalPolicy := scheduling.IntervalPolicyFromBusinessPolicy(policy)
	if role == "teacher" {
		intervalPolicy.BookingMinimum = 0
	}
	interval, err := scheduling.ValidateBookingStart(start, now, 0, intervalPolicy)
	if err != nil {
		return scheduling.Interval{}, mapValidationError(err)
	}
	return interval, nil
}

func validateBookingAvailability(app core.App, now time.Time, interval scheduling.Interval, teacherID string) error {
	if err := validateAvailability(app, now, interval, teacherID); err != nil {
		if errors.Is(err, errConflict) {
			return errBookingNotAvailable
		}
		return err
	}
	return nil
}

func mapEligibilityError(err error) error {
	switch {
	case errors.Is(err, commercial.ErrContractActive), errors.Is(err, commercial.ErrPlanPrecedence):
		return errPlanPrecedence
	case errors.Is(err, commercial.ErrPackageUnavailable):
		return errPackageUnavailable
	case errors.Is(err, commercial.ErrPackageExpired):
		return errPackageExpired
	default:
		return err
	}
}

func bookingError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, errShortNotice):
		return respond(e, http.StatusBadRequest, "short_notice_confirmation_required", "Teacher confirmation is required for a start inside the learner booking minimum.")
	case errors.Is(err, errDurationOverride):
		return respond(e, http.StatusBadRequest, "duration_override", "Commercial lessons always use the policy duration.")
	case errors.Is(err, errPlanPrecedence):
		return respond(e, http.StatusBadRequest, "plan_precedence", "The active lesson plan does not permit flexible booking.")
	case errors.Is(err, errPackageUnavailable):
		return respond(e, http.StatusConflict, "package_token_exhausted", "No eligible package token is available.")
	case errors.Is(err, errPackageExpired):
		return respond(e, http.StatusBadRequest, "package_expired", "The package is not valid for the requested lesson start.")
	case errors.Is(err, errBookingNotAvailable), errors.Is(err, errBookingConflict), errors.Is(err, errConflict):
		return respond(e, http.StatusConflict, "lesson_conflict", "The requested interval conflicts with a participant lesson or availability.")
	default:
		return handleError(e, err)
	}
}
