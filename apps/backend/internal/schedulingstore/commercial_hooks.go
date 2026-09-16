// Package schedulingstore enforces commercial relation integrity at the PocketBase record boundary.
// Domain commands can bypass these defaults only by supplying complete, assignment-consistent records.
package schedulingstore

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func registerCommercialHooks(app core.App) {
	app.OnRecordValidate().Bind(&hook.Handler[*core.RecordEvent]{
		Id: "nutkaCommercialRecordValidation",
		Func: func(e *core.RecordEvent) error {
			if err := validateCommercialRecord(e.App, e.Record); err != nil {
				return err
			}
			return e.Next()
		},
	})
	app.OnRecordUpdate(BusinessEventsCollectionName).Bind(&hook.Handler[*core.RecordEvent]{
		Id: "nutkaBusinessEventsAppendOnlyUpdate",
		Func: func(e *core.RecordEvent) error {
			return fmt.Errorf("business events are append-only")
		},
	})
	app.OnRecordDelete(BusinessEventsCollectionName).Bind(&hook.Handler[*core.RecordEvent]{
		Id: "nutkaBusinessEventsAppendOnlyDelete",
		Func: func(e *core.RecordEvent) error {
			return fmt.Errorf("business events are append-only")
		},
	})
}
