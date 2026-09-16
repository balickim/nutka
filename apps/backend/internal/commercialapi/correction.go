// Package commercialapi persists authorized package token corrections with compensating history.
// Corrections retain the original token and event records and require a non-empty reason.
package commercialapi

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func correctPackage(e *core.RequestEvent, clock Clock) error {
	if err := requireIntent(e); err != nil {
		return commercialError(e, err)
	}
	teacher, err := authenticatedCaller(e, "teacher")
	if err != nil {
		return commercialError(e, err)
	}
	var input correctionRequest
	if err := decodeStrict(e, &input); err != nil {
		return commercialError(e, err)
	}
	now := clock().UTC()
	var result *core.Record
	packageMutationMu.Lock()
	defer packageMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		var correctionErr error
		result, correctionErr = correctPackageTransaction(tx, e.Request.PathValue("id"), teacher.Id, input, now)
		return correctionErr
	})
	if err != nil {
		return commercialError(e, mapDomainError(err))
	}
	value, valueErr := packageValue(e.App, result)
	if valueErr != nil {
		return commercialError(e, valueErr)
	}
	return e.JSON(http.StatusOK, value)
}

func correctPackageTransaction(tx core.App, packageID, teacherID string, input correctionRequest, now time.Time) (*core.Record, error) {
	row, err := ownedPackage(tx, packageID, teacherID)
	if err != nil {
		return nil, err
	}
	value, err := packageFromRecords(tx, row)
	if err != nil {
		return nil, err
	}
	if _, err := value.CorrectTokenState(commercial.CorrectionRequest{TokenID: input.TokenID, Target: input.Target, CorrectsEvent: input.CorrectsEvent, Reason: input.Reason, At: now}); err != nil {
		return nil, err
	}
	if err := validateCorrectionEvent(tx, row, input); err != nil {
		return nil, err
	}
	priorState, err := applyTokenCorrection(tx, input, now)
	if err != nil {
		return nil, err
	}
	event := history.Event{Type: history.AdministrativeCorrection, AggregateType: "package_token", AggregateID: input.TokenID, AssignmentID: row.GetString(schedulingstore.AssignmentField), Actor: history.Actor{Role: history.TeacherActor, ID: teacherID}, EventAt: now, Reason: input.Reason, CorrectsEventID: input.CorrectsEvent, PriorState: map[string]any{"state": priorState}, NewState: map[string]any{"state": string(input.Target)}}
	return row, (history.Storage{}).Append(tx, event)
}

func validateCorrectionEvent(tx core.App, row *core.Record, input correctionRequest) error {
	original, err := tx.FindRecordById(schedulingstore.BusinessEventsCollectionName, input.CorrectsEvent)
	if err != nil || original.GetString(schedulingstore.AssignmentField) != row.GetString(schedulingstore.AssignmentField) {
		return errForbidden
	}
	if original.GetString(schedulingstore.AggregateTypeField) != "package_token" || original.GetString(schedulingstore.AggregateIDField) != input.TokenID || original.GetString(schedulingstore.EventTypeField) == string(history.AdministrativeCorrection) {
		return errInvalid
	}
	_, err = tx.FindFirstRecordByFilter(schedulingstore.BusinessEventsCollectionName, "corrects_event = {:event}", dbx.Params{"event": input.CorrectsEvent})
	if err == nil {
		return errConflict
	}
	if err != sql.ErrNoRows {
		return err
	}
	return nil
}

func applyTokenCorrection(tx core.App, input correctionRequest, now time.Time) (string, error) {
	token, err := tx.FindRecordById(schedulingstore.PackageTokensCollectionName, input.TokenID)
	if err != nil {
		return "", err
	}
	priorState := token.GetString(schedulingstore.TokenStateField)
	lessonID := token.GetString(schedulingstore.LessonField)
	if correctionRequiresLesson(input.Target) && lessonID == "" {
		return "", errInvalid
	}
	token.Set(schedulingstore.TokenStateField, string(input.Target))
	token.Set(schedulingstore.ChangedAtField, now.Format(time.RFC3339))
	if correctionClearsLesson(input.Target) {
		token.Set(schedulingstore.LessonField, "")
	}
	if err := tx.Save(token); err != nil {
		return "", err
	}
	if lessonID != "" && correctionClearsLesson(input.Target) {
		return priorState, clearLessonToken(tx, lessonID)
	}
	return priorState, nil
}

func correctionRequiresLesson(target commercial.TokenState) bool {
	return target == commercial.TokenReserved || target == commercial.TokenUsed
}

func correctionClearsLesson(target commercial.TokenState) bool {
	return target == commercial.TokenAvailable || target == commercial.TokenExpired || target == commercial.TokenInvalidated
}

func clearLessonToken(tx core.App, lessonID string) error {
	lesson, err := tx.FindRecordById(schedulingstore.LessonsCollectionName, lessonID)
	if err != nil {
		return err
	}
	lesson.Set(schedulingstore.PackageTokenField, "")
	return tx.Save(lesson)
}
