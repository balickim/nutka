// This file returns teacher-owned availability rules and exceptions without exposing mutable PocketBase records.
package schedulingapi

import (
	"net/http"

	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/pocketbase/core"
)

func teacherRules(e *core.RequestEvent) error {
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	rows, err := recordsByField(e.App, schedulingstore.AvailabilityRulesCollectionName, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	items := make([]ruleDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, ruleValue(row))
	}
	return e.JSON(http.StatusOK, map[string]any{"availability_rules": items})
}

func teacherExceptions(e *core.RequestEvent) error {
	teacher, err := caller(e, "teacher")
	if err != nil {
		return handleError(e, err)
	}
	rows, err := recordsByField(e.App, schedulingstore.AvailabilityExceptionsCollectionName, "teacher", teacher.Id)
	if err != nil {
		return handleError(e, err)
	}
	items := make([]exceptionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, exceptionValue(row))
	}
	return e.JSON(http.StatusOK, map[string]any{"availability_exceptions": items})
}
