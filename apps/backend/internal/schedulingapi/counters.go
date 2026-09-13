// This file exposes cancellation totals attributed to the authenticated dashboard persona.
package schedulingapi

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func teacherCancellationCounters(e *core.RequestEvent) error {
	return cancellationCounterResponse(e, "teacher")
}

func learnerCancellationCounters(e *core.RequestEvent) error {
	return cancellationCounterResponse(e, "learner")
}

func cancellationCounterResponse(e *core.RequestEvent, role string) error {
	account, err := caller(e, role)
	if err != nil {
		return handleError(e, err)
	}
	lessons, err := lessonsFor(e.App, role, account.Id)
	if err != nil {
		return handleError(e, err)
	}
	counters := cancellationCounters(lessons, account.Id, role)
	return e.JSON(http.StatusOK, counters)
}
