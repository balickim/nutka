// This file formats UTC instants and provides deterministic ordering for calendar views.
package commercialread

import (
	"sort"
	"time"
)

func sortLessons(values []LessonView) {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].StartAt < values[j].StartAt || values[i].StartAt == values[j].StartAt && values[i].ID < values[j].ID
	})
}

func sortOccurrences(values []ContractOccurrenceView) {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].StartAt < values[j].StartAt || values[i].StartAt == values[j].StartAt && values[i].ID < values[j].ID
	})
}

func instant(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func dateString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}
