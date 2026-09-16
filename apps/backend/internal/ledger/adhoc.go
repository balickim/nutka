// This file defines ad hoc settlement transitions and cancellation/no-show resolution.
// It keeps payment history immutable while returning a new lesson projection.
package ledger

import "time"

// AdHocLesson carries only fields needed to settle an ad hoc obligation.
type AdHocLesson struct {
	ID              string
	AssignmentID    string
	StartAt         time.Time
	EndAt           time.Time
	UnitPriceMinor  int64
	Currency        string
	SettlementState SettlementState
	Transitions     []SettlementTransition
}

type SettlementTransition struct {
	From      SettlementState
	To        SettlementState
	Actor     Actor
	At        time.Time
	Reason    string
	EntryType EntryType
}

type AdHocSettlementCommand struct {
	State SettlementState
	Actor Actor
	At    time.Time
}

type AdHocSettlementDecision struct {
	Lesson     AdHocLesson
	Transition SettlementTransition
	Entry      *FinancialEntry
}

// SettleAdHoc records a teacher-submitted paid or intentional nonpayment choice.
// A later payment may move intentional nonpayment to paid, but no other reversal is allowed.
func SettleAdHoc(lesson AdHocLesson, command AdHocSettlementCommand) (AdHocSettlementDecision, error) {
	if lesson.SettlementState == "" {
		lesson.SettlementState = PendingSettlement
	}
	if err := validateSettlement(lesson, command); err != nil {
		return AdHocSettlementDecision{}, err
	}
	entryType := EntrySettlementPaid
	if command.State == IntentionallyUnpaid {
		entryType = EntrySettlementUnpaid
	}
	transition := SettlementTransition{From: lesson.SettlementState, To: command.State, Actor: command.Actor, At: command.At.UTC(), EntryType: entryType}
	lesson.SettlementState = command.State
	lesson.Transitions = append(copyTransitions(lesson.Transitions), transition)
	entry := &FinancialEntry{AssignmentID: lesson.AssignmentID, EntryType: entryType, AmountMinor: lesson.UnitPriceMinor, Currency: lesson.Currency, RelatedLessonID: lesson.ID, Actor: command.Actor, EventAt: command.At.UTC()}
	return AdHocSettlementDecision{Lesson: lesson, Transition: transition, Entry: entry}, nil
}

func validateSettlement(lesson AdHocLesson, command AdHocSettlementCommand) error {
	if command.Actor.Role != TeacherActor {
		return ErrTeacherRequired
	}
	if command.Actor.ID == "" || lesson.UnitPriceMinor <= 0 || !validCurrency(lesson.Currency) {
		return ErrInvalidMoney
	}
	if command.At.IsZero() || !lesson.EndAt.IsZero() && command.At.Before(lesson.EndAt) {
		return ErrLessonNotEnded
	}
	return validateSettlementTransition(lesson.SettlementState, command.State)
}

func validateSettlementTransition(current, target SettlementState) error {
	if target != Paid && target != IntentionallyUnpaid {
		return ErrInvalidState
	}
	if current == PendingSettlement {
		return nil
	}
	if current == IntentionallyUnpaid && target == Paid {
		return nil
	}
	return ErrInvalidTransition
}

// ResolveAdHocNotApplicable closes cancellation and no-show work without creating debt.
func ResolveAdHocNotApplicable(lesson AdHocLesson, outcome string, actor Actor, at time.Time) (AdHocSettlementDecision, error) {
	if lesson.SettlementState == "" {
		lesson.SettlementState = PendingSettlement
	}
	if err := validateNotApplicable(lesson, outcome, actor, at); err != nil {
		return AdHocSettlementDecision{}, err
	}
	transition := SettlementTransition{From: lesson.SettlementState, To: NotApplicable, Actor: actor, At: at.UTC(), Reason: outcome, EntryType: EntrySettlementNAA}
	lesson.SettlementState = NotApplicable
	lesson.Transitions = append(copyTransitions(lesson.Transitions), transition)
	entry := &FinancialEntry{AssignmentID: lesson.AssignmentID, EntryType: EntrySettlementNAA, AmountMinor: 0, Currency: lesson.Currency, RelatedLessonID: lesson.ID, Reason: outcome, Actor: actor, EventAt: at.UTC()}
	return AdHocSettlementDecision{Lesson: lesson, Transition: transition, Entry: entry}, nil
}

func validateNotApplicable(lesson AdHocLesson, outcome string, actor Actor, at time.Time) error {
	if !validNotApplicableOutcome(outcome) {
		return ErrInvalidState
	}
	if !actorCanResolve(actor, outcome) {
		return ErrTeacherRequired
	}
	if lesson.SettlementState != PendingSettlement {
		return ErrInvalidTransition
	}
	if (actor.Role != SystemActor && actor.ID == "") || lesson.UnitPriceMinor <= 0 || !validCurrency(lesson.Currency) || at.IsZero() {
		return ErrInvalidMoney
	}
	return nil
}

func validNotApplicableOutcome(outcome string) bool {
	return outcome == "cancelled" || outcome == "learner_no_show" || outcome == "teacher_cancelled"
}

func actorCanResolve(actor Actor, outcome string) bool {
	if outcome == "cancelled" {
		return actor.Role == TeacherActor || actor.Role == LearnerActor || actor.Role == SystemActor
	}
	return actor.Role == TeacherActor || actor.Role == SystemActor
}

func copyTransitions(values []SettlementTransition) []SettlementTransition {
	return append([]SettlementTransition(nil), values...)
}
