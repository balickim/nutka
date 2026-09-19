// Package paymentdetailsapi lets the signed-in teacher read and replace the teacher's own bank transfer details.
// Validation lives in paymentdetails. Persona checks live in personaroute.
package paymentdetailsapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/paymentdetails"
	"github.com/balickim/nutka/apps/backend/internal/personaroute"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const path = "/api/teachers/payment-details"

// RegisterRoutes binds the teacher payment details routes.
func RegisterRoutes(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET(path, readDetails)
		e.Router.PUT(path, replaceDetails)
		return e.Next()
	})
}

func readDetails(e *core.RequestEvent) error {
	teacher, err := personaroute.Caller(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	record, err := e.App.FindRecordById(authconfig.TeachersCollectionName, teacher.Id)
	if err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, stored(record))
}

func replaceDetails(e *core.RequestEvent) error {
	if err := personaroute.RequireIntent(e); err != nil {
		return respondError(e, err)
	}
	teacher, err := personaroute.Caller(e, personaroute.Teacher)
	if err != nil {
		return respondError(e, err)
	}
	var input paymentdetails.Details
	if err := json.NewDecoder(http.MaxBytesReader(e.Response, e.Request.Body, 4096)).Decode(&input); err != nil {
		return respondError(e, paymentdetails.ErrInvalid)
	}
	details := paymentdetails.Normalize(input)
	if err := paymentdetails.Validate(details); err != nil {
		return respondError(e, err)
	}
	record, err := e.App.FindRecordById(authconfig.TeachersCollectionName, teacher.Id)
	if err != nil {
		return respondError(e, err)
	}
	record.Set(paymentdetails.HolderField, details.AccountHolder)
	record.Set(paymentdetails.IBANField, details.IBAN)
	record.Set(paymentdetails.NoteField, details.Note)
	if err := e.App.Save(record); err != nil {
		return respondError(e, err)
	}
	return e.JSON(http.StatusOK, details)
}

func stored(record *core.Record) paymentdetails.Details {
	return paymentdetails.Details{AccountHolder: record.GetString(paymentdetails.HolderField), IBAN: record.GetString(paymentdetails.IBANField), Note: record.GetString(paymentdetails.NoteField)}
}

func respondError(e *core.RequestEvent, err error) error {
	if handled, writeErr := personaroute.WriteError(e, err); handled {
		return writeErr
	}
	if errors.Is(err, paymentdetails.ErrInvalid) {
		return e.JSON(http.StatusBadRequest, map[string]string{"code": "invalid_payment_details", "message": "The payment details are invalid."})
	}
	return e.JSON(http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": "The payment details request failed."})
}
