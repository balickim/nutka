package practice

import (
	"reflect"
	"testing"
	"time"
)

func TestWindowStart(t *testing.T) {
	now := time.Date(2030, time.March, 10, 12, 0, 0, 0, time.UTC)
	lessons := []time.Time{
		time.Date(2030, time.March, 3, 15, 0, 0, 0, time.UTC),
		time.Date(2030, time.March, 7, 23, 30, 0, 0, time.UTC),
		time.Date(2030, time.March, 12, 15, 0, 0, 0, time.UTC),
	}
	// 23:30 UTC on 7 March is 8 March in Warsaw.
	if got := WindowStart(lessons, now, now, warsaw); got != "2030-03-08" {
		t.Fatalf("latest started lesson: %s", got)
	}
	if got := WindowStart(lessons, lessons[1], now, warsaw); got != "2030-03-03" {
		t.Fatalf("previous lesson of a lesson: %s", got)
	}
	if got := WindowStart(nil, now, now, warsaw); got != "2030-03-04" {
		t.Fatalf("no lesson: %s", got)
	}
}

func TestSummarize(t *testing.T) {
	sessions := []SessionFact{
		{PracticedOn: "2030-03-05", Minutes: 40, Tasks: []string{"a"}, Comment: "przed lekcją"},
		{PracticedOn: "2030-03-08", Minutes: 20, Tasks: []string{"a", "b"}, Comment: "takt 5"},
		{PracticedOn: "2030-03-08", Minutes: 0, Tasks: []string{"a"}},
		{PracticedOn: "2030-03-09", Minutes: 15, Comment: "lepiej"},
	}
	summary := Summarize(sessions, "2030-03-08")
	if summary.Days != 2 || summary.Minutes != 35 || summary.Sessions != 3 {
		t.Fatalf("counts: %+v", summary)
	}
	if !reflect.DeepEqual(summary.TaskCounts, map[string]int{"a": 2, "b": 1}) {
		t.Fatalf("tasks: %v", summary.TaskCounts)
	}
	if !reflect.DeepEqual(summary.Comments, []Comment{{"2030-03-09", "lepiej"}, {"2030-03-08", "takt 5"}}) {
		t.Fatalf("comments: %v", summary.Comments)
	}
	if empty := Summarize(nil, "2030-03-08"); empty.Days != 0 || empty.Comments == nil {
		t.Fatalf("empty: %+v", empty)
	}
}

func TestRecentDays(t *testing.T) {
	now := time.Date(2030, time.March, 29, 12, 0, 0, 0, time.UTC)
	sessions := []SessionFact{{PracticedOn: "2030-03-20"}, {PracticedOn: "2030-03-01"}, {PracticedOn: "2030-03-02"}, {PracticedOn: "2030-03-20"}}
	if got := RecentDays(sessions, now, warsaw); !reflect.DeepEqual(got, []string{"2030-03-02", "2030-03-20"}) {
		t.Fatal(got)
	}
}
