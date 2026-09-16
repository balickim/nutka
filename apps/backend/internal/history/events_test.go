package history

import (
	"testing"
	"time"
)

func validInput() EventInput {
	return EventInput{
		AggregateType: "lesson", AggregateID: "lesson-1", AssignmentID: "assignment-1",
		Actor: Actor{Role: LearnerActor, ID: "learner-1"}, EventAt: time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC),
		PriorState: map[string]any{"start_at": "2026-09-14T10:00:00Z"},
		NewState:   map[string]any{"start_at": "2026-09-15T10:00:00Z"},
	}
}

func TestConstructorsUseEnglishTypesAndCopyState(t *testing.T) {
	input := validInput()
	event, err := NewLessonRescheduled(input)
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != LessonRescheduled || event.EventAt.Location() != time.UTC {
		t.Fatalf("unexpected event: %#v", event)
	}
	input.NewState["start_at"] = "changed"
	if event.NewState["start_at"] == "changed" {
		t.Fatal("constructor retained mutable input state")
	}
}

func TestCorrectionRequiresReasonAndCompensatesOriginalEffect(t *testing.T) {
	original, err := NewPackageTokenUsed(validInput())
	if err != nil {
		t.Fatal(err)
	}
	original.ID = "original"
	if _, err := NewCorrection(validInput(), ""); err != ErrCorrectionReason {
		t.Fatalf("missing reason error: %v", err)
	}
	correctionInput := validInput()
	correctionInput.Reason = "Corrected mistaken token usage"
	correction, err := NewCorrection(correctionInput, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	correction.ID = "correction"
	current := CurrentEffects([]Event{original, correction})
	if len(current) != 1 || current[0].ID != correction.ID {
		t.Fatalf("unexpected current effects: %#v", current)
	}
	if len(Uncompensated([]Event{original, correction})) != 1 {
		t.Fatal("compensated event remained active")
	}
}

func TestCorrectionOfCorrectionRestoresOriginalEffect(t *testing.T) {
	original, err := NewPackageTokenUsed(validInput())
	if err != nil {
		t.Fatal(err)
	}
	original.ID = "original"
	correctionInput := validInput()
	correctionInput.Reason = "Reverse mistaken token usage"
	correction, err := NewCorrection(correctionInput, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	correction.ID = "correction"
	secondInput := validInput()
	secondInput.Reason = "Restore the valid token usage"
	second, err := NewCorrection(secondInput, correction.ID)
	if err != nil {
		t.Fatal(err)
	}
	second.ID = "correction-of-correction"
	current := CurrentEffects([]Event{original, correction, second})
	if len(current) != 2 || current[0].ID != original.ID || current[1].ID != second.ID {
		t.Fatalf("correction chain effects: %#v", current)
	}
}

func TestSystemActorMayOmitIdentifierButHumanActorMayNot(t *testing.T) {
	input := validInput()
	input.Actor = Actor{Role: SystemActor}
	if _, err := NewPackageTokenExpired(input); err != nil {
		t.Fatal(err)
	}
	input.Actor = Actor{Role: TeacherActor}
	if _, err := NewPackageClosed(input); err == nil {
		t.Fatal("human event without actor identifier was accepted")
	}
}
