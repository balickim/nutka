package migrations

import (
	"database/sql"
	"errors"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId(authconfig.LearnersCollectionName); err == nil {
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		collection := core.NewAuthCollection(authconfig.LearnersCollectionName)
		collection.ListRule = nil
		collection.ViewRule = nil
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil
		// Verification is checked in the auth hook so unverified and unknown
		// identities share the same generic credential error.
		collection.AuthRule = types.Pointer("")
		collection.AuthToken.Duration = 12 * 60 * 60
		collection.PasswordAuth.Enabled = true
		collection.PasswordAuth.IdentityFields = []string{core.FieldNameEmail}
		collection.MFA.Enabled = false
		collection.OTP.Enabled = false
		collection.Fields.Add(
			&core.TextField{
				Name:        authconfig.LearnerNameField,
				Required:    true,
				Max:         160,
				Presentable: true,
			},
			&core.DateField{
				Name:   "last_login_at",
				Hidden: true,
			},
			&core.JSONField{
				Name:    "login_metadata",
				Hidden:  true,
				MaxSize: 16 * 1024,
			},
		)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(authconfig.LearnersCollectionName)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
