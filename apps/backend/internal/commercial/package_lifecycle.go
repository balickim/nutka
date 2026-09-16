// This file defines package cancellation, closure, renewal, and correction decisions.
package commercial

import "time"

// TeacherCancel returns one token and appends one seven-local-day extension.
func (p *Package) TeacherCancel(lessonID string, cancelledAt time.Time) error {
	if p == nil || p.Status != PackageOpen {
		return ErrPackageClosed
	}
	token, err := p.tokenForLesson(lessonID)
	if err != nil {
		return err
	}
	token.State = TokenAvailable
	token.LessonID = ""
	token.LessonStart = time.Time{}
	p.ValidThrough = p.ValidThrough.AddDate(0, 0, p.Policy.TeacherCancellationExtensionDays)
	return nil
}

type RefundInfo struct {
	Amount Money
	Note   string
}

type CloseDecision struct {
	Invalidated int
	Refund      *RefundInfo
	Reason      string
}

func (p *Package) Close(now time.Time, reason string, refund *RefundInfo) (CloseDecision, error) {
	if err := p.validateClosure(now, reason, refund); err != nil {
		return CloseDecision{}, err
	}
	for _, token := range p.Tokens {
		if token.State == TokenReserved && token.LessonStart.After(now) {
			return CloseDecision{}, ErrFuturePackageLesson
		}
	}
	invalidated := 0
	for index := range p.Tokens {
		if p.Tokens[index].State == TokenAvailable {
			p.Tokens[index].State = TokenInvalidated
			invalidated++
		}
	}
	p.Status = PackageClosed
	p.ClosedAt = now.UTC()
	return CloseDecision{Invalidated: invalidated, Refund: refund, Reason: reason}, nil
}

func (p *Package) validateClosure(now time.Time, reason string, refund *RefundInfo) error {
	if p == nil || p.Status != PackageOpen {
		return ErrPackageClosed
	}
	if now.IsZero() {
		return ErrInvalidLesson
	}
	if !validReason(reason) {
		return ErrCloseReasonRequired
	}
	if refund != nil && (refund.Amount.Validate() != nil || refund.Amount.Currency != p.Price.Currency) {
		return ErrInvalidRefund
	}
	return nil
}

func (p Package) CanRenew() bool { return p.AvailableCount() == 0 }

func ValidateRenewal(previous Package) error {
	if !previous.CanRenew() {
		return ErrPackageRenewal
	}
	return nil
}

type CorrectionRequest struct {
	TokenID       string
	Target        TokenState
	CorrectsEvent string
	Reason        string
	At            time.Time
}

// CorrectTokenState appends a compensating event while preserving the original token history.
func (p *Package) CorrectTokenState(request CorrectionRequest) (PackageEvent, error) {
	if err := validateCorrection(p, request); err != nil {
		return PackageEvent{}, err
	}
	for index := range p.Tokens {
		if p.Tokens[index].ID != request.TokenID {
			continue
		}
		if request.Target == TokenReserved && p.Tokens[index].LessonID == "" {
			return PackageEvent{}, ErrInvalidLesson
		}
		if request.Target == TokenAvailable || request.Target == TokenExpired || request.Target == TokenInvalidated {
			p.Tokens[index].LessonID = ""
			p.Tokens[index].LessonStart = time.Time{}
		}
		p.Tokens[index].State = request.Target
		event := PackageEvent{ID: request.CorrectsEvent + ":correction", Type: "package_token_corrected", TokenID: request.TokenID, Reason: request.Reason, CorrectsEvent: request.CorrectsEvent, At: request.At.UTC()}
		p.Events = append(p.Events, event)
		return event, nil
	}
	return PackageEvent{}, ErrTokenNotFound
}

func validateCorrection(p *Package, request CorrectionRequest) error {
	if p == nil || request.TokenID == "" {
		return ErrTokenNotFound
	}
	if !validReason(request.Reason) {
		return ErrCorrectionReason
	}
	if request.CorrectsEvent == "" || request.At.IsZero() {
		return ErrCorrectionEvent
	}
	if !validTokenState(request.Target) {
		return ErrInvalidTokenState
	}
	return nil
}

func validTokenState(state TokenState) bool {
	switch state {
	case TokenAvailable, TokenReserved, TokenUsed, TokenExpired, TokenInvalidated:
		return true
	default:
		return false
	}
}
