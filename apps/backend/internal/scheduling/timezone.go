// This file validates IANA zones and resolves local recurring wall times at the UTC boundary.
package scheduling

import (
	"sort"
	"time"
)

func LoadTimezone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, ErrInvalidTimezone
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, ErrInvalidTimezone
	}
	return location, nil
}

func ValidateTimezone(name string) error {
	_, err := LoadTimezone(name)
	return err
}

func NormalizeTimezone(name string) (string, error) {
	if name == "" {
		name = DefaultTimezone
	}
	if err := ValidateTimezone(name); err != nil {
		return "", err
	}
	return name, nil
}

// ResolveWallTime uses the earliest matching instant for an ambiguous wall time.
// For a nonexistent time, it shifts the requested wall time forward by the DST gap.
func ResolveWallTime(location *time.Location, year int, month time.Month, day, hour, minute, second, nanosecond int) (time.Time, error) {
	if location == nil {
		return time.Time{}, ErrInvalidTimezone
	}
	if !validWallDate(year, month, day) || !validWallClock(hour, minute, second, nanosecond) {
		return time.Time{}, ErrInvalidRule
	}
	wall := time.Date(year, month, day, hour, minute, second, nanosecond, time.UTC)
	offsets := wallOffsets(location, wall)
	candidates := wallCandidates(location, wall, offsets, year, month, day, hour, minute, second, nanosecond)
	if len(candidates) > 0 {
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].Before(candidates[j]) })
		return candidates[0], nil
	}
	return resolveNonexistentWall(location, wall, year, month, day, hour, minute, second, nanosecond)
}

func validWallDate(year int, month time.Month, day int) bool {
	if month < time.January || month > time.December || day < 1 || day > 31 {
		return false
	}
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	dateYear, dateMonth, dateDay := date.Date()
	return dateYear == year && dateMonth == month && dateDay == day
}

func validWallClock(hour, minute, second, nanosecond int) bool {
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59 && second >= 0 && second <= 59 && nanosecond >= 0 && nanosecond < int(time.Second)
}

func wallCandidates(location *time.Location, wall time.Time, offsets map[int]struct{}, year int, month time.Month, day, hour, minute, second, nanosecond int) []time.Time {
	candidates := make([]time.Time, 0, len(offsets))
	for offset := range offsets {
		candidate := wall.Add(-time.Duration(offset) * time.Second)
		if sameWall(candidate.In(location), year, month, day, hour, minute, second, nanosecond) {
			candidates = append(candidates, candidate.UTC())
		}
	}
	return candidates
}

func resolveNonexistentWall(location *time.Location, wall time.Time, year int, month time.Month, day, hour, minute, second, nanosecond int) (time.Time, error) {

	// time.Date resolves a gap to a nearby valid local time. Compare offsets on
	// both sides and apply the offset increase to preserve the wall-clock offset.
	normalized := time.Date(year, month, day, hour, minute, second, nanosecond, location)
	_, beforeOffset := normalized.Zone()
	afterOffset := beforeOffset
	for probe := normalized.UTC(); probe.Before(normalized.UTC().Add(7 * 24 * time.Hour)); probe = probe.Add(15 * time.Minute) {
		_, offset := probe.In(location).Zone()
		if offset > beforeOffset {
			afterOffset = offset
			break
		}
	}
	if afterOffset <= beforeOffset {
		return time.Time{}, ErrInvalidTimezone
	}
	shifted := wall.Add(time.Duration(afterOffset-beforeOffset) * time.Second)
	return time.Date(shifted.Year(), shifted.Month(), shifted.Day(), shifted.Hour(), shifted.Minute(), shifted.Second(), shifted.Nanosecond(), location).UTC(), nil
}

func wallOffsets(location *time.Location, wall time.Time) map[int]struct{} {
	offsets := make(map[int]struct{})
	for candidate := wall.Add(-36 * time.Hour); !candidate.After(wall.Add(36 * time.Hour)); candidate = candidate.Add(15 * time.Minute) {
		_, offset := candidate.In(location).Zone()
		offsets[offset] = struct{}{}
	}
	return offsets
}

func sameWall(value time.Time, year int, month time.Month, day, hour, minute, second, nanosecond int) bool {
	vYear, vMonth, vDay := value.Date()
	vHour, vMinute, vSecond := value.Clock()
	return sameWallDate(vYear, vMonth, vDay, year, month, day) && sameWallClock(vHour, vMinute, vSecond, value.Nanosecond(), hour, minute, second, nanosecond)
}

func sameWallDate(valueYear int, valueMonth time.Month, valueDay, year int, month time.Month, day int) bool {
	return valueYear == year && valueMonth == month && valueDay == day
}

func sameWallClock(valueHour, valueMinute, valueSecond, valueNanosecond, hour, minute, second, nanosecond int) bool {
	return valueHour == hour && valueMinute == minute && valueSecond == second && valueNanosecond == nanosecond
}

func localDateTime(date time.Time, minute int, location *time.Location) (time.Time, error) {
	if minute < 0 || minute > 24*60 {
		return time.Time{}, ErrInvalidRule
	}
	if minute == 24*60 {
		next := date.AddDate(0, 0, 1)
		year, month, day := next.Date()
		return ResolveWallTime(location, year, month, day, 0, 0, 0, 0)
	}
	year, month, day := date.Date()
	return ResolveWallTime(location, year, month, day, minute/60, minute%60, 0, 0)
}
