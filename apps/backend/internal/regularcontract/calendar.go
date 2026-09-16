// This file resolves teacher-local contract dates into UTC intervals and groups business months.
package regularcontract

import (
	"fmt"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

func teacherDate(value time.Time, location *time.Location) time.Time {
	year, month, day := value.In(location).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}

func dateKey(value time.Time, location *time.Location) string {
	year, month, day := value.In(location).Date()
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func parseDateKey(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidContract
	}
	return parsed, nil
}

func occurrenceAfterLocalDate(value string, boundary time.Time, location *time.Location) bool {
	if _, err := parseDateKey(value); err != nil {
		return false
	}
	return value > dateKey(boundary, location)
}

func localDate(value time.Time, location *time.Location) time.Time {
	date := teacherDate(value, location)
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
}

func nextContractEnd(start time.Time, location *time.Location, policy Policy) time.Time {
	date := localDate(start, location)
	year := date.Year()
	end := time.Date(year, policy.ContractEndMonth, policy.ContractEndDay, 0, 0, 0, 0, location)
	if end.Before(date) {
		end = end.AddDate(1, 0, 0)
	}
	return end
}

func resolveOccurrence(date time.Time, minute int, timezone string, duration time.Duration) (scheduling.Interval, error) {
	location, err := scheduling.LoadTimezone(timezone)
	if err != nil {
		return scheduling.Interval{}, err
	}
	start, err := scheduling.ResolveWallTime(location, date.Year(), date.Month(), date.Day(), minute/60, minute%60, 0, 0)
	if err != nil {
		return scheduling.Interval{}, err
	}
	return scheduling.NewInterval(start, start.Add(duration))
}

func containsInterval(windows []scheduling.Interval, candidate scheduling.Interval) bool {
	for _, window := range windows {
		if window.Contains(candidate) {
			return true
		}
	}
	return false
}

func intersectsAny(interval scheduling.Interval, values []scheduling.Interval) bool {
	for _, value := range values {
		if interval.Intersects(value) {
			return true
		}
	}
	return false
}

func monthKey(value time.Time, location *time.Location) string {
	year, month, _ := value.In(location).Date()
	return fmt.Sprintf("%04d-%02d", year, month)
}

func firstOfMonth(value time.Time, location *time.Location) time.Time {
	year, month, _ := value.In(location).Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, location)
}

func nextMonth(value time.Time, location *time.Location) time.Time {
	return firstOfMonth(value, location).AddDate(0, 1, 0)
}

func lastOfMonth(value time.Time, location *time.Location) time.Time {
	return nextMonth(value, location).AddDate(0, 0, -1)
}

func dueDate(month time.Time, location *time.Location) time.Time {
	year, monthValue, _ := month.In(location).Date()
	return time.Date(year, monthValue, 5, 0, 0, 0, 0, location)
}
