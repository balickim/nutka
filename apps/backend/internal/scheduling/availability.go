// This file expands weekly local rules and merges UTC exceptions into effective availability windows.
package scheduling

import "time"

// ExpandWeeklyRules converts recurring local wall-clock windows into UTC intervals.
func ExpandWeeklyRules(start, end time.Time, timezone string, rules []WeekdayRule) ([]Interval, error) {
	query, err := NewInterval(start, end)
	if err != nil {
		return nil, err
	}
	location, err := LoadTimezone(timezone)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if rule.Enabled && !rule.fieldsValid() {
			return nil, ErrInvalidRule
		}
	}
	localStart := query.Start.In(location)
	localEnd := query.End.In(location)
	date := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, location).AddDate(0, 0, -1)
	lastDate := time.Date(localEnd.Year(), localEnd.Month(), localEnd.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
	intervals := make([]Interval, 0)
	for !date.After(lastDate) {
		for _, rule := range rules {
			candidate, ok, candidateErr := weeklyRuleInterval(date, rule, location, query)
			if candidateErr != nil {
				return nil, candidateErr
			}
			if ok {
				intervals = append(intervals, candidate)
			}
		}
		date = date.AddDate(0, 0, 1)
	}
	return mergeIntervals(intervals), nil
}

func weeklyRuleInterval(date time.Time, rule WeekdayRule, location *time.Location, query Interval) (Interval, bool, error) {
	if !rule.Enabled {
		return Interval{}, false, nil
	}
	if !rule.fieldsValid() {
		return Interval{}, false, ErrInvalidRule
	}
	if rule.Weekday != date.Weekday() {
		return Interval{}, false, nil
	}
	windowStart, startErr := localDateTime(date, rule.StartMinute, location)
	windowEnd, endErr := localDateTime(date, rule.EndMinute, location)
	if startErr != nil || endErr != nil {
		return Interval{}, false, ErrInvalidRule
	}
	candidate, candidateErr := NewInterval(windowStart, windowEnd)
	if candidateErr != nil {
		return Interval{}, false, ErrInvalidRule
	}
	clipped, ok := clipInterval(candidate, query)
	return clipped, ok, nil
}

// EffectiveAvailability merges recurring and available exceptions, then subtracts every unavailable exception.
func EffectiveAvailability(start, end time.Time, timezone string, rules []WeekdayRule, exceptions []AvailabilityException) ([]Interval, error) {
	query, err := NewInterval(start, end)
	if err != nil {
		return nil, err
	}
	available, err := ExpandWeeklyRules(query.Start, query.End, timezone, rules)
	if err != nil {
		return nil, err
	}
	unavailable := make([]Interval, 0)
	for _, exception := range exceptions {
		if !exception.Interval.Valid() {
			return nil, ErrInvalidException
		}
		if exception.Kind != AvailableException && exception.Kind != UnavailableException {
			return nil, ErrInvalidExceptionKind
		}
		clipped, overlaps := clipInterval(exception.Interval, query)
		if !overlaps {
			continue
		}
		if exception.Kind == AvailableException {
			available = append(available, clipped)
		} else {
			unavailable = append(unavailable, clipped)
		}
	}
	available = mergeIntervals(available)
	for _, block := range mergeIntervals(unavailable) {
		available = subtractIntervalSet(available, block)
	}
	return available, nil
}

func clipInterval(value, bounds Interval) (Interval, bool) {
	start, end := value.Start, value.End
	if start.Before(bounds.Start) {
		start = bounds.Start
	}
	if end.After(bounds.End) {
		end = bounds.End
	}
	if !start.Before(end) {
		return Interval{}, false
	}
	return Interval{Start: start, End: end}, true
}

func subtractIntervalSet(values []Interval, block Interval) []Interval {
	result := make([]Interval, 0, len(values))
	for _, value := range values {
		if !value.Intersects(block) {
			result = append(result, value)
			continue
		}
		if value.Start.Before(block.Start) {
			result = append(result, Interval{Start: value.Start, End: block.Start})
		}
		if block.End.Before(value.End) {
			result = append(result, Interval{Start: block.End, End: value.End})
		}
	}
	return result
}
