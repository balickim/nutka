// Application assembly wires PocketBase migrations, auth hooks, seed commands, and bootstrap orchestration.
// Scheduling rules and resource handlers live in their dedicated packages.
package main

import (
	"log"
	"os"
	"time"

	"github.com/balickim/nutka/apps/backend/database"
	_ "github.com/balickim/nutka/apps/backend/database/migrations"
	"github.com/balickim/nutka/apps/backend/internal/businesspolicyapi"
	"github.com/balickim/nutka/apps/backend/internal/commercialapi"
	"github.com/balickim/nutka/apps/backend/internal/historyapi"
	"github.com/balickim/nutka/apps/backend/internal/ledgerapi"
	"github.com/balickim/nutka/apps/backend/internal/lessonnotesapi"
	"github.com/balickim/nutka/apps/backend/internal/materialsapi"
	"github.com/balickim/nutka/apps/backend/internal/paymentdetailsapi"
	"github.com/balickim/nutka/apps/backend/internal/practiceapi"
	"github.com/balickim/nutka/apps/backend/internal/regularcontractapi"
	"github.com/balickim/nutka/apps/backend/internal/repertoireapi"
	"github.com/balickim/nutka/apps/backend/internal/schedulingapi"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
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
	configureAppWithClock(app, time.Now)
}

func configureAppWithClock(app *pocketbase.PocketBase, clock schedulingapi.Clock) {
	schedulingstore.RegisterHooks(app)
	schedulingapi.RegisterRoutesWithClock(app, clock)
	schedulingapi.RegisterBookingRoutes(app, clock)
	schedulingapi.RegisterAvailabilityRoutesWithClock(app, clock)
	businesspolicyapi.RegisterRoutes(app)
	commercialapi.RegisterRoutesWithClock(app, commercialapi.Clock(clock))
	historyapi.RegisterRoutes(app)
	materialsapi.RegisterRoutes(app)
	lessonnotesapi.RegisterRoutes(app, lessonnotesapi.Clock(clock))
	paymentdetailsapi.RegisterRoutes(app)
	repertoireapi.RegisterRoutes(app, repertoireapi.Clock(clock))
	practiceapi.RegisterRoutes(app, practiceapi.Clock(clock))
	contractRepository := regularcontractapi.NewPocketBaseRepository(app, regularcontractapi.Clock(clock))
	regularcontractapi.RegisterRoutes(app, regularcontractapi.New(contractRepository, regularcontractapi.Clock(clock)))
	ledgerService := ledgerapi.NewPocketBaseService(app, func() time.Time { return clock() })
	ledgerapi.RegisterRoutesWithClock(app, ledgerService, ledgerapi.Clock(clock))
	registerDualPersonaAuth(app)
	registerSeedLearnerCommand(app)
	registerSeedTeacherCommand(app)
	registerSeedAssignmentCommand(app)
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
