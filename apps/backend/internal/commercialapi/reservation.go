// Package commercialapi provides the transactional package-token reservation adapter used by booking.
// A process mutex plus transaction re-read prevents concurrent requests from over-reserving a package.
package commercialapi

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

// TokenReservationCommand identifies an assignment-owned lesson and package.
type TokenReservationCommand struct {
	PackageID    string
	LessonID     string
	AssignmentID string
	Now          time.Time
	Actor        history.Actor
}

// TokenReservation contains the token selected by the atomic reservation.
type TokenReservation struct {
	PackageID string
	TokenID   string
	Ordinal   int
}

// ReservePackageToken reserves the lowest available token and links it to the lesson atomically.
func ReservePackageToken(app core.App, command TokenReservationCommand) (TokenReservation, error) {
	if !validReservationCommand(app, command) {
		return TokenReservation{}, errInvalid
	}
	if command.Actor.Role != history.TeacherActor && command.Actor.Role != history.LearnerActor {
		return TokenReservation{}, errForbidden
	}
	packageMutationMu.Lock()
	defer packageMutationMu.Unlock()
	var result TokenReservation
	err := app.RunInTransaction(func(tx core.App) error {
		var reserveErr error
		result, reserveErr = reservePackageTokenTx(tx, command)
		return reserveErr
	})
	if err != nil {
		if packageReservationConflict(err) {
			return TokenReservation{}, errConflict
		}
		return TokenReservation{}, err
	}
	return result, nil
}

func validReservationCommand(app core.App, command TokenReservationCommand) bool {
	return app != nil && command.PackageID != "" && command.LessonID != "" && !command.Now.IsZero()
}

func packageReservationConflict(err error) bool {
	return errors.Is(err, commercial.ErrPackageUnavailable) || errors.Is(err, commercial.ErrPackageExpired) || errors.Is(err, commercial.ErrPackageClosed)
}

// ReservePackageTokenInTransaction composes token reservation with a caller-owned booking transaction.
// The caller must invoke it while holding the same application mutation boundary as lesson creation.
func ReservePackageTokenInTransaction(tx core.App, command TokenReservationCommand) (TokenReservation, error) {
	if tx == nil || !tx.IsTransactional() {
		return TokenReservation{}, errInvalid
	}
	packageMutationMu.Lock()
	defer packageMutationMu.Unlock()
	return reservePackageTokenTx(tx, command)
}

func reservePackageTokenTx(tx core.App, command TokenReservationCommand) (TokenReservation, error) {
	packageRecord, value, lesson, err := reservationState(tx, command)
	if err != nil {
		return TokenReservation{}, err
	}
	token, err := value.Reserve(commercial.LessonReference{ID: lesson.Id, AssignmentID: lesson.GetString(schedulingstore.AssignmentField), TeacherID: lesson.GetString("teacher"), LearnerID: lesson.GetString("learner"), StartAt: lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()}, command.Now)
	if err != nil {
		return TokenReservation{}, err
	}
	if err := persistReservedToken(tx, token, lesson.Id, command.Now); err != nil {
		return TokenReservation{}, err
	}
	if err := persistPackageLesson(tx, lesson, value, token.ID); err != nil {
		return TokenReservation{}, err
	}
	event, eventErr := history.NewPackageTokenReserved(history.EventInput{AggregateType: "package_token", AggregateID: token.ID, AssignmentID: packageRecord.GetString(schedulingstore.AssignmentField), Actor: command.Actor, EventAt: command.Now.UTC(), RelatedIDs: map[string]string{"lesson": lesson.Id, "package": packageRecord.Id}, NewState: map[string]any{"state": string(token.State), "lesson": lesson.Id}})
	if eventErr != nil {
		return TokenReservation{}, eventErr
	}
	if err := (history.Storage{}).Append(tx, event); err != nil {
		return TokenReservation{}, err
	}
	return TokenReservation{PackageID: packageRecord.Id, TokenID: token.ID, Ordinal: token.Ordinal}, nil
}

func reservationState(tx core.App, command TokenReservationCommand) (*core.Record, commercial.Package, *core.Record, error) {
	packageRecord, err := tx.FindRecordById(schedulingstore.LessonPackagesCollectionName, command.PackageID)
	if err != nil {
		return nil, commercial.Package{}, nil, errForbidden
	}
	value, err := packageFromRecords(tx, packageRecord)
	if err != nil {
		return nil, commercial.Package{}, nil, err
	}
	lesson, err := tx.FindRecordById(schedulingstore.LessonsCollectionName, command.LessonID)
	if err != nil || !reservationOwned(packageRecord, lesson, command) {
		return nil, commercial.Package{}, nil, errForbidden
	}
	return packageRecord, value, lesson, nil
}

func reservationOwned(packageRecord, lesson *core.Record, command TokenReservationCommand) bool {
	assignmentID := lesson.GetString(schedulingstore.AssignmentField)
	if assignmentID != packageRecord.GetString(schedulingstore.AssignmentField) || command.AssignmentID != "" && command.AssignmentID != assignmentID {
		return false
	}
	if command.Actor.Role == history.TeacherActor {
		return command.Actor.ID == lesson.GetString("teacher")
	}
	return command.Actor.Role == history.LearnerActor && command.Actor.ID == lesson.GetString("learner")
}

func persistReservedToken(tx core.App, token commercial.Token, lessonID string, now time.Time) error {
	tokenRecord, err := tx.FindRecordById(schedulingstore.PackageTokensCollectionName, token.ID)
	if err != nil {
		return err
	}
	tokenRecord.Set(schedulingstore.TokenStateField, string(token.State))
	tokenRecord.Set(schedulingstore.LessonField, lessonID)
	tokenRecord.Set(schedulingstore.ChangedAtField, now.UTC().Format(time.RFC3339))
	if err := tx.Save(tokenRecord); err != nil {
		return errConflict
	}
	return nil
}

func persistPackageLesson(tx core.App, lesson *core.Record, value commercial.Package, tokenID string) error {
	encoded, encodeErr := json.Marshal(value.Policy)
	if encodeErr != nil {
		return encodeErr
	}
	lesson.Set(schedulingstore.PlanTypeField, string(commercial.PackagePlan))
	lesson.Set(schedulingstore.PackageTokenField, tokenID)
	lesson.Set(schedulingstore.PolicyVersionField, value.Policy.Version)
	lesson.Set(schedulingstore.PolicySnapshotField, encoded)
	lesson.Set(schedulingstore.UnitPriceMinorField, 0)
	lesson.Set(schedulingstore.CurrencyField, value.Price.Currency)
	if err := tx.Save(lesson); err != nil {
		return errConflict
	}
	return nil
}
