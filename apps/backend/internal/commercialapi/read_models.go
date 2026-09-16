// Package commercialapi builds bounded package and assignment commercial summaries.
// Read models contain only records owned by the authenticated persona.
package commercialapi

import (
	"net/http"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/regularcontractapi"
	"github.com/balickim/nutka/apps/backend/internal/schedulingstore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type packageDTO struct {
	ID            string     `json:"id"`
	Assignment    string     `json:"assignment"`
	Status        string     `json:"status"`
	PurchasedOn   string     `json:"purchased_on"`
	ValidThrough  string     `json:"valid_through"`
	PriceMinor    int64      `json:"price_minor"`
	Currency      string     `json:"currency"`
	PolicyVersion string     `json:"policy_version"`
	Tokens        []tokenDTO `json:"tokens"`
}

type tokenDTO struct {
	ID      string  `json:"id"`
	Ordinal int     `json:"ordinal"`
	State   string  `json:"state"`
	Lesson  *string `json:"lesson,omitempty"`
}

type summaryDTO struct {
	Assignment string              `json:"assignment"`
	ActivePlan string              `json:"active_plan"`
	Package    *packageSummaryDTO  `json:"package"`
	Contract   *contractSummaryDTO `json:"contract"`
	Payments   paymentSummaryDTO   `json:"payments"`
}

type contractSummaryDTO struct {
	ID                          string `json:"id"`
	Status                      string `json:"status"`
	StartOn                     string `json:"start_on"`
	EndOn                       string `json:"end_on"`
	RemainingMonthlyReschedules int    `json:"remaining_monthly_reschedules"`
	RemainingFreeCancellations  int    `json:"remaining_free_cancellations"`
	PriceMinor                  int64  `json:"price_minor"`
	Currency                    string `json:"currency"`
}

type packageSummaryDTO struct {
	ID           string          `json:"id"`
	ValidThrough string          `json:"valid_through"`
	TokenBalance tokenBalanceDTO `json:"token_balance"`
}

type tokenBalanceDTO struct {
	Available   int `json:"available"`
	Reserved    int `json:"reserved"`
	Used        int `json:"used"`
	Expired     int `json:"expired"`
	Invalidated int `json:"invalidated"`
}

type paymentSummaryDTO struct {
	Pending             int    `json:"pending"`
	IntentionallyUnpaid int    `json:"intentionally_unpaid"`
	Overdue             int    `json:"overdue"`
	CreditMinor         int64  `json:"credit_minor"`
	Currency            string `json:"currency"`
}

func listPackages(e *core.RequestEvent, role string) error {
	account, err := authenticatedCaller(e, role)
	if err != nil {
		return commercialError(e, err)
	}
	assignment, err := ownedAssignment(e.App, e.Request.PathValue("id"), role, account.Id)
	if err != nil {
		return commercialError(e, err)
	}
	rows, err := e.App.FindAllRecords(schedulingstore.LessonPackagesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignment.Id})
	if err != nil {
		return commercialError(e, err)
	}
	items := make([]packageDTO, 0, len(rows))
	for _, row := range rows {
		item, valueErr := packageValue(e.App, row)
		if valueErr != nil {
			return commercialError(e, valueErr)
		}
		items = append(items, item)
	}
	return e.JSON(http.StatusOK, map[string]any{"packages": items})
}

func commercialSummary(e *core.RequestEvent, role string, clock Clock) error {
	account, err := authenticatedCaller(e, role)
	if err != nil {
		return commercialError(e, err)
	}
	assignment, err := ownedAssignment(e.App, e.Request.PathValue("id"), role, account.Id)
	if err != nil {
		return commercialError(e, err)
	}
	value, err := buildSummary(e.App, assignment, clock().UTC())
	if err != nil {
		return commercialError(e, err)
	}
	return e.JSON(http.StatusOK, value)
}

// SummaryForAssignment returns a safe commercial summary after assignment ownership is already established.
func SummaryForAssignment(app core.App, assignmentID string, now time.Time) (any, error) {
	assignment, err := app.FindRecordById(schedulingstore.TeacherLearnersCollectionName, assignmentID)
	if err != nil {
		return nil, errForbidden
	}
	return buildSummary(app, assignment, now.UTC())
}

