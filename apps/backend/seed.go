package main

import (
	"errors"
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

func registerSeedLearnerCommand(app *pocketbase.PocketBase) {
	if !isDevelopment() {
		return
	}

	var email string
	var password string
	var name string
	command := &cobra.Command{
		Use:   "seed-learner",
		Short: "Create or update a verified local learner",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if email == "" || password == "" || name == "" {
				return errors.New("email, password, and name are required")
			}
			if err := seedLearner(app, email, password, name); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "seeded learner %s\n", email)
			return err
		},
	}
	command.Flags().StringVar(&email, "email", "", "learner email")
	command.Flags().StringVar(&password, "password", "", "learner password")
	command.Flags().StringVar(&name, "name", "", "learner display name")
	app.RootCmd.AddCommand(command)
}

func seedLearner(app core.App, email, password, name string) error {
	collection, err := app.FindCollectionByNameOrId(authconfig.LearnersCollectionName)
	if err != nil {
		return fmt.Errorf("find learners collection: %w", err)
	}

	record, err := app.FindFirstRecordByData(collection, core.FieldNameEmail, email)
	if err != nil {
		record = core.NewRecord(collection)
		record.SetEmail(email)
	}
	record.Set(authconfig.LearnerNameField, name)
	record.SetPassword(password)
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("save learner: %w", err)
	}
	return nil
}
