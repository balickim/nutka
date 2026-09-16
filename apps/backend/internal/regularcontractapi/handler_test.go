package regularcontractapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
	"github.com/balickim/nutka/apps/backend/internal/commercial"
	"github.com/balickim/nutka/apps/backend/internal/domain"
	"github.com/balickim/nutka/apps/backend/internal/regularcontract"
	"github.com/balickim/nutka/apps/backend/internal/scheduling"
)

type fakeStore struct {
	assignment      Assignment
	contracts       []regularcontract.RegularContract
	converted       []commercial.LessonReference
	convertErr      error
	activation      ActivationContext
	contractContext ContractContext
	saveErr         error
}

func (s *fakeStore) Assignment(context.Context, string) (Assignment, error) { return s.assignment, nil }
func (s *fakeStore) Contracts(context.Context, string) ([]regularcontract.RegularContract, error) {
	return s.contracts, nil
}
func (s *fakeStore) Contract(_ context.Context, id string) (regularcontract.RegularContract, error) {
	for _, c := range s.contracts {
		if c.ID == id {
			return c, nil
		}
	}
	return regularcontract.RegularContract{}, errNotFound
}
func (s *fakeStore) RunInTransaction(ctx context.Context, fn func(Transaction) error) error {
	beforeContracts, beforeConverted := append([]regularcontract.RegularContract(nil), s.contracts...), append([]commercial.LessonReference(nil), s.converted...)
	err := fn(s)
	if err != nil {
		s.contracts, s.converted = beforeContracts, beforeConverted
	}
	return err
}
func (s *fakeStore) NewContractID(context.Context) (string, error) {
	return "generated-contract-id", nil
}
func (s *fakeStore) ActivationContext(context.Context, string) (ActivationContext, error) {
	if s.activation.Assignment.ID != "" {
		return s.activation, nil
	}
	return ActivationContext{Assignment: s.assignment, TeacherTimezone: "Europe/Warsaw", Policy: businesspolicy.Current(), CommercialState: commercial.OverlapState{Assignment: commercial.Assignment{ID: s.assignment.ID, TeacherID: s.assignment.TeacherID, LearnerID: s.assignment.LearnerID}}}, nil
}
func (s *fakeStore) ContractContext(_ context.Context, id string) (ContractContext, error) {
	if s.contractContext.Contract.ID == id {
		return s.contractContext, nil
	}
	c, err := s.Contract(context.Background(), id)
	return ContractContext{Contract: c, Policy: businesspolicy.Current()}, err
}
func (s *fakeStore) SaveContract(_ context.Context, c regularcontract.RegularContract) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.contracts = []regularcontract.RegularContract{c}
	return nil
}
func (s *fakeStore) ConvertContractLessons(_ context.Context, _ string, lessons []commercial.LessonReference) error {
	if s.convertErr != nil {
		return s.convertErr
	}
	s.converted = append(s.converted, lessons...)
	return nil
}

func testContract() regularcontract.RegularContract {
	zone, _ := time.LoadLocation("Europe/Warsaw")
	start := time.Date(2030, 8, 1, 0, 0, 0, 0, zone)
	end := time.Date(2031, 6, 30, 0, 0, 0, 0, zone)
	return regularcontract.RegularContract{ID: "contract-1", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", TeacherTimezone: "Europe/Warsaw", StartOn: start, EndOn: end, EffectiveEndOn: end, Status: regularcontract.Active, PriceMinor: 5000, Currency: "PLN", Policy: businesspolicy.Current(), PolicySnapshot: businesspolicy.CurrentSnapshot()}
}

func request(method, path string, body string, p Principal) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	return r.WithContext(WithPrincipal(r.Context(), p))
}

