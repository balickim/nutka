package practice

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var warsaw = mustLocation("Europe/Warsaw")

func mustLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return location
}

func TestValidateTask(t *testing.T) {
	cases := []struct {
		name string
		task Task
		want error
	}{
		{"title only", Task{Title: "Refren, tempo 70"}, nil},
		{"full", Task{Title: "Gamy", Details: "C-dur i G-dur", SuggestedMinutes: 120}, nil},
		{"empty title", NormalizeTask(Task{Title: "  "}), ErrInvalidTask},
		{"long title", Task{Title: strings.Repeat("ą", TitleMaxLength+1)}, ErrInvalidTask},
		{"long details", Task{Title: "x", Details: strings.Repeat("a", DetailsMaxLength+1)}, ErrInvalidTask},
		{"too many minutes", Task{Title: "x", SuggestedMinutes: 121}, ErrInvalidTask},
		{"negative minutes", Task{Title: "x", SuggestedMinutes: -1}, ErrInvalidTask},
	}
	for _, tc := range cases {
		if err := ValidateTask(tc.task); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestCheckDayUsesTheTeacherDay(t *testing.T) {
	// 23:30 UTC on 1 March is 00:30 on 2 March in Warsaw.
	now := time.Date(2030, time.March, 1, 23, 30, 0, 0, time.UTC)
	cases := []struct {
		date string
		want error
	}{
		{"2030-03-02", nil},
		{"2030-02-16", nil},
		{"2030-02-15", ErrInvalidSession},
		{"2030-03-03", ErrInvalidSession},
		{"2030-3-2", ErrInvalidSession},
	}
	for _, tc := range cases {
		if err := CheckDay(tc.date, now, warsaw); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v, want %v", tc.date, err, tc.want)
		}
	}
	if Today(now, warsaw) != "2030-03-02" || Today(now, time.UTC) != "2030-03-01" {
		t.Fatal("today does not follow the location")
	}
}

func TestValidateSession(t *testing.T) {
	now := time.Date(2030, time.March, 2, 12, 0, 0, 0, time.UTC)
	valid := Session{PracticedOn: "2030-03-02", Minutes: 30, Tasks: []string{"a"}, Comment: "takt 5"}
	if err := ValidateSession(valid, now, warsaw); err != nil {
		t.Fatal(err)
	}
	for name, session := range map[string]Session{
		"minutes":  {PracticedOn: "2030-03-02", Minutes: 241},
		"comment":  {PracticedOn: "2030-03-02", Comment: strings.Repeat("a", CommentMaxLength+1)},
		"tasks":    {PracticedOn: "2030-03-02", Tasks: make([]string, MaxSessionTasks+1)},
		"no day":   {},
		"negative": {PracticedOn: "2030-03-02", Minutes: -5},
	} {
		if err := ValidateSession(session, now, warsaw); !errors.Is(err, ErrInvalidSession) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestLearnerMayDelete(t *testing.T) {
	created := time.Date(2030, time.March, 1, 12, 0, 0, 0, time.UTC)
	if err := LearnerMayDelete(created, created.Add(DeleteWindow)); err != nil {
		t.Fatal(err)
	}
	if err := LearnerMayDelete(created, created.Add(DeleteWindow+time.Second)); !errors.Is(err, ErrLocked) {
		t.Fatalf("late delete: %v", err)
	}
}

func TestAddDaysCrossesDaylightSavingByCalendar(t *testing.T) {
	if got := AddDays("2030-03-31", -1); got != "2030-03-30" {
		t.Fatal(got)
	}
	if got := AddDays("bad", 1); got != "" {
		t.Fatal(got)
	}
}
