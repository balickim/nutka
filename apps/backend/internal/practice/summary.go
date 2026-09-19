// This file computes the practice summary window and the practice summary from stored sessions.
package practice

import (
	"sort"
	"time"
)

// DefaultWindowDays is the window length when the assignment has no earlier lesson.
const DefaultWindowDays = 7

// RecentWindowDays is the length of the learner practice calendar.
const RecentWindowDays = 28

// SessionFact is one stored session as the summary reads it.
type SessionFact struct {
	PracticedOn string
	Minutes     int
	Tasks       []string
	Comment     string
}

// Comment is one learner comment with its day.
type Comment struct {
	PracticedOn string `json:"practiced_on"`
	Comment     string `json:"comment"`
}

// Summary describes the practice since the window start.
type Summary struct {
	SinceOn    string         `json:"since_on"`
	Days       int            `json:"days"`
	Minutes    int            `json:"minutes"`
	Sessions   int            `json:"sessions"`
	TaskCounts map[string]int `json:"-"`
	Comments   []Comment      `json:"comments"`
}

// WindowStart returns the local day of the latest lesson start that is before reference, or DefaultWindowDays including today without such a lesson.
func WindowStart(lessonStarts []time.Time, reference, now time.Time, location *time.Location) string {
	var latest time.Time
	for _, start := range lessonStarts {
		if start.Before(reference) && start.After(latest) {
			latest = start
		}
	}
	if latest.IsZero() {
		return AddDays(Today(now, location), -(DefaultWindowDays - 1))
	}
	return latest.In(location).Format(DateLayout)
}

// Summarize counts sessions on or after sinceOn. Comments are newest first.
func Summarize(sessions []SessionFact, sinceOn string) Summary {
	summary := Summary{SinceOn: sinceOn, TaskCounts: map[string]int{}, Comments: []Comment{}}
	days := map[string]bool{}
	for _, session := range sessions {
		if session.PracticedOn < sinceOn {
			continue
		}
		days[session.PracticedOn] = true
		summary.Sessions++
		summary.Minutes += session.Minutes
		for _, task := range session.Tasks {
			summary.TaskCounts[task]++
		}
		if session.Comment != "" {
			summary.Comments = append(summary.Comments, Comment{PracticedOn: session.PracticedOn, Comment: session.Comment})
		}
	}
	summary.Days = len(days)
	sort.SliceStable(summary.Comments, func(left, right int) bool {
		return summary.Comments[left].PracticedOn > summary.Comments[right].PracticedOn
	})
	return summary
}

// RecentDays returns the sorted practice days of the RecentWindowDays that end today.
func RecentDays(sessions []SessionFact, now time.Time, location *time.Location) []string {
	first := AddDays(Today(now, location), -(RecentWindowDays - 1))
	seen := map[string]bool{}
	days := []string{}
	for _, session := range sessions {
		if session.PracticedOn >= first && !seen[session.PracticedOn] {
			seen[session.PracticedOn] = true
			days = append(days, session.PracticedOn)
		}
	}
	sort.Strings(days)
	return days
}