// PaymentSummaryForAssignment returns the role-safe payment projection embedded in learner calendars.
func PaymentSummaryForAssignment(app core.App, assignmentID string, now time.Time) (any, error) {
	return paymentSummary(app, assignmentID, now.UTC())
}

func buildSummary(app core.App, assignment *core.Record, now time.Time) (summaryDTO, error) {
	selected, err := selectedPackageSummary(app, assignment, now)
	if err != nil {
		return summaryDTO{}, err
	}
	result := summaryDTO{Assignment: assignment.Id, Payments: paymentSummaryDTO{Currency: "PLN"}}
	if selected != nil {
		result.ActivePlan = "package"
		result.Package = selected
	}
	contract, err := activeContractSummary(app, assignment, now)
	if err != nil {
		return summaryDTO{}, err
	}
	if contract != nil {
		result.ActivePlan = "regular_contract"
		result.Contract = contract
	}
	if result.ActivePlan == "" && assignment.GetBool(schedulingstore.ActiveField) {
		result.ActivePlan = "ad_hoc"
	}
	result.Payments, err = paymentSummary(app, assignment.Id, now)
	if err != nil {
		return summaryDTO{}, err
	}
	return result, nil
}

func selectedPackageSummary(app core.App, assignment *core.Record, now time.Time) (*packageSummaryDTO, error) {
	rows, err := app.FindAllRecords(schedulingstore.LessonPackagesCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignment.Id})
	if err != nil {
		return nil, err
	}
	var selected *packageSummaryDTO
	var selectedDate string
	for _, row := range rows {
		value, err := packageFromRecords(app, row)
		if err != nil {
			return nil, err
		}
		balance := tokenBalance(value.Tokens)
		if selectablePackage(value, balance, now) && earlierPackage(row, selected, selectedDate) {
			selected = &packageSummaryDTO{ID: row.Id, ValidThrough: row.GetString(schedulingstore.ValidThroughField), TokenBalance: balance}
			selectedDate = row.GetString(schedulingstore.PurchasedOnField)
		}
	}
	return selected, nil
}

func selectablePackage(value commercial.Package, balance tokenBalanceDTO, now time.Time) bool {
	return value.Status == "open" && (balance.Available > 0 || balance.Reserved > 0) && (value.ValidOn(now, now) || balance.Reserved > 0)
}

func earlierPackage(row *core.Record, selected *packageSummaryDTO, selectedDate string) bool {
	date := row.GetString(schedulingstore.PurchasedOnField)
	return selected == nil || date < selectedDate || date == selectedDate && row.Id < selected.ID
}

func activeContractSummary(app core.App, assignment *core.Record, now time.Time) (*contractSummaryDTO, error) {
	rows, err := app.FindAllRecords(schedulingstore.RegularContractsCollectionName, dbx.HashExp{schedulingstore.AssignmentField: assignment.Id})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		active, err := contractActive(app, assignment, row, now)
		if err != nil {
			return nil, err
		}
		if active {
			return contractSummaryValue(app, row, now)
		}
	}
	return nil, nil
}

func contractSummaryValue(app core.App, row *core.Record, now time.Time) (*contractSummaryDTO, error) {
	value, err := regularcontractapi.LoadContract(app, row.Id)
	if err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(value.TeacherTimezone)
	if err != nil {
		return nil, err
	}
	allowances := value.Allowances(now.In(location).Format("2006-01"))
	return &contractSummaryDTO{ID: row.Id, Status: row.GetString(schedulingstore.StatusField), StartOn: value.StartOn.Format("2006-01-02"), EndOn: value.EffectiveEndOn.Format("2006-01-02"), RemainingMonthlyReschedules: allowances.MonthlyReschedulesRemaining, RemainingFreeCancellations: allowances.FreeCancellationsRemaining, PriceMinor: value.PriceMinor, Currency: value.Currency}, nil
}

func tokenBalance(tokens []commercial.Token) tokenBalanceDTO {
	var value tokenBalanceDTO
	for _, token := range tokens {
		switch token.State {
		case commercial.TokenAvailable:
			value.Available++
		case commercial.TokenReserved:
			value.Reserved++
		case commercial.TokenUsed:
			value.Used++
		case commercial.TokenExpired:
			value.Expired++
		case commercial.TokenInvalidated:
			value.Invalidated++
		}
	}
	return value
}
