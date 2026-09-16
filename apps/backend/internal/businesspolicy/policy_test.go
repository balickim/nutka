package businesspolicy

import (
	"errors"
	"testing"
	"time"
)

func TestCurrentPolicyHasAgreedValues(t *testing.T) {
	policy := Current()
	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
	if policy.Version != CurrentVersion || policy.Currency != CurrencyPLN || policy.AdHocPrice.Minor != 8000 || policy.PackagePrice.Minor != 26000 || policy.RegularLessonPrice.Minor != 5000 {
		t.Fatalf("unexpected prices or version: %#v", policy)
	}
	if policy.LessonDuration != 45*time.Minute || policy.StartGrid != 15*time.Minute || policy.ParticipantBuffer != 5*time.Minute || policy.LearnerBookingMinimum != 24*time.Hour || policy.LearnerChangeCutoff != 24*time.Hour || policy.BookingHorizon != 14*24*time.Hour {
		t.Fatalf("unexpected scheduling values: %#v", policy)
	}
	if policy.PackageTokenCount != 4 || policy.PackageValidityDays != 60 || policy.TeacherCancellationExtensionDays != 7 || policy.ContractMonthlyReschedules != 1 || policy.ContractFreeCancellations != 2 || policy.ContractReplacementDays != 30 || policy.MonthlyPaymentDueDay != 5 || policy.ContractEndMonth != time.June || policy.ContractEndDay != 30 {
		t.Fatalf("unexpected commercial values: %#v", policy)
	}
}

func TestPolicySnapshotIsStableCopy(t *testing.T) {
	policy := Current()
	snapshot := policy.Snapshot()
	policy.AdHocPrice.Minor = 1
	policy.PackageTokenCount = 99
	if snapshot.AdHocPriceMinor != AdHocPriceMinor || snapshot.PackageTokenCount != PackageTokenCount {
		t.Fatalf("snapshot changed with policy copy: %#v", snapshot)
	}
	if Current().AdHocPrice.Minor != AdHocPriceMinor || Current().PackageTokenCount != PackageTokenCount {
		t.Fatal("current policy must remain unchanged")
	}
}

func TestPolicyRejectsInconsistentValues(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Policy)
	}{
		{"duration grid", func(p *Policy) { p.LessonDuration = 46 * time.Minute }},
		{"horizon minimum", func(p *Policy) { p.LearnerBookingMinimum = p.BookingHorizon + time.Hour }},
		{"negative limit", func(p *Policy) { p.PackageValidityDays = 0 }},
		{"invalid end date", func(p *Policy) { p.ContractEndDay = 31 }},
		{"mismatched money", func(p *Policy) { p.PackagePrice.Currency = "EUR" }},
		{"negative money", func(p *Policy) { p.AdHocPrice.Minor = -1 }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			policy := Current()
			test.mutate(&policy)
			if !errors.Is(policy.Validate(), ErrInvalidPolicy) {
				t.Fatalf("expected invalid policy, got %v", policy.Validate())
			}
		})
	}
}
