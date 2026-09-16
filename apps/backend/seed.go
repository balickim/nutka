// Registers development-only commands that create or update verified local persona records.
package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

func registerSeedLearnerCommand(app *pocketbase.PocketBase) {
	registerSeedCommand(app, "seed-learner", "learner", authconfig.LearnersCollectionName)
}

func registerSeedTeacherCommand(app *pocketbase.PocketBase) {
	registerSeedCommand(app, "seed-teacher", "teacher", authconfig.TeachersCollectionName)
}

func registerSeedAssignmentCommand(app *pocketbase.PocketBase) {
	if !isDevelopment() {
		return
	}
	var teacher, learner string
	command := &cobra.Command{Use: "seed-assignment", Short: "Assign a local teacher to a learner", RunE: func(cmd *cobra.Command, _ []string) error {
		if teacher == "" || learner == "" {
			return errors.New("teacher and learner are required")
		}
		if err := seedAssignmentCommand(app, teacher, learner); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "seeded assignment %s -> %s\n", teacher, learner)
		return err
	}}
	command.Flags().StringVar(&teacher, "teacher", "", "teacher id or email")
	command.Flags().StringVar(&learner, "learner", "", "learner id or email")
	app.RootCmd.AddCommand(command)
}

func registerSeedCommand(app *pocketbase.PocketBase, commandName, persona, collectionName string) {
	if !isDevelopment() {
		return
	}

	var email string
	var password string
	var name string
	command := &cobra.Command{
		Use:   commandName,
		Short: "Create or update a verified local " + persona,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if email == "" || password == "" || name == "" {
				return errors.New("email, password, and name are required")
			}
			if err := seedPersona(app, collectionName, email, password, name); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "seeded %s %s\n", persona, email)
			return err
		},
	}
	command.Flags().StringVar(&email, "email", "", persona+" email")
	command.Flags().StringVar(&password, "password", "", persona+" password")
	command.Flags().StringVar(&name, "name", "", persona+" display name")
	app.RootCmd.AddCommand(command)
}

func seedLearner(app core.App, email, password, name string) error {
	return seedPersona(app, authconfig.LearnersCollectionName, email, password, name)
}

func seedTeacher(app core.App, email, password, name string) error {
	return seedPersona(app, authconfig.TeachersCollectionName, email, password, name)
}

func seedAssignmentCommand(app core.App, teacherRef, learnerRef string) error {
	teacher, err := findSeedAccount(app, authconfig.TeachersCollectionName, teacherRef)
	if err != nil {
		return err
	}
	learner, err := findSeedAccount(app, authconfig.LearnersCollectionName, learnerRef)
	if err != nil {
		return err
	}
	assignments, err := app.FindAllRecords(schedulingstore.TeacherLearnersCollectionName, dbx.HashExp{"teacher": teacher.Id, "learner": learner.Id})
	if err != nil {
		return fmt.Errorf("find assignment: %w", err)
	}
	if len(assignments) > 0 {
		return errors.New("assignment already exists")
	}
	collection, err := app.FindCollectionByNameOrId(schedulingstore.TeacherLearnersCollectionName)
	if err != nil {
		return fmt.Errorf("find assignments collection: %w", err)
	}
	row := core.NewRecord(collection)
	row.Set("teacher", teacher.Id)
	row.Set("learner", learner.Id)
	row.Set(schedulingstore.ActiveField, true)
	if err := app.Save(row); err != nil {
		return fmt.Errorf("save assignment: %w", err)
	}
	return nil
}

func findSeedAccount(app core.App, collection, reference string) (*core.Record, error) {
	row, err := app.FindRecordById(collection, reference)
	if err == nil {
		return row, nil
	}
	row, err = app.FindFirstRecordByData(collection, core.FieldNameEmail, reference)
	if err != nil {
		return nil, fmt.Errorf("find %s account: %w", collection, err)
	}
	return row, nil
}

func seedPersona(app core.App, collectionName, email, password, name string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return fmt.Errorf("find %s collection: %w", collectionName, err)
	}

	record, err := app.FindFirstRecordByData(collection, core.FieldNameEmail, email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find %s record: %w", collectionName, err)
		}
		record = core.NewRecord(collection)
		record.SetEmail(email)
	}
	record.Set("name", name)
	record.SetPassword(password)
	record.SetVerified(true)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("save %s: %w", collectionName, err)
	}
	return nil
}
