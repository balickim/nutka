package repertoire

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	got := Normalize(Input{Title: "  Hallelujah ", Artist: " Cohen ", Status: " learning "})
	if got != (Input{Title: "Hallelujah", Artist: "Cohen", Status: "learning"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestValidate(t *testing.T) {
	long := strings.Repeat("ą", TextMaxLength+1)
	cases := []struct {
		name  string
		input Input
		want  error
	}{
		{"title only", Input{Title: "Hallelujah", Status: StatusWish}, nil},
		{"every status", Input{Title: "Hallelujah", Artist: "Cohen", Status: StatusRepertoire}, nil},
		{"title at limit in runes", Input{Title: strings.Repeat("ą", TextMaxLength), Status: StatusPlaying}, nil},
		{"empty title", Input{Status: StatusLearning}, ErrInvalid},
		{"long title", Input{Title: long, Status: StatusLearning}, ErrInvalid},
		{"long artist", Input{Title: "Hallelujah", Artist: long, Status: StatusLearning}, ErrInvalid},
		{"unknown status", Input{Title: "Hallelujah", Status: "done"}, ErrInvalid},
		{"missing status", Input{Title: "Hallelujah"}, ErrInvalid},
	}
	for _, tc := range cases {
		if err := Validate(tc.input); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestLearnerMayDelete(t *testing.T) {
	cases := []struct {
		name       string
		proposedBy string
		status     string
		want       error
	}{
		{"own wish", ProposedByLearner, StatusWish, nil},
		{"accepted wish", ProposedByLearner, StatusLearning, ErrLocked},
		{"teacher wish", ProposedByTeacher, StatusWish, ErrLocked},
		{"teacher piece", ProposedByTeacher, StatusPlaying, ErrLocked},
	}
	for _, tc := range cases {
		if err := LearnerMayDelete(tc.proposedBy, tc.status); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v, want %v", tc.name, err, tc.want)
		}
	}
}
