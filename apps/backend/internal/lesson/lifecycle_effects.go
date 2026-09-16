// This file applies package, contract, and ledger consequences for lifecycle commands.
package lesson

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func applyReschedule(c *Command, replacement scheduling.Interval) error {
	ref := commercial.LessonReference{ID: c.Lesson.ID, AssignmentID: c.Lesson.AssignmentID, TeacherID: c.Lesson.TeacherID, LearnerID: c.Lesson.LearnerID, StartAt: replacement.Start}
	if c.Lesson.Plan == commercial.PackagePlan {
		if c.Package == nil {
			return ErrInvalidCommand
		}
		if c.Actor.Role == history.TeacherActor {
			cutoff := c.Package.Policy.LearnerChangeCutoffHours
			c.Package.Policy.LearnerChangeCutoffHours = 0
			err := c.Package.Reschedule(c.Lesson.ID, ref, c.Now)
			c.Package.Policy.LearnerChangeCutoffHours = cutoff
			return err
		}
		return c.Package.Reschedule(c.Lesson.ID, ref, c.Now)
	}
	if c.Lesson.Plan == commercial.RegularContract {
		if c.Contract == nil {
			return ErrInvalidCommand
		}
		return c.Contract.Reschedule(regularcontract.RescheduleRequest{OccurrenceID: c.Lesson.ID, AssignmentID: c.Lesson.AssignmentID, ReplacementStart: replacement.Start, Now: c.Now, Actor: regularcontract.Actor{Role: regularcontract.ActorRole(c.Actor.Role), ID: c.Actor.ID}, Available: c.Availability, ParticipantLessonRecords: c.ParticipantLessons})
	}
	return nil
}

func applyCancellation(c *Command, timely bool) (string, error) {
	switch c.Lesson.Plan {
	case commercial.PackagePlan:
		return cancelPackageLesson(c, timely)
	case commercial.RegularContract:
		return cancelContractLesson(c)
	case commercial.AdHoc:
		return cancelAdHocLesson(c)
	}
	return "", ErrInvalidCommand
}

func cancelPackageLesson(c *Command, timely bool) (string, error) {
	if c.Package == nil {
		return "", ErrInvalidCommand
	}
	if c.Actor.Role == history.TeacherActor {
		if err := c.Package.TeacherCancel(c.Lesson.ID, c.Now); err != nil {
			return "", err
		}
		return "package_token_returned_and_extended", nil
	}
	if err := c.Package.CancelLearner(c.Lesson.ID, c.Now); err != nil {
		return "", err
	}
	if timely {
		return "package_token_returned", nil
	}
	return "package_token_used", nil
}

func cancelContractLesson(c *Command) (string, error) {
	if c.Contract == nil {
		return "", ErrInvalidCommand
	}
	err := c.Contract.Cancel(regularcontract.CancellationRequest{OccurrenceID: c.Lesson.ID, Now: c.Now, Actor: regularcontract.Actor{Role: regularcontract.ActorRole(c.Actor.Role), ID: c.Actor.ID}})
	return contractEffect(*c.Contract, c.Lesson.ID, err)
}

func cancelAdHocLesson(c *Command) (string, error) {
	if c.Lesson.Settlement == "" {
		c.Lesson.Settlement = ledger.PendingSettlement
	}
	_, err := ledger.ResolveAdHocNotApplicable(ledger.AdHocLesson{ID: c.Lesson.ID, AssignmentID: c.Lesson.AssignmentID, StartAt: c.Lesson.Interval.Start, EndAt: c.Lesson.Interval.End, UnitPriceMinor: c.Lesson.UnitPriceMinor, Currency: c.Lesson.Currency, SettlementState: c.Lesson.Settlement}, "cancelled", ledger.Actor{Role: string(c.Actor.Role), ID: c.Actor.ID}, c.Now)
	if err != nil {
		return "", err
	}
	c.Lesson.Settlement = ledger.NotApplicable
	return "ad_hoc_not_applicable", nil
}

func contractEffect(contract regularcontract.RegularContract, id string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	for _, occurrence := range contract.Occurrences {
		if occurrence.ID == id && occurrence.BillingOutcome == regularcontract.FreeLearnerCancel {
			return "contract_free_cancellation", nil
		}
	}
	return "contract_billable_cancellation", nil
}

func applyOutcome(c *Command, outcome domain.Outcome) error {
	if c.Lesson.Plan == commercial.PackagePlan {
		if c.Package == nil {
			return ErrInvalidCommand
		}
		if outcome == domain.Completed {
			return c.Package.Complete(c.Lesson.ID)
		}
		return c.Package.NoShow(c.Lesson.ID)
	}
	if c.Lesson.Plan == commercial.RegularContract {
		if c.Contract == nil {
			return ErrInvalidCommand
		}
		return c.Contract.RecordOutcome(c.Lesson.ID, string(outcome), regularcontract.Actor{Role: regularcontract.Teacher, ID: c.Actor.ID}, c.Now)
	}
	if outcome != domain.LearnerNoShow {
		return nil
	}
	_, err := ledger.ResolveAdHocNotApplicable(ledger.AdHocLesson{ID: c.Lesson.ID, AssignmentID: c.Lesson.AssignmentID, StartAt: c.Lesson.Interval.Start, EndAt: c.Lesson.Interval.End, UnitPriceMinor: c.Lesson.UnitPriceMinor, Currency: c.Lesson.Currency, SettlementState: c.Lesson.Settlement}, "learner_no_show", ledger.Actor{Role: ledger.TeacherActor, ID: c.Actor.ID}, c.Now)
	if err == nil {
		c.Lesson.Settlement = ledger.NotApplicable
	}
	return err
}

func correctPackage(p *commercial.Package, lessonID, eventID, reason string, at time.Time) error {
	for _, token := range p.Tokens {
		if token.LessonID == lessonID {
			_, err := p.CorrectTokenState(commercial.CorrectionRequest{TokenID: token.ID, Target: commercial.TokenAvailable, CorrectsEvent: eventID, Reason: reason, At: at})
			return err
		}
	}
	return commercial.ErrTokenNotFound
}
