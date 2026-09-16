// Package schedulingstore binds scheduling record defaults and timezone validation.
// The application must call RegisterHooks on every boot after PocketBase initializes hooks.
package schedulingstore

import (
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/authconfig"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// RegisterHooks installs defaults and timezone validation for every application boot.
func RegisterHooks(app core.App) {
	registerCommercialHooks(app)
	RegisterTimezoneGuard(app)
	app.OnRecordCreate().Bind(&hook.Handler[*core.RecordEvent]{
		Id: "nutkaSchedulingRecordDefaults",
		Func: func(e *core.RecordEvent) error {
			applyRecordDefaults(e.Record)
			return e.Next()
		},
	})
	app.OnRecordValidate().Bind(&hook.Handler[*core.RecordEvent]{
		Id: "nutkaTeacherTimezoneValidation",
		Func: func(e *core.RecordEvent) error {
			if e.Record.Collection().Name == authconfig.TeachersCollectionName {
				value := e.Record.GetString(TeacherTimezoneField)
				if value == "" {
					e.Record.Set(TeacherTimezoneField, scheduling.DefaultTimezone)
					value = scheduling.DefaultTimezone
				}
				if _, err := time.LoadLocation(value); err != nil {
					return fmt.Errorf("timezone must be a valid IANA identifier: %q", value)
				}
			}
			return e.Next()
		},
	})
}

func applyRecordDefaults(record *core.Record) {
	switch record.Collection().Name {
	case authconfig.TeachersCollectionName:
		defaultTeacherTimezone(record)
	case TeacherLearnersCollectionName:
		defaultAssignment(record)
	case LessonsCollectionName:
		defaultLessonStatus(record)
	}
}

func defaultTeacherTimezone(record *core.Record) {
	if record.GetString(TeacherTimezoneField) == "" {
		record.Set(TeacherTimezoneField, scheduling.DefaultTimezone)
	}
}

func defaultAssignment(record *core.Record) {
	if !record.IsNew() {
		return
	}
	if record.Collection().Fields.GetByName(DefaultDurationMinutesField) != nil && record.GetInt(DefaultDurationMinutesField) == 0 {
		record.Set(DefaultDurationMinutesField, int(scheduling.DefaultLessonDuration/time.Minute))
	}
	if !record.GetBool(ActiveField) {
		record.Set(ActiveField, true)
	}
}

func defaultLessonStatus(record *core.Record) {
	if record.IsNew() && record.GetString(StatusField) == "" {
		record.Set(StatusField, "scheduled")
	}
}