func TestContractReadsRejectUnrelatedParticipant(t *testing.T) {
	store := &fakeStore{assignment: Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, contracts: []regularcontract.RegularContract{testContract()}}
	h := New(store, func() time.Time { return time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC) })
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request(http.MethodGet, "/api/teachers/contracts/contract-1/series", "", Principal{Role: domain.TeacherActor, ID: "teacher-2"}))
	if response.Code != http.StatusForbidden {
		t.Fatalf("unrelated teacher status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "contract-1") {
		t.Fatal("unauthorized response exposed contract identity")
	}
}

func TestContractReadsRejectWrongRealmAndAllowAssignedLearnerHorizonView(t *testing.T) {
	store := &fakeStore{assignment: Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, contracts: []regularcontract.RegularContract{testContract()}}
	h := New(store, func() time.Time { return time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC) })
	wrong := httptest.NewRecorder()
	h.ServeHTTP(wrong, request(http.MethodGet, "/api/learners/contracts/contract-1/series", "", Principal{Role: domain.TeacherActor, ID: "teacher-1"}))
	if wrong.Code != http.StatusNotFound {
		t.Fatalf("wrong realm status = %d", wrong.Code)
	}
	allowed := httptest.NewRecorder()
	h.ServeHTTP(allowed, request(http.MethodGet, "/api/learners/assignments/assignment-1/contracts", "", Principal{Role: domain.LearnerActor, ID: "learner-1"}))
	if allowed.Code != http.StatusOK {
		t.Fatalf("assigned learner status = %d body=%s", allowed.Code, allowed.Body.String())
	}
}

func TestContractMutationRequiresIntentAndParticipant(t *testing.T) {
	store := &fakeStore{assignment: Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, contracts: []regularcontract.RegularContract{testContract()}}
	h := New(store, func() time.Time { return time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC) })
	r := request(http.MethodPost, "/api/teachers/contracts/contract-1/notice", `{}`,
		Principal{Role: domain.TeacherActor, ID: "teacher-2"})
	withoutIntent := httptest.NewRecorder()
	h.ServeHTTP(withoutIntent, r)
	if withoutIntent.Code != http.StatusForbidden {
		t.Fatalf("missing intent status = %d", withoutIntent.Code)
	}
	r.Header.Set("X-Requested-With", "fetch")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, r)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unrelated teacher mutation status = %d", response.Code)
	}
}

