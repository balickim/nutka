package history

import (
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func historyTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir(), DefaultDev: false})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	assignment := core.NewBaseCollection("teacher_learners")
	assignment.Fields.Add(&core.TextField{Name: "teacher"})
	assignment.Fields.Add(&core.TextField{Name: "learner"})
	events := core.NewBaseCollection(BusinessEventsCollection)
	events.Fields.Add(&core.TextField{Name: AssignmentField, Required: true})
	events.Fields.Add(&core.TextField{Name: AggregateTypeField, Required: true})
	events.Fields.Add(&core.TextField{Name: AggregateIDField, Required: true})
	events.Fields.Add(&core.TextField{Name: EventTypeField, Required: true})
	events.Fields.Add(&core.SelectField{Name: ActorRoleField, Values: []string{string(TeacherActor), string(LearnerActor), string(SystemActor)}, Required: true})
	events.Fields.Add(&core.TextField{Name: ActorIDField})
	events.Fields.Add(&core.DateField{Name: EventAtField, Required: true})
	events.Fields.Add(&core.JSONField{Name: RelatedIDsField})
	events.Fields.Add(&core.JSONField{Name: PriorStateField})
	events.Fields.Add(&core.JSONField{Name: NewStateField})
	events.Fields.Add(&core.TextField{Name: ReasonField})
	events.Fields.Add(&core.TextField{Name: InternalNoteField})
	events.Fields.Add(&core.TextField{Name: CorrectsEventField})
	if err := app.Save(assignment); err != nil {
		t.Fatal(err)
	}
	if err := app.Save(events); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}

func seedHistoryData(t *testing.T, app core.App) string {
	t.Helper()
	assignment, err := app.FindCollectionByNameOrId("teacher_learners")
	if err != nil {
		t.Fatal(err)
	}
	row := core.NewRecord(assignment)
	row.Set("teacher", "teacher-1")
	row.Set("learner", "learner-1")
	if err := app.Save(row); err != nil {
		t.Fatal(err)
	}
	store := Storage{}
	if err := app.RunInTransaction(func(tx core.App) error {
		first, eventErr := NewLessonCreated(EventInput{
			AggregateType: "lesson", AggregateID: "lesson-1", AssignmentID: row.Id,
			Actor: Actor{Role: TeacherActor, ID: "teacher-1"}, EventAt: time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC),
			NewState: map[string]any{"status": "scheduled", "internal_note": "also private"}, InternalNote: "teacher-only note",
		})
		if eventErr != nil {
			return eventErr
		}
		return store.Append(tx, first)
	}); err != nil {
		t.Fatal(err)
	}
	return row.Id
}

func TestHistoryAuthorizationAndLearnerRedaction(t *testing.T) {
	app := historyTestApp(t)
	assignmentID := seedHistoryData(t, app)
	teacherPage, err := QueryPage(app, assignmentID, Viewer{Role: TeacherActor, ID: "teacher-1"}, 1, 20)
	if err != nil || len(teacherPage.Items) != 1 || teacherPage.Items[0].InternalNote != "teacher-only note" {
		t.Fatalf("teacher history: %#v, %v", teacherPage, err)
	}
	learnerPage, err := QueryPage(app, assignmentID, Viewer{Role: LearnerActor, ID: "learner-1"}, 1, 20)
	if err != nil || len(learnerPage.Items) != 1 || learnerPage.Items[0].InternalNote != "" || learnerPage.Items[0].NewState["internal_note"] != nil {
		t.Fatalf("learner redaction: %#v, %v", learnerPage, err)
	}
	if _, err := QueryPage(app, assignmentID, Viewer{Role: TeacherActor, ID: "other-teacher"}, 1, 20); err != ErrHistoryUnauthorized {
		t.Fatalf("unrelated teacher authorization: %v", err)
	}
	if _, err := QueryPage(app, assignmentID, Viewer{Role: LearnerActor, ID: "other-learner"}, 1, 20); err != ErrHistoryUnauthorized {
		t.Fatalf("unrelated learner authorization: %v", err)
	}
}

func TestHistoryPaginationIsStableAndBounded(t *testing.T) {
	app := historyTestApp(t)
	assignmentID := seedHistoryData(t, app)
	for index := 0; index < 2; index++ {
		if err := app.RunInTransaction(func(tx core.App) error {
			event, eventErr := NewLessonCreated(EventInput{
				AggregateType: "lesson", AggregateID: "lesson-extra", AssignmentID: assignmentID,
				Actor: Actor{Role: SystemActor}, EventAt: time.Date(2026, 9, 13, 10+index, 0, 0, 0, time.UTC),
			})
			if eventErr != nil {
				return eventErr
			}
			return (Storage{}).Append(tx, event)
		}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := QueryPage(app, assignmentID, Viewer{Role: TeacherActor, ID: "teacher-1"}, 2, 2)
	if err != nil || len(page.Items) != 1 || page.TotalItems != 3 || page.TotalPages != 2 {
		t.Fatalf("unexpected pagination: %#v, %v", page, err)
	}
	if _, err := QueryPage(app, assignmentID, Viewer{Role: TeacherActor, ID: "teacher-1"}, 1, 101); err != ErrHistoryPagination {
		t.Fatalf("unbounded page accepted: %v", err)
	}
}

func TestStorageRequiresTransactionAndRollsBackWithCaller(t *testing.T) {
	app := historyTestApp(t)
	event, err := NewLessonCreated(EventInput{
		AggregateType: "lesson", AggregateID: "lesson-rollback", AssignmentID: "assignment-1",
		Actor: Actor{Role: SystemActor}, EventAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := (Storage{}).Append(app, event); err != ErrTransactionRequired {
		t.Fatalf("append outside transaction: %v", err)
	}
	rollback := errors.New("business mutation failed")
	if err := app.RunInTransaction(func(tx core.App) error {
		if err := (Storage{}).Append(tx, event); err != nil {
			return err
		}
		return rollback
	}); err == nil {
		t.Fatal("failed business mutation committed event")
	}
	rows, err := app.FindAllRecords(BusinessEventsCollection)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rollback left %d events", len(rows))
	}
}

func TestHistoryPageDoesNotDecodeEventsOutsideTheRequestedPage(t *testing.T) {
	app := historyTestApp(t)
	assignmentID := seedHistoryData(t, app)
	collection, err := app.FindCollectionByNameOrId(BusinessEventsCollection)
	if err != nil {
		t.Fatal(err)
	}
	malformed := core.NewRecord(collection)
	malformed.Set(AssignmentField, assignmentID)
	malformed.Set(AggregateTypeField, "lesson")
	malformed.Set(AggregateIDField, "malformed")
	malformed.Set(EventTypeField, "not_an_event")
	malformed.Set(ActorRoleField, string(SystemActor))
	malformed.Set(EventAtField, "2020-01-01T00:00:00Z")
	if err := app.Save(malformed); err != nil {
		t.Fatal(err)
	}
	page, err := QueryPage(app, assignmentID, Viewer{Role: TeacherActor, ID: "teacher-1"}, 1, 1)
	if err != nil || len(page.Items) != 1 || page.TotalItems != 2 {
		t.Fatalf("bounded page decoded later malformed event: %#v, %v", page, err)
	}
}
