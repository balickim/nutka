// Package migrations adds hidden bank transfer fields to teachers. Hidden fields stay out of auth responses and native routes.
package migrations

import (
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/paymentdetails"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(addTeacherPaymentDetails, func(core.App) error { return nil })
}

func addTeacherPaymentDetails(app core.App) error {
	teachers, err := app.FindCollectionByNameOrId(authconfig.TeachersCollectionName)
	if err != nil {
		return fmt.Errorf("find teachers collection: %w", err)
	}
	teachers.Fields.Add(
		&core.TextField{Name: paymentdetails.HolderField, Hidden: true, Max: paymentdetails.HolderMaxLength},
		&core.TextField{Name: paymentdetails.IBANField, Hidden: true, Max: 34},
		&core.TextField{Name: paymentdetails.NoteField, Hidden: true, Max: paymentdetails.NoteMaxLength},
	)
	if err := app.Save(teachers); err != nil {
		return fmt.Errorf("add teacher payment details: %w", err)
	}
	return nil
}
