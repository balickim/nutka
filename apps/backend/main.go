package main

import (
	"log"
	"os"

	"github.com/balickim/nutka/apps/backend/database"
	_ "github.com/balickim/nutka/apps/backend/database/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := newApp()

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func newApp() *pocketbase.PocketBase {
	// Keep SQL query logging disabled even for `go run` so credential-derived
	// values (including password hashes) never reach the terminal.
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDev: false})
	database.RegisterMigrate(app)
	configureApp(app)
	return app
}

func configureApp(app *pocketbase.PocketBase) {
	registerLearnerAuth(app)
	registerSeedLearnerCommand(app)
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if err := e.App.RunAppMigrations(); err != nil {
			return err
		}
		return e.App.ReloadCachedCollections()
	})
}

func isDevelopment() bool {
	return os.Getenv("NUTKA_ENV") == "development"
}
