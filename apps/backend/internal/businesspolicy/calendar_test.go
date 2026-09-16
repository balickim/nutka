package businesspolicy

import (
	"testing"
	"time"
)

func TestTeacherCalendarDateRules(t *testing.T) {
	zone := "Europe/Warsaw"
	purchase := time.Date(2026, time.September, 13, 12, 30, 0, 0, time.UTC)
	validThrough, err := PackageValidThrough(purchase, 60, zone)
	if err != nil {
		t.Fatal(err)
	}
	if year, month, day := validThrough.Date(); year != 2026 || month != time.November || day != 11 {
		t.Fatalf("package validity must count local purchase day: %s", validThrough)
	}
	extended, err := ExtendPackageValidity(validThrough, 7, zone)
	if err != nil {
		t.Fatal(err)
	}
	if extended.Day() != 18 || extended.Month() != time.November {
		t.Fatalf("unexpected extension: %s", extended)
	}
	monthStart, err := MonthStart(purchase, zone)
	if err != nil || monthStart.Day() != 1 || monthStart.Month() != time.September {
		t.Fatalf("unexpected month start: %s %v", monthStart, err)
	}
	monthEnd, err := MonthEnd(purchase, zone)
	if err != nil || monthEnd.Day() != 30 || monthEnd.Month() != time.September {
		t.Fatalf("unexpected month end: %s %v", monthEnd, err)
	}
}

func TestContractNoticeAndReplacementDates(t *testing.T) {
	zone := "Europe/Warsaw"
	noticeEnd, err := NoticeEnd(time.Date(2026, time.October, 14, 10, 0, 0, 0, time.UTC), zone)
	if err != nil || noticeEnd.Month() != time.November || noticeEnd.Day() != 30 {
		t.Fatalf("notice must end after the following local month: %s %v", noticeEnd, err)
	}
	deadline, err := ReplacementDeadline(time.Date(2026, time.June, 25, 10, 0, 0, 0, time.UTC), 30, zone)
	if err != nil || deadline.Month() != time.July || deadline.Day() != 25 {
		t.Fatalf("unexpected replacement deadline: %s %v", deadline, err)
	}
	end, err := NextJune30(time.Date(2026, time.July, 1, 10, 0, 0, 0, time.UTC), zone)
	if err != nil || end.Year() != 2027 || end.Month() != time.June || end.Day() != 30 {
		t.Fatalf("unexpected contract end: %s %v", end, err)
	}
}
