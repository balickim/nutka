// This file computes contract-month billable forecasts and idempotent charge projections.
// It evaluates teacher-local business dates without persistence or external side effects.
package ledger

import "time"

type OccurrenceDisposition string

const (
	BillableOccurrence    OccurrenceDisposition = "billable"
	PlannedOmission       OccurrenceDisposition = "planned_omission"
	FreeCancellation      OccurrenceDisposition = "free_cancellation"
	TeacherCancellation   OccurrenceDisposition = "teacher_cancellation"
	LateCancellation      OccurrenceDisposition = "late_cancellation"
	ExhaustedCancellation OccurrenceDisposition = "exhausted_cancellation"
	NoShow                OccurrenceDisposition = "no_show"
	ContractTermination   OccurrenceDisposition = "contract_termination"
)

// BillingOutcome aliases the contract vocabulary used by occurrence materializers.
type BillingOutcome = OccurrenceDisposition

const (
	BillableOrdinary   BillingOutcome = BillableOccurrence
	BillableLateCancel BillingOutcome = LateCancellation
	BillableExhausted  BillingOutcome = ExhaustedCancellation
	FreeLearnerCancel  BillingOutcome = FreeCancellation
	TeacherCancel      BillingOutcome = TeacherCancellation
)

// OccurrenceSnapshot freezes the value and commercial consequence used by a month.
type OccurrenceSnapshot struct {
	ID             string
	ContractID     string
	AssignmentID   string
	Month          string
	UnitPriceMinor int64
	Currency       string
	Disposition    OccurrenceDisposition
	BillingOutcome BillingOutcome
	ScheduleState  string
	Outcome        string
}

func (occurrence OccurrenceSnapshot) IsBillable() bool {
	disposition := occurrence.Disposition
	if disposition == "" {
		disposition = occurrence.BillingOutcome
	}
	if occurrence.ScheduleState == "omitted" {
		return false
	}
	switch disposition {
	case PlannedOmission, FreeCancellation, TeacherCancellation, ContractTermination:
		return false
	default:
		return true
	}
}

