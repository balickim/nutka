// This file derives teacher-local business dates for package, contract, and payment rules.
package businesspolicy

import (
	"errors"
	"time"
)

var ErrInvalidCalendarDate = errors.New("calendar date is invalid")

func locationForCalendar(timezone string) (*time.Location, error) {
	if timezone == "" || timezone == "Local" {
		return nil, ErrInvalidCalendarDate
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, ErrInvalidCalendarDate
	}
	return location, nil
}

// TeacherDate returns midnight for the calendar date containing instant in timezone.
func TeacherDate(instant time.Time, timezone string) (time.Time, error) {
	if instant.IsZero() {
		return time.Time{}, ErrInvalidCalendarDate
	}
	location, err := locationForCalendar(timezone)
	if err != nil {
		return time.Time{}, err
	}
	local := instant.In(location)
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location), nil
}

// AddTeacherDays advances a teacher-local date without converting calendar days to hours.
func AddTeacherDays(date time.Time, days int, timezone string) (time.Time, error) {
	localDate, err := TeacherDate(date, timezone)
	if err != nil {
		return time.Time{}, err
	}
	return localDate.AddDate(0, 0, days), nil
}

// PackageValidThrough includes purchaseDate as validity day one.
func PackageValidThrough(purchaseDate time.Time, validityDays int, timezone string) (time.Time, error) {
	if validityDays <= 0 {
		return time.Time{}, ErrInvalidCalendarDate
	}
	return AddTeacherDays(purchaseDate, validityDays-1, timezone)
}

// ExtendPackageValidity adds local calendar days to an existing validity date.
func ExtendPackageValidity(validThrough time.Time, extensionDays int, timezone string) (time.Time, error) {
	if extensionDays <= 0 {
		return time.Time{}, ErrInvalidCalendarDate
	}
	return AddTeacherDays(validThrough, extensionDays, timezone)
}

func MonthStart(date time.Time, timezone string) (time.Time, error) {
	localDate, err := TeacherDate(date, timezone)
	if err != nil {
		return time.Time{}, err
	}
	year, month, _ := localDate.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, localDate.Location()), nil
}

func MonthEnd(date time.Time, timezone string) (time.Time, error) {
	start, err := MonthStart(date, timezone)
	if err != nil {
		return time.Time{}, err
	}
	return start.AddDate(0, 1, -1), nil
}

// NextJune30 returns the next applicable configured contract end date.
func NextJune30(date time.Time, timezone string) (time.Time, error) {
	localDate, err := TeacherDate(date, timezone)
	if err != nil {
		return time.Time{}, err
	}
	policy := Current()
	year, month, day := localDate.Date()
	end := time.Date(year, policy.ContractEndMonth, policy.ContractEndDay, 0, 0, 0, 0, localDate.Location())
	if month > policy.ContractEndMonth || (month == policy.ContractEndMonth && day > policy.ContractEndDay) {
		end = end.AddDate(1, 0, 0)
	}
	return end, nil
}

// NoticeEnd returns the final date of the local month after the notice month.
func NoticeEnd(noticeAt time.Time, timezone string) (time.Time, error) {
	localDate, err := TeacherDate(noticeAt, timezone)
	if err != nil {
		return time.Time{}, err
	}
	return MonthEnd(localDate.AddDate(0, 1, 0), timezone)
}

func ReplacementDeadline(originalStart time.Time, replacementDays int, timezone string) (time.Time, error) {
	if replacementDays <= 0 {
		return time.Time{}, ErrInvalidCalendarDate
	}
	return AddTeacherDays(originalStart, replacementDays, timezone)
}

func (p Policy) PackageValidThrough(purchaseDate time.Time, timezone string) (time.Time, error) {
	return PackageValidThrough(purchaseDate, p.PackageValidityDays, timezone)
}

func (p Policy) ExtendPackageValidity(validThrough time.Time, timezone string) (time.Time, error) {
	return ExtendPackageValidity(validThrough, p.TeacherCancellationExtensionDays, timezone)
}

func (p Policy) ReplacementDeadline(originalStart time.Time, timezone string) (time.Time, error) {
	return ReplacementDeadline(originalStart, p.ContractReplacementDays, timezone)
}

func (p Policy) MonthStart(date time.Time, timezone string) (time.Time, error) {
	return MonthStart(date, timezone)
}

func (p Policy) MonthEnd(date time.Time, timezone string) (time.Time, error) {
	return MonthEnd(date, timezone)
}

func (p Policy) NoticeEnd(noticeAt time.Time, timezone string) (time.Time, error) {
	return NoticeEnd(noticeAt, timezone)
}