func broadAvailability() []scheduling.Interval {
	interval, err := scheduling.NewInterval(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	return []scheduling.Interval{interval}
}

func TestActivationConvertsSelectedAdHocLessonsAtomically(t *testing.T) {
	now := time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{assignment: Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}}
	store.activation = ActivationContext{Assignment: store.assignment, TeacherTimezone: "Europe/Warsaw", Policy: businesspolicy.Current(), Availability: broadAvailability(), CommercialState: commercial.OverlapState{
		Assignment: commercial.Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, Now: now,
		FutureAdHocLessons: []commercial.LessonReference{{ID: "ad-hoc-1", AssignmentID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1", Plan: commercial.AdHoc, StartAt: now.Add(24 * time.Hour)}},
	}}
	h := New(store, func() time.Time { return now })
	r := request(http.MethodPost, "/api/teachers/assignments/assignment-1/contracts", `{"start_on":"2030-09-16","weekday":1,"start_time":"17:00","convert_lesson_ids":["ad-hoc-1"]}`, Principal{Role: domain.TeacherActor, ID: "teacher-1"})
	r.Header.Set("X-Requested-With", "fetch")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, r)
	if response.Code != http.StatusCreated || len(store.converted) != 1 || store.converted[0].ID != "ad-hoc-1" {
		t.Fatalf("activation conversion failed: status=%d body=%s converted=%#v", response.Code, response.Body.String(), store.converted)
	}
	if len(store.contracts) != 1 {
		t.Fatalf("activation did not persist contract: %d", len(store.contracts))
	}

	store.convertErr = errInvalidRequest
	r = request(http.MethodPost, "/api/teachers/assignments/assignment-1/contracts", `{"start_on":"2030-09-16","weekday":1,"start_time":"17:00","convert_lesson_ids":["ad-hoc-1"]}`, Principal{Role: domain.TeacherActor, ID: "teacher-1"})
	r.Header.Set("X-Requested-With", "fetch")
	response = httptest.NewRecorder()
	h.ServeHTTP(response, r)
	if response.Code != http.StatusBadRequest || len(store.converted) != 1 || len(store.contracts) != 1 {
		t.Fatalf("conversion rollback failed: status=%d converted=%d contracts=%d", response.Code, len(store.converted), len(store.contracts))
	}
}

func TestContractEndpointsRejectWrongRolesAndUnknownFields(t *testing.T) {
	store := &fakeStore{assignment: Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, contracts: []regularcontract.RegularContract{testContract()}}
	h := New(store, func() time.Time { return time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC) })
	cases := []struct {
		path, body string
		role       domain.ActorRole
	}{
		{"/api/teachers/assignments/assignment-1/contracts", `{"start_on":"2030-09-16","weekday":1,"start_time":"17:00","unexpected":true}`, domain.TeacherActor},
		{"/api/teachers/contracts/contract-1/schedule", `{"weekday":1,"start_time":"17:00","effective_on":"2030-09-16"}`, domain.LearnerActor},
		{"/api/teachers/contracts/contract-1/amendments", `{"effective_on":"2030-11-01","price_minor":6000,"currency":"PLN"}`, domain.LearnerActor},
		{"/api/teachers/contracts/contract-1/early-end", `{"end_on":"2030-09-20","reason":"mutual"}`, domain.LearnerActor},
		{"/api/teachers/contracts/contract-1/correction", `{"event_id":"event-1","reason":"repair"}`, domain.LearnerActor},
	}
	for _, test := range cases {
		r := request(http.MethodPost, test.path, test.body, Principal{Role: test.role, ID: map[domain.ActorRole]string{domain.TeacherActor: "teacher-1", domain.LearnerActor: "learner-1"}[test.role]})
		r.Header.Set("X-Requested-With", "fetch")
		response := httptest.NewRecorder()
		h.ServeHTTP(response, r)
		want := http.StatusForbidden
		if strings.Contains(test.body, "unexpected") {
			want = http.StatusBadRequest
		}
		if response.Code != want {
			t.Errorf("%s role=%s status=%d want=%d body=%s", test.path, test.role, response.Code, want, response.Body.String())
		}
	}
}

func TestNoticeIsEmptyAndLifecycleCommandsUseDistinctRoutes(t *testing.T) {
	now := time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{assignment: Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, contracts: []regularcontract.RegularContract{testContract()}}
	h := New(store, func() time.Time { return now })
	r := request(http.MethodPost, "/api/teachers/contracts/contract-1/notice", `{"reason":"hidden"}`, Principal{Role: domain.TeacherActor, ID: "teacher-1"})
	r.Header.Set("X-Requested-With", "fetch")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, r)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("notice accepted early-end fields: %d", response.Code)
	}
	store.contracts = []regularcontract.RegularContract{testContract()}
	r = request(http.MethodPost, "/api/teachers/contracts/contract-1/notice", `{}`, Principal{Role: domain.TeacherActor, ID: "teacher-1"})
	r.Header.Set("X-Requested-With", "fetch")
	response = httptest.NewRecorder()
	h.ServeHTTP(response, r)
	if response.Code != http.StatusOK {
		t.Fatalf("notice failed: %d %s", response.Code, response.Body.String())
	}
}

