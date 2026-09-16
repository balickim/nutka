package ledgerapi

import (
	"errors"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/ledger"
	"github.com/pocketbase/pocketbase/core"
)

func TestPocketBaseServiceRejectsUnownedAssignmentAndContract(t *testing.T) {
	e, teacher := authEvent(t, "teacher")
	assignments := core.NewBaseCollection("teacher_learners")
	assignments.Fields.Add(&core.TextField{Name: "teacher"})
	assignments.Fields.Add(&core.TextField{Name: "learner"})
	if err := e.App.Save(assignments); err != nil {
		t.Fatal(err)
	}
	assignment := core.NewRecord(assignments)
	assignment.Set("teacher", "another-teacher")
	assignment.Set("learner", "learner")
	if err := e.App.Save(assignment); err != nil {
		t.Fatal(err)
	}
	service := NewPocketBaseService(e.App, nil)
	if _, err := service.FinancialHistory(nil, ledger.Actor{Role: ledger.TeacherActor, ID: teacher.Id}, assignment.Id, 1, 20); !errors.Is(err, errOwnership) {
		t.Fatalf("unowned assignment was readable: %v", err)
	}
	contracts := core.NewBaseCollection("regular_contracts")
	contracts.Fields.Add(&core.TextField{Name: "assignment"})
	if err := e.App.Save(contracts); err != nil {
		t.Fatal(err)
	}
	contract := core.NewRecord(contracts)
	contract.Set("assignment", assignment.Id)
	if err := e.App.Save(contract); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ContractMonths(nil, ledger.Actor{Role: ledger.TeacherActor, ID: teacher.Id}, contract.Id, 1, 20); !errors.Is(err, errOwnership) {
		t.Fatalf("unowned contract was readable: %v", err)
	}
}

func TestPocketBaseServiceBoundsTeacherFinancialWork(t *testing.T) {
	e, teacher := authEvent(t, "teacher")
	assignments := core.NewBaseCollection("teacher_learners")
	assignments.Fields.Add(&core.TextField{Name: "teacher"})
	if err := e.App.Save(assignments); err != nil {
		t.Fatal(err)
	}
	assignment := core.NewRecord(assignments)
	assignment.Set("teacher", teacher.Id)
	if err := e.App.Save(assignment); err != nil {
		t.Fatal(err)
	}
	charges := core.NewBaseCollection("charges")
	for _, field := range []string{"assignment", "source_type", "source_id", "period", "currency", "settlement_state", "due_on"} {
		charges.Fields.Add(&core.TextField{Name: field})
	}
	charges.Fields.Add(&core.NumberField{Name: "original_amount_minor"})
	charges.Fields.Add(&core.NumberField{Name: "current_amount_minor"})
	charges.Fields.Add(&core.DateField{Name: "paid_at"})
	if err := e.App.Save(charges); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 101; index++ {
		charge := core.NewRecord(charges)
		charge.Set("assignment", assignment.Id)
		charge.Set("source_type", "ad_hoc")
		charge.Set("source_id", "lesson")
		charge.Set("currency", "PLN")
		charge.Set("settlement_state", "pending_settlement")
		charge.Set("original_amount_minor", 8000)
		charge.Set("current_amount_minor", 8000)
		charge.Set("created_at", time.Now())
		if err := e.App.Save(charge); err != nil {
			t.Fatal(err)
		}
	}
	entries := core.NewBaseCollection("financial_entries")
	for _, field := range []string{"assignment", "entry_type", "currency", "event_at"} {
		entries.Fields.Add(&core.TextField{Name: field})
	}
	entries.Fields.Add(&core.NumberField{Name: "amount_minor"})
	if err := e.App.Save(entries); err != nil {
		t.Fatal(err)
	}
	result, err := NewPocketBaseService(e.App, nil).FinancialWork(nil, ledger.Actor{Role: ledger.TeacherActor, ID: teacher.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Charges) != 100 {
		t.Fatalf("teacher financial work was unbounded: %d", len(result.Charges))
	}
}
