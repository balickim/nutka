package lessonnotes

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCheckLesson(t *testing.T) {
	now := time.Date(2030, time.January, 2, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		start time.Time
		state string
		want  error
	}{
		{"started", now.Add(-time.Hour), "scheduled", nil},
		{"starts now", now, "scheduled", nil},
		{"future", now.Add(time.Minute), "scheduled", ErrNotStarted},
		{"cancelled", now.Add(-time.Hour), "cancelled", ErrCancelled},
	}
	for _, tc := range cases {
		if err := CheckLesson(tc.start, tc.state, now); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := Validate("<p>Dobrze</p>", MaxMaterials); err != nil {
		t.Fatalf("valid note rejected: %v", err)
	}
	for name, err := range map[string]error{
		"empty":          Validate("", 0),
		"too long":       Validate(strings.Repeat("a", BodyMaxBytes+1), 0),
		"many materials": Validate("<p>x</p>", MaxMaterials+1),
	} {
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s accepted: %v", name, err)
		}
	}
}