func validDisposition(value OccurrenceDisposition) bool {
	switch value {
	case BillableOccurrence, PlannedOmission, FreeCancellation, TeacherCancellation, LateCancellation, ExhaustedCancellation, NoShow, ContractTermination:
		return true
	default:
		return false
	}
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

type ContractMonthForecast struct {
	ContractID          string
	AssignmentID        string
	Month               string
	BillableCount       int
	ForecastAmountMinor int64
	Currency            string
	OccurrenceIDs       []string
}

// ForecastContractMonth sums frozen billable occurrence values and ignores omissions and free cancellations.
func ForecastContractMonth(contractID, assignmentID, month string, occurrences []OccurrenceSnapshot) (ContractMonthForecast, error) {
	parsedMonth, err := time.Parse("2006-01", month)
	if err != nil || parsedMonth.Format("2006-01") != month {
		return ContractMonthForecast{}, ErrInvalidDate
	}
	forecast := ContractMonthForecast{ContractID: contractID, AssignmentID: assignmentID, Month: month, OccurrenceIDs: []string{}}
	for _, occurrence := range occurrences {
		if err := forecast.addOccurrence(occurrence); err != nil {
			return ContractMonthForecast{}, err
		}
	}
	return forecast, nil
}

func (forecast *ContractMonthForecast) addOccurrence(occurrence OccurrenceSnapshot) error {
	if occurrence.ContractID != forecast.ContractID || occurrence.AssignmentID != forecast.AssignmentID || occurrence.Month != forecast.Month {
		return nil
	}
	if err := validateOccurrenceSnapshot(occurrence); err != nil {
		return err
	}
	if forecast.Currency == "" {
		forecast.Currency = occurrence.Currency
	}
	if occurrence.Currency != forecast.Currency {
		return ErrCurrencyMismatch
	}
	if occurrence.IsBillable() {
		forecast.BillableCount++
		forecast.ForecastAmountMinor += occurrence.UnitPriceMinor
		forecast.OccurrenceIDs = append(forecast.OccurrenceIDs, occurrence.ID)
	}
	return nil
}

func validateOccurrenceSnapshot(occurrence OccurrenceSnapshot) error {
	if occurrence.ID == "" || occurrence.UnitPriceMinor <= 0 || !validCurrency(occurrence.Currency) {
		return ErrInvalidOccurrence
	}
	disposition := occurrence.Disposition
	if disposition == "" {
		disposition = occurrence.BillingOutcome
	}
	if !validDisposition(disposition) {
		return ErrInvalidDisposition
	}
	return nil
}

type ContractMonth struct {
	ID                  string
	ContractID          string
	AssignmentID        string
	Month               string
	ForecastAmountMinor int64
	ChargeID            string
	GeneratedAt         time.Time
}

type MonthReconciliationInput struct {
	Existing       *ContractMonth
	ContractID     string
	AssignmentID   string
	Month          string
	ActivationDate string
	Now            time.Time
	Location       *time.Location
	DueDay         int
	Currency       string
	Occurrences    []OccurrenceSnapshot
	Charge         *Charge
}

type MonthReconciliationDecision struct {
	Projection ContractMonth
	Forecast   ContractMonthForecast
	Charge     *Charge
	Created    bool
	Payable    bool
}

// ReconcileContractMonth is idempotent: an existing projection keeps its charge identity.
// A month becomes payable on its first local day, or immediately for a mid-month activation.
func ReconcileContractMonth(input MonthReconciliationInput) (MonthReconciliationDecision, error) {
	if input.Location == nil || input.Now.IsZero() {
		return MonthReconciliationDecision{}, ErrInvalidDate
	}
	forecast, err := ForecastContractMonth(input.ContractID, input.AssignmentID, input.Month, input.Occurrences)
	if err != nil {
		return MonthReconciliationDecision{}, err
	}
	if forecast.Currency == "" {
		forecast.Currency = input.Currency
	}
	monthDate, err := parseDate(input.Month+"-01", input.Location)
	if err != nil {
		return MonthReconciliationDecision{}, err
	}
	activation, err := parseDate(input.ActivationDate, input.Location)
	if err != nil {
		return MonthReconciliationDecision{}, err
	}
	payable := monthPayable(monthDate, activation, input.Now, input.Location)
	projection := monthProjection(input, forecast)
	decision := MonthReconciliationDecision{Projection: projection, Forecast: forecast, Payable: payable}
	if !payable {
		return decision, nil
	}
	if preserveExistingCharge(&decision, input) {
		return decision, nil
	}
	charge, err := newMonthCharge(input, projection, forecast, monthDate)
	if err != nil {
		return MonthReconciliationDecision{}, err
	}
	decision.Charge = charge
	decision.Created = true
	decision.Projection.ChargeID = charge.ID
	return decision, nil
}

func monthPayable(monthDate, activation, now time.Time, location *time.Location) bool {
	localNow := now.In(location)
	currentMonth := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, location)
	return !monthDate.After(currentMonth) && activation.Before(monthDate.AddDate(0, 1, 0))
}

func monthProjection(input MonthReconciliationInput, forecast ContractMonthForecast) ContractMonth {
	if input.Existing != nil {
		projection := *input.Existing
		projection.ForecastAmountMinor = forecast.ForecastAmountMinor
		return projection
	}
	return ContractMonth{ID: input.ContractID + ":" + input.Month, ContractID: input.ContractID, AssignmentID: input.AssignmentID, Month: input.Month, ForecastAmountMinor: forecast.ForecastAmountMinor, GeneratedAt: input.Now.UTC()}
}

func preserveExistingCharge(decision *MonthReconciliationDecision, input MonthReconciliationInput) bool {
	if input.Charge != nil {
		charge := *input.Charge
		decision.Charge = &charge
		decision.Projection.ChargeID = charge.ID
		return true
	}
	return input.Existing != nil && input.Existing.ChargeID != ""
}

func newMonthCharge(input MonthReconciliationInput, projection ContractMonth, forecast ContractMonthForecast, monthDate time.Time) (*Charge, error) {
	dueDay := input.DueDay
	if dueDay == 0 {
		dueDay = 5
	}
	if dueDay < 1 || dueDay > 28 {
		return nil, ErrInvalidDate
	}
	due := time.Date(monthDate.Year(), monthDate.Month(), dueDay, 0, 0, 0, 0, input.Location)
	return &Charge{ID: projection.ID + ":charge", AssignmentID: input.AssignmentID, SourceType: "regular_contract", SourceID: input.ContractID, Period: input.Month, OriginalAmountMinor: forecast.ForecastAmountMinor, CurrentAmountMinor: forecast.ForecastAmountMinor, Currency: forecast.Currency, SettlementState: PendingPayment, DueOn: due.Format("2006-01-02"), CreatedAt: input.Now.UTC()}, nil
}
