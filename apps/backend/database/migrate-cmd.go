package database

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

// RegisterMigrate makes the checked-in Go migrations available to PocketBase.
// Auto-migration snapshots remain a go-run development convenience only.
func RegisterMigrate(app *pocketbase.PocketBase) {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Dir:         filepath.Join(workingDir, "database", "migrations"),
		Automigrate: isGoRun,
	})
}
