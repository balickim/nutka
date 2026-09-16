// This file exposes package snapshots, token balances, validity, and expiry reconciliation.
package commercial

import "time"

func (p Package) Clone() Package {
	p.Tokens = cloneTokens(p.Tokens)
	p.Events = append([]PackageEvent(nil), p.Events...)
	return p
}

func (p Package) AvailableTokens() []Token {
	available := make([]Token, 0)
	for _, token := range cloneTokens(p.Tokens) {
		if token.Available() {
			available = append(available, token)
		}
	}
	return available
}

func (p Package) AvailableCount() int { return len(p.AvailableTokens()) }

func (p Package) ValidOn(start, now time.Time) bool {
	if p.Status != PackageOpen || start.IsZero() || now.IsZero() {
		return false
	}
	startDate, startErr := localDate(start, p.TeacherZone)
	nowDate, nowErr := localDate(now, p.TeacherZone)
	if startErr != nil || nowErr != nil || nowDate.After(p.ValidThrough) {
		return false
	}
	return !startDate.Before(p.PurchasedOn) && !startDate.After(p.ValidThrough)
}

// ReconcileExpiry expires only available tokens after the final local validity date.
func (p *Package) ReconcileExpiry(now time.Time) error {
	if p == nil || p.Status != PackageOpen {
		return ErrPackageClosed
	}
	nowDate, err := localDate(now, p.TeacherZone)
	if err != nil {
		return err
	}
	if !nowDate.After(p.ValidThrough) {
		return nil
	}
	for index := range p.Tokens {
		if p.Tokens[index].State == TokenAvailable {
			p.Tokens[index].State = TokenExpired
		}
	}
	return nil
}
