// Package commercialapi request and record codecs keep PocketBase fields behind stable DTOs.
// Dates remain teacher-local strings, while persisted instants remain UTC.
package commercialapi

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type purchaseRequest struct {
	PurchasedOn      string   `json:"purchased_on,omitempty"`
	ConvertLessonIDs []string `json:"convert_lesson_ids,omitempty"`
}

type closeRequest struct {
	Reason string       `json:"reason"`
	Refund *refundInput `json:"refund,omitempty"`
}

type refundInput struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Note        string `json:"note,omitempty"`
}

type correctionRequest struct {
	TokenID       string                `json:"token_id"`
	Target        commercial.TokenState `json:"target"`
	CorrectsEvent string                `json:"corrects_event"`
	Reason        string                `json:"reason"`
}

func decodeStrict(e *core.RequestEvent, destination any) error {
	if e.Request.Body == nil {
		return errInvalid
	}
	raw, err := io.ReadAll(e.Request.Body)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		return errInvalid
	}
	trimmed := bytes.TrimSpace(raw)
	if trimmed[0] != '{' || bytes.Equal(trimmed, []byte("null")) {
		return errInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errInvalid
	}
	return nil
}

func teacherDate(value string, now time.Time, timezone string) (time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" {
		return time.Time{}, errInvalid
	}
	if value == "" {
		year, month, day := now.In(location).Date()
		return time.Date(year, month, day, 12, 0, 0, 0, location), nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return time.Time{}, errInvalid
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 12, 0, 0, 0, location), nil
}

func policySnapshot(record *core.Record) (businesspolicy.PolicySnapshot, error) {
	var snapshot businesspolicy.PolicySnapshot
	encoded, err := json.Marshal(record.Get(schedulingstore.PolicySnapshotField))
	if err != nil || json.Unmarshal(encoded, &snapshot) != nil || snapshot.Validate() != nil {
		return businesspolicy.PolicySnapshot{}, errInvalid
	}
	return snapshot, nil
}

func packageFromRecords(app core.App, packageRecord *core.Record) (commercial.Package, error) {
	assignmentID := packageRecord.GetString(schedulingstore.AssignmentField)
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, assignmentID)
	if err != nil {
		return commercial.Package{}, errForbidden
	}
	teacher, err := app.FindRecordById("teachers", assignment.GetString("teacher"))
	if err != nil {
		return commercial.Package{}, errForbidden
	}
	purchased, err := time.Parse("2006-01-02", packageRecord.GetString(schedulingstore.PurchasedOnField))
	if err != nil {
		return commercial.Package{}, errInvalid
	}
	validThrough, err := time.Parse("2006-01-02", packageRecord.GetString(schedulingstore.ValidThroughField))
	if err != nil {
		return commercial.Package{}, errInvalid
	}
	snapshot, err := policySnapshot(packageRecord)
	if err != nil {
		return commercial.Package{}, err
	}
	tokenRows, err := app.FindAllRecords(schedulingstore.PackageTokensCollectionName, dbx.HashExp{schedulingstore.PackageField: packageRecord.Id})
	if err != nil {
		return commercial.Package{}, err
	}
	tokens := make([]commercial.Token, 0, len(tokenRows))
	for _, row := range tokenRows {
		lessonStart := time.Time{}
		if lessonID := row.GetString(schedulingstore.LessonField); lessonID != "" {
			lesson, lessonErr := app.FindRecordById(schedulingstore.LessonsCollectionName, lessonID)
			if lessonErr != nil {
				return commercial.Package{}, errInvalid
			}
			lessonStart = lesson.GetDateTime(schedulingstore.StartAtField).Time().UTC()
		}
		tokens = append(tokens, commercial.Token{ID: row.Id, Ordinal: row.GetInt(schedulingstore.OrdinalField), State: commercial.TokenState(row.GetString(schedulingstore.TokenStateField)), LessonID: row.GetString(schedulingstore.LessonField), LessonStart: lessonStart})
	}
	return commercial.Package{ID: packageRecord.Id, Assignment: commercial.Assignment{ID: assignment.Id, TeacherID: assignment.GetString("teacher"), LearnerID: assignment.GetString("learner")}, Status: commercial.PackageStatus(packageRecord.GetString(schedulingstore.PackageStatusField)), PurchasedOn: purchased, ValidThrough: validThrough, ClosedAt: packageRecord.GetDateTime(schedulingstore.ClosedAtField).Time().UTC(), Price: commercial.Money{Minor: int64(packageRecord.GetInt(schedulingstore.UnitPriceMinorField)), Currency: packageRecord.GetString(schedulingstore.CurrencyField)}, Policy: snapshot, TeacherZone: teacher.GetString("timezone"), Tokens: tokens}, nil
}

func packageValue(app core.App, row *core.Record) (packageDTO, error) {
	value, err := packageFromRecords(app, row)
	if err != nil {
		return packageDTO{}, err
	}
	return packageDTO{ID: row.Id, Assignment: row.GetString(schedulingstore.AssignmentField), Status: string(value.Status), PurchasedOn: row.GetString(schedulingstore.PurchasedOnField), ValidThrough: row.GetString(schedulingstore.ValidThroughField), PriceMinor: value.Price.Minor, Currency: value.Price.Currency, PolicyVersion: value.Policy.Version, Tokens: tokenValues(app, value.Tokens)}, nil
}

func tokenValues(app core.App, tokens []commercial.Token) []tokenDTO {
	items := make([]tokenDTO, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, tokenDTO{ID: token.ID, Ordinal: token.Ordinal, State: string(token.State), Lesson: emptyOrValue(app, token.LessonID)})
	}
	return items
}

func emptyOrValue(app core.App, id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
