// This file defines package purchase and token lifecycle decisions using teacher-local dates.
package commercial

import (
	"strconv"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
)

type Package struct {
	ID           string
	Assignment   Assignment
	Status       PackageStatus
	PurchasedOn  time.Time
	ValidThrough time.Time
	ClosedAt     time.Time
	Price        Money
	Policy       PolicySnapshot
	TeacherZone  string
	Tokens       []Token
	Events       []PackageEvent
}

type PackageEvent struct {
	ID            string
	Type          string
	TokenID       string
	Reason        string
	CorrectsEvent string
	At            time.Time
}

type PurchaseRequest struct {
	ID           string
	Assignment   Assignment
	PurchaseDate time.Time
	Now          time.Time
	TeacherZone  string
}

func NewPackage(request PurchaseRequest) (Package, error) {
	if !validAssignment(request.Assignment) {
		return Package{}, ErrInvalidAssignment
	}
	if err := businesspolicy.ValidateCurrent(); err != nil {
		return Package{}, ErrInvalidPolicy
	}
	policy := businesspolicy.CurrentSnapshot()
	if !validTimezone(request.TeacherZone) {
		return Package{}, ErrInvalidTimezone
	}
	if request.Now.IsZero() || request.PurchaseDate.IsZero() {
		return Package{}, ErrInvalidPurchaseDate
	}
	today, err := localDate(request.Now, request.TeacherZone)
	if err != nil {
		return Package{}, err
	}
	purchased, err := localDate(request.PurchaseDate, request.TeacherZone)
	if err != nil {
		return Package{}, err
	}
	if purchased.After(today) {
		return Package{}, ErrInvalidPurchaseDate
	}
	tokens := newTokens(request.ID, policy.PackageTokenCount)
	return Package{
		ID:           request.ID,
		Assignment:   request.Assignment,
		Status:       PackageOpen,
		PurchasedOn:  purchased,
		ValidThrough: purchased.AddDate(0, 0, policy.PackageValidityDays-1),
		Price:        Money{Minor: policy.PackagePriceMinor, Currency: policy.Currency},
		Policy:       policy,
		TeacherZone:  request.TeacherZone,
		Tokens:       tokens,
	}, nil
}

func validTimezone(zone string) bool {
	_, err := time.LoadLocation(zone)
	return zone != "" && err == nil
}

func newTokens(packageID string, count int) []Token {
	tokens := make([]Token, count)
	for index := range tokens {
		ordinal := index + 1
		tokens[index] = Token{ID: tokenID(packageID, ordinal), Ordinal: ordinal, State: TokenAvailable}
	}
	return tokens
}

func tokenID(packageID string, ordinal int) string {
	if packageID == "" {
		return "token-" + strconv.Itoa(ordinal)
	}
	return packageID + ":" + strconv.Itoa(ordinal)
}

// Reserve makes the lowest ordinal available token reserved for one future lesson.
func (p *Package) Reserve(lesson LessonReference, now time.Time) (Token, error) {
	if err := p.validateReservation(lesson, now); err != nil {
		return Token{}, err
	}
	index := p.lowestAvailable()
	if index < 0 {
		return Token{}, ErrPackageUnavailable
	}
	p.Tokens[index].State = TokenReserved
	p.Tokens[index].LessonID = lesson.ID
	p.Tokens[index].LessonStart = lesson.StartAt.UTC()
	return p.Tokens[index], nil
}

func (p *Package) validateReservation(lesson LessonReference, now time.Time) error {
	if p == nil || p.Status != PackageOpen {
		return ErrPackageClosed
	}
	if !lesson.BelongsTo(p.Assignment) || lesson.ID == "" || lesson.StartAt.IsZero() {
		return ErrInvalidLesson
	}
	if !p.ValidOn(lesson.StartAt, now) {
		return ErrPackageExpired
	}
	return nil
}

func (p Package) lowestAvailable() int {
	index := -1
	for candidate, token := range p.Tokens {
		if token.Available() && (index < 0 || token.Ordinal < p.Tokens[index].Ordinal) {
			index = candidate
		}
	}
	return index
}

func (p *Package) tokenForLesson(lessonID string) (*Token, error) {
	if lessonID == "" {
		return nil, ErrTokenNotFound
	}
	for index := range p.Tokens {
		if p.Tokens[index].State == TokenReserved && p.Tokens[index].LessonID == lessonID {
			return &p.Tokens[index], nil
		}
	}
	return nil, ErrTokenNotFound
}

func (p *Package) settle(lessonID string) error {
	token, err := p.tokenForLesson(lessonID)
	if err != nil {
		return err
	}
	token.State = TokenUsed
	return nil
}

func (p *Package) Complete(lessonID string) error { return p.settle(lessonID) }

func (p *Package) NoShow(lessonID string) error { return p.settle(lessonID) }

func (p *Package) LateCancellation(lessonID string) error { return p.settle(lessonID) }

// CancelLearner returns a token when requested at least 24 hours before start and settles it later.
func (p *Package) CancelLearner(lessonID string, requestedAt time.Time) error {
	if p == nil || p.Status != PackageOpen {
		return ErrPackageClosed
	}
	token, err := p.tokenForLesson(lessonID)
	if err != nil {
		return err
	}
	if token.LessonStart.Sub(requestedAt) < time.Duration(p.Policy.LearnerChangeCutoffHours)*time.Hour {
		return p.settle(lessonID)
	}
	nowDate, dateErr := localDate(requestedAt, p.TeacherZone)
	if dateErr != nil {
		return dateErr
	}
	if nowDate.After(p.ValidThrough) {
		token.State = TokenExpired
		token.LessonID = ""
		token.LessonStart = time.Time{}
		return nil
	}
	token.State = TokenAvailable
	token.LessonID = ""
	token.LessonStart = time.Time{}
	return nil
}

// Reschedule retains the same token and changes only its lesson link.
func (p *Package) Reschedule(lessonID string, replacement LessonReference, requestedAt time.Time) error {
	if p == nil || p.Status != PackageOpen {
		return ErrPackageClosed
	}
	token, err := p.tokenForLesson(lessonID)
	if err != nil {
		return err
	}
	if !replacement.BelongsTo(p.Assignment) || replacement.ID == "" || replacement.StartAt.IsZero() {
		return ErrInvalidLesson
	}
	if replacement.ID != lessonID {
		return ErrInvalidLesson
	}
	if token.LessonStart.Sub(requestedAt) < time.Duration(p.Policy.LearnerChangeCutoffHours)*time.Hour {
		return ErrLearnerChangeTooLate
	}
	if !p.ValidOn(replacement.StartAt, requestedAt) {
		return ErrReplacementExpired
	}
	token.LessonID = replacement.ID
	token.LessonStart = replacement.StartAt.UTC()
	return nil
}
