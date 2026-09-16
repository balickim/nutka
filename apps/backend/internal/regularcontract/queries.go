// This file exposes horizon-scoped series views, monthly charge inputs, and obligation blockers.
package regularcontract

import (
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

// Partition separates teacher-visible occurrences by the current learner booking horizon.
func (c RegularContract) Partition(now time.Time) (SeriesPartition, error) {
	if _, err := c.location(); err != nil {
		return SeriesPartition{}, err
	}
	horizon := now.UTC().Add(c.lifecyclePolicy().BookingHorizon)
	partition := SeriesPartition{Near: make([]Occurrence, 0), Later: make([]Occurrence, 0)}
	for _, occurrence := range c.Occurrences {
		if occurrence.Interval.Start.After(now.UTC()) && !occurrence.Interval.Start.After(horizon) {
			partition.Near = append(partition.Near, occurrence)
			continue
		}
		if occurrence.Interval.Start.After(horizon) {
			partition.Later = append(partition.Later, occurrence)
		}
	}
	return partition, nil
}

// LearnerOccurrences returns only future occurrences inside the configured start horizon.
func (c RegularContract) LearnerOccurrences(now time.Time) ([]Occurrence, error) {
	partition, err := c.Partition(now)
	if err != nil {
		return nil, err
	}
	return partition.Near, nil
}

// TeacherOccurrences returns the complete materialized series, including later reservations.
func (c RegularContract) TeacherOccurrences() []Occurrence {
	result := make([]Occurrence, len(c.Occurrences))
	copy(result, c.Occurrences)
	return result
}

// ActiveAt derives whether a contract still covers a teacher-local calendar date.
func (c RegularContract) ActiveAt(now time.Time) bool {
	location, err := c.location()
	if err != nil || c.Status == Ended || now.IsZero() {
		return false
	}
	date := localDate(now, location)
	return !date.Before(c.StartOn) && !date.After(c.EffectiveEndOn)
}

// MonthSnapshots computes forecasts and charges from billable occurrence snapshots.
func (c RegularContract) MonthSnapshots(now time.Time) ([]MonthSnapshot, error) {
	location, err := c.location()
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		return nil, ErrInvalidContract
	}
	months := make(map[string]*MonthSnapshot)
	for _, occurrence := range c.Occurrences {
		if err := c.addMonthOccurrence(months, occurrence, location); err != nil {
			return nil, err
		}
	}
	currentMonth := monthKey(now, location)
	result := make([]MonthSnapshot, 0, len(months))
	for month, snapshot := range months {
		snapshot.Forecast = month > currentMonth
		snapshot.Charge = !snapshot.Forecast
		result = append(result, *snapshot)
	}
	// Month strings sort lexicographically in calendar order.
	for left := 0; left < len(result); left++ {
		for right := left + 1; right < len(result); right++ {
			if result[right].Month < result[left].Month {
				result[left], result[right] = result[right], result[left]
			}
		}
	}
	return result, nil
}

func (c RegularContract) addMonthOccurrence(months map[string]*MonthSnapshot, occurrence Occurrence, location *time.Location) error {
	month := occurrence.OriginalLocalMonth()
	if month == "" {
		return nil
	}
	snapshot, err := contractMonthSnapshot(months, month, location)
	if err != nil {
		return err
	}
	snapshot.OccurrenceIDs = append(snapshot.OccurrenceIDs, occurrence.ID)
	if !occurrence.Billable() {
		return nil
	}
	snapshot.BillableCount++
	price, _ := c.priceForMonth(month)
	if occurrence.UnitPriceMinor > 0 {
		price = occurrence.UnitPriceMinor
	}
	snapshot.AmountMinor += price
	return nil
}

func contractMonthSnapshot(months map[string]*MonthSnapshot, month string, location *time.Location) (*MonthSnapshot, error) {
	if snapshot := months[month]; snapshot != nil {
		return snapshot, nil
	}
	parsed, err := time.ParseInLocation("2006-01", month, location)
	if err != nil {
		return nil, err
	}
	snapshot := &MonthSnapshot{Month: month, DueOn: dueDate(parsed, location)}
	months[month] = snapshot
	return snapshot, nil
}

// CanChangeTimezone blocks ordinary profile edits while obligations retain local meaning.
func CanChangeTimezone(current, proposed string, activeContract bool, futureLessons []time.Time) error {
	if current == proposed {
		return nil
	}
	if activeContract || len(futureLessons) > 0 {
		return ErrTimezoneChangeBlocked
	}
	return nil
}

// CanChangeTimezoneWithLessons is the adapter used by scheduling records that retain full intervals.
func CanChangeTimezoneWithLessons(current, proposed string, activeContract bool, lessons []scheduling.Lesson, now time.Time) error {
	if current == proposed {
		return nil
	}
	for _, lesson := range lessons {
		if lesson.Status != scheduling.Cancelled && lesson.Interval.Start.After(now.UTC()) {
			return ErrTimezoneChangeBlocked
		}
	}
	return CanChangeTimezone(current, proposed, activeContract, nil)
}

// CanDeactivateAssignment protects unresolved commercial and scheduled obligations.
func CanDeactivateAssignment(obligations ObligationSet) error {
	if obligations.ActiveContract || obligations.AvailablePackageTokens > 0 || obligations.ReservedPackageLessons > 0 || obligations.FutureScheduledLessons > 0 {
		return ErrAssignmentBlocked
	}
	return nil
}
