// Package commercialapi persists package purchase and lifecycle decisions as one PocketBase transaction.
// Every payment and business event is written through the caller-owned transaction.
package commercialapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/history"
	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func purchasePackage(e *core.RequestEvent, clock Clock) error {
	if err := requireIntent(e); err != nil {
		return commercialError(e, err)
	}
	teacher, err := authenticatedCaller(e, "teacher")
	if err != nil {
		return commercialError(e, err)
	}
	var input purchaseRequest
	if err := decodeStrict(e, &input); err != nil {
		return commercialError(e, err)
	}
	now := clock().UTC()
	var result *core.Record
	packageMutationMu.Lock()
	defer packageMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		var purchaseErr error
		result, purchaseErr = purchasePackageTransaction(tx, e.Request.PathValue("id"), teacher.Id, input, now)
		return purchaseErr
	})
	if err != nil {
		return commercialError(e, mapDomainError(err))
	}
	value, valueErr := packageValue(e.App, result)
	if valueErr != nil {
		return commercialError(e, valueErr)
	}
	return e.JSON(http.StatusCreated, value)
}

func closePackage(e *core.RequestEvent, clock Clock) error {
	if err := requireIntent(e); err != nil {
		return commercialError(e, err)
	}
	teacher, err := authenticatedCaller(e, "teacher")
	if err != nil {
		return commercialError(e, err)
	}
	var input closeRequest
	if err := decodeStrict(e, &input); err != nil {
		return commercialError(e, err)
	}
	now := clock().UTC()
	var result *core.Record
	packageMutationMu.Lock()
	defer packageMutationMu.Unlock()
	err = e.App.RunInTransaction(func(tx core.App) error {
		var closeErr error
		result, closeErr = closePackageTransaction(tx, e.Request.PathValue("id"), teacher.Id, input, now)
		return closeErr
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

func purchasePackageTransaction(tx core.App, assignmentID, teacherID string, input purchaseRequest, now time.Time) (*core.Record, error) {
	assignment, err := ownedAssignment(tx, assignmentID, "teacher", teacherID)
	if err != nil {
		return nil, err
	}
	teacher, err := tx.FindRecordById("teachers", teacherID)
	if err != nil {
		return nil, errForbidden
	}
	zone := teacher.GetString("timezone")
	purchaseDate, err := teacherDate(input.PurchasedOn, now, zone)
	if err != nil {
		return nil, err
	}
	state, err := purchaseOverlapState(tx, assignment, now)
	if err != nil {
		return nil, err
	}
	request := commercial.PurchaseRequest{ID: newRecordID(), Assignment: commercial.Assignment{ID: assignment.Id, TeacherID: teacherID, LearnerID: assignment.GetString("learner")}, PurchaseDate: purchaseDate, Now: now, TeacherZone: zone}
	decision, err := commercial.DecidePackagePurchase(request, state, input.ConvertLessonIDs)
	if err != nil {
		return nil, err
	}
	return persistPackagePurchase(tx, assignment, teacherID, decision, now)
}

func persistPackagePurchase(tx core.App, assignment *core.Record, teacherID string, decision commercial.ConversionDecision, now time.Time) (*core.Record, error) {
	packageRecord, err := savePackage(tx, *decision.Package)
	if err != nil {
		return nil, fmt.Errorf("save package: %w", err)
	}
	tokenIDs, err := saveTokens(tx, *decision.Package, now)
	if err != nil {
		return nil, fmt.Errorf("save tokens: %w", err)
	}
	for index := range decision.Package.Tokens {
		decision.Package.Tokens[index].ID = tokenIDs[decision.Package.Tokens[index].ID]
	}
	for _, lesson := range decision.ConvertedLessons {
		if err := convertLesson(tx, lesson.ID, packageRecord.Id, decision.Package); err != nil {
			return nil, err
		}
	}
	if err := persistPackagePurchaseAudit(tx, assignment, teacherID, packageRecord, decision.Package, now); err != nil {
		return nil, err
	}
	return packageRecord, nil
}

func persistPackagePurchaseAudit(tx core.App, assignment *core.Record, teacherID string, row *core.Record, value *commercial.Package, now time.Time) error {
	payment, err := ledger.RecordPackagePurchase(ledger.PackagePurchaseCommand{Package: value, Actor: ledger.Actor{Role: ledger.TeacherActor, ID: teacherID}, At: now})
	if err != nil {
		return err
	}
	if err := savePayment(tx, payment.Payment, row.Id); err != nil {
		return fmt.Errorf("save payment: %w", err)
	}
	event, err := history.NewPackagePurchased(history.EventInput{AggregateType: "package", AggregateID: row.Id, AssignmentID: assignment.Id, Actor: history.Actor{Role: history.TeacherActor, ID: teacherID}, EventAt: now, NewState: map[string]any{"status": "open", "purchased_on": row.GetString(schedulingstore.PurchasedOnField), "price_minor": value.Price.Minor}})
	if err != nil {
		return fmt.Errorf("purchase event construct: %w", err)
	}
	if err := (history.Storage{}).Append(tx, event); err != nil {
		return fmt.Errorf("purchase event: %w", err)
	}
	return appendReservedTokenEvents(tx, assignment.Id, teacherID, value.Tokens, now)
}

func appendReservedTokenEvents(tx core.App, assignmentID, teacherID string, tokens []commercial.Token, now time.Time) error {
	for _, token := range tokens {
		if token.LessonID == "" {
			continue
		}
		event, err := history.NewPackageTokenReserved(history.EventInput{AggregateType: "package_token", AggregateID: token.ID, AssignmentID: assignmentID, Actor: history.Actor{Role: history.TeacherActor, ID: teacherID}, EventAt: now, NewState: map[string]any{"state": string(token.State), "lesson": token.LessonID}})
		if err != nil {
			return fmt.Errorf("token event construct: %w", err)
		}
		if err := (history.Storage{}).Append(tx, event); err != nil {
			return err
		}
	}
	return nil
}

func closePackageTransaction(tx core.App, packageID, teacherID string, input closeRequest, now time.Time) (*core.Record, error) {
	row, err := ownedPackage(tx, packageID, teacherID)
	if err != nil {
		return nil, err
	}
	value, err := packageFromRecords(tx, row)
	if err != nil {
		return nil, err
	}
	invalidatedTokenIDs := availableTokenIDs(value)
	decision, err := value.Close(now, input.Reason, refundInfo(input.Refund))
	if err != nil {
		return nil, err
	}
	row.Set(schedulingstore.PackageStatusField, string(value.Status))
	row.Set(schedulingstore.ClosedAtField, now.Format(time.RFC3339))
	if err := tx.Save(row); err != nil {
		return nil, err
	}
	if err := persistPackageClosure(tx, row, value, invalidatedTokenIDs, decision, input, teacherID, now); err != nil {
		return nil, err
	}
	return row, nil
}

func refundInfo(input *refundInput) *commercial.RefundInfo {
	if input == nil {
		return nil
	}
	return &commercial.RefundInfo{Amount: commercial.Money{Minor: input.AmountMinor, Currency: input.Currency}, Note: input.Note}
}

func persistPackageClosure(tx core.App, row *core.Record, value commercial.Package, tokenIDs []string, decision commercial.CloseDecision, input closeRequest, teacherID string, now time.Time) error {
	if err := invalidateTokens(tx, value, now); err != nil {
		return err
	}
	if input.Refund != nil {
		if err := savePackageRefund(tx, row, *input.Refund, input.Reason, teacherID, now); err != nil {
			return err
		}
	}
	if err := appendInvalidatedTokenEvents(tx, row, tokenIDs, teacherID, now); err != nil {
		return err
	}
	event, err := history.NewPackageClosed(history.EventInput{AggregateType: "package", AggregateID: row.Id, AssignmentID: row.GetString(schedulingstore.AssignmentField), Actor: history.Actor{Role: history.TeacherActor, ID: teacherID}, EventAt: now, Reason: input.Reason, NewState: map[string]any{"status": "closed", "invalidated": decision.Invalidated, "refund": input.Refund}})
	if err != nil {
		return err
	}
	return (history.Storage{}).Append(tx, event)
}

func appendInvalidatedTokenEvents(tx core.App, row *core.Record, tokenIDs []string, teacherID string, now time.Time) error {
	for _, tokenID := range tokenIDs {
		event, err := history.NewPackageTokenInvalidated(history.EventInput{AggregateType: "package_token", AggregateID: tokenID, AssignmentID: row.GetString(schedulingstore.AssignmentField), Actor: history.Actor{Role: history.TeacherActor, ID: teacherID}, EventAt: now, NewState: map[string]any{"state": string(commercial.TokenInvalidated)}})
		if err != nil {
			return err
		}
		if err := (history.Storage{}).Append(tx, event); err != nil {
			return err
		}
	}
	return nil
}

func availableTokenIDs(value commercial.Package) []string {
	result := make([]string, 0)
	for _, token := range value.Tokens {
		if token.State == commercial.TokenAvailable {
			result = append(result, token.ID)
		}
	}
	return result
}

func mapDomainError(err error) error {
	if packageConflict(err) {
		return errConflict
	}
	if packageInvalid(err) {
		return errInvalid
	}
	if errors.Is(err, commercial.ErrActiveContractOverlap) || errors.Is(err, commercial.ErrAssignmentMismatch) {
		return errConflict
	}
	return err
}

func packageConflict(err error) bool {
	return errors.Is(err, commercial.ErrPackageRenewal) || errors.Is(err, commercial.ErrAdHocOverlap) || errors.Is(err, commercial.ErrFuturePackageLesson) || errors.Is(err, commercial.ErrPackageUnavailable) || errors.Is(err, commercial.ErrConversionNotEligible) || errors.Is(err, commercial.ErrConversionLimit)
}

func packageInvalid(err error) bool {
	return errors.Is(err, commercial.ErrCloseReasonRequired) || errors.Is(err, commercial.ErrInvalidRefund) || errors.Is(err, commercial.ErrCorrectionReason) || errors.Is(err, commercial.ErrCorrectionEvent)
}