func TestTeacherLifecycleEndpointsPersistDomainTransitions(t *testing.T) {
	now := time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC)
	base := testContract()
	base.EndOn = time.Date(2030, 8, 31, 0, 0, 0, 0, time.FixedZone("UTC", 0))
	base.EffectiveEndOn = base.EndOn
	base.Status = regularcontract.Ended
	store := &fakeStore{assignment: Assignment{ID: "assignment-1", TeacherID: "teacher-1", LearnerID: "learner-1"}, contracts: []regularcontract.RegularContract{base}}
	store.contractContext = ContractContext{Contract: base, Policy: businesspolicy.Current(), Availability: broadAvailability()}
	h := New(store, func() time.Time { return now })

	checks := []struct {
		path, body string
		status     int
	}{
		{"/api/teachers/contracts/contract-1/schedule", `{"weekday":2,"start_time":"18:00","effective_on":"2030-09-10"}`, http.StatusOK},
		{"/api/teachers/contracts/contract-1/amendments", `{"effective_on":"2030-11-01","price_minor":6000,"currency":"PLN"}`, http.StatusOK},
		{"/api/teachers/contracts/contract-1/early-end", `{"end_on":"2030-08-20","reason":"mutual agreement"}`, http.StatusBadRequest},
		{"/api/teachers/contracts/contract-1/correction", `{"event_id":"missing","reason":"repair"}`, http.StatusBadRequest},
	}
	for _, check := range checks {
		r := request(http.MethodPost, check.path, check.body, Principal{Role: domain.TeacherActor, ID: "teacher-1"})
		r.Header.Set("X-Requested-With", "fetch")
		response := httptest.NewRecorder()
		h.ServeHTTP(response, r)
		if response.Code != check.status {
			t.Errorf("%s status=%d want=%d body=%s", check.path, response.Code, check.status, response.Body.String())
		}
	}

	renew := request(http.MethodPost, "/api/teachers/contracts/contract-1/renew", `{"start_on":"2030-09-16"}`, Principal{Role: domain.TeacherActor, ID: "teacher-1"})
	renew.Header.Set("X-Requested-With", "fetch")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, renew)
	if response.Code != http.StatusCreated {
		t.Fatalf("renewal status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCorrectionEndpointAppendsCompensatingEvent(t *testing.T) {
	now := time.Date(2030, 9, 1, 12, 0, 0, 0, time.UTC)
	contract := testContract()
	interval, _ := scheduling.NewInterval(now.Add(48*time.Hour), now.Add(48*time.Hour+45*time.Minute))
	contract.Occurrences = []regularcontract.Occurrence{{ID: "occurrence-1", ContractID: contract.ID, AssignmentID: contract.AssignmentID, TeacherID: contract.TeacherID, LearnerID: contract.LearnerID, OriginalLocalDate: "2030-09-03", Interval: interval, ScheduleState: regularcontract.Cancelled, BillingOutcome: regularcontract.FreeLearnerCancel}}
	contract.Events = []regularcontract.Event{{ID: "event-1", Type: regularcontract.EventLearnerFreeCancel, ContractID: contract.ID, OccurrenceID: "occurrence-1", OriginalMonth: "2030-09", Actor: regularcontract.Actor{Role: regularcontract.Learner, ID: contract.LearnerID}, At: now}}
	store := &fakeStore{assignment: Assignment{ID: contract.AssignmentID, TeacherID: contract.TeacherID, LearnerID: contract.LearnerID}, contracts: []regularcontract.RegularContract{contract}}
	store.contractContext = ContractContext{Contract: contract, Policy: businesspolicy.Current()}
	h := New(store, func() time.Time { return now })
	r := request(http.MethodPost, "/api/teachers/contracts/contract-1/correction", `{"event_id":"event-1","reason":"restore allowance"}`, Principal{Role: domain.TeacherActor, ID: "teacher-1"})
	r.Header.Set("X-Requested-With", "fetch")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, r)
	if response.Code != http.StatusOK {
		t.Fatalf("correction status=%d body=%s", response.Code, response.Body.String())
	}
	if len(store.contracts) != 1 || len(store.contracts[0].Events) != 2 || store.contracts[0].Events[1].CorrectsEvent != "event-1" {
		t.Fatalf("correction was not persisted: %#v", store.contracts)
	}
}
