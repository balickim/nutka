// This file adapts the global business policy to pure interval validation without persistence or HTTP dependencies.
package scheduling

import (
	"errors"
	"time"

	"github.com/balickim/nutka/apps/backend/internal/businesspolicy"
)

var ErrInvalidIntervalPolicy = errors.New("scheduling interval policy is invalid")

// IntervalPolicy contains the configurable values used by slot and conflict rules.
type IntervalPolicy struct {
	Grid           time.Duration
	Duration       time.Duration
	Buffer         time.Duration
	BookingMinimum time.Duration
	Horizon        time.Duration
}

func DefaultIntervalPolicy() IntervalPolicy {
	return IntervalPolicyFromBusinessPolicy(businesspolicy.Current())
}

func IntervalPolicyFromBusinessPolicy(policy businesspolicy.Policy) IntervalPolicy {
	return IntervalPolicy{Grid: policy.StartGrid, Duration: policy.LessonDuration, Buffer: policy.ParticipantBuffer, BookingMinimum: policy.LearnerBookingMinimum, Horizon: policy.BookingHorizon}
}

func (p IntervalPolicy) Validate() error {
	if p.Grid <= 0 || p.Duration <= 0 || p.Buffer < 0 || p.BookingMinimum < 0 || p.Horizon <= 0 || p.Duration%p.Grid != 0 || p.BookingMinimum > p.Horizon {
		return ErrInvalidIntervalPolicy
	}
	return nil
}

func resolveIntervalPolicy(options ...IntervalPolicy) (IntervalPolicy, error) {
	policy := DefaultIntervalPolicy()
	for _, option := range options {
		policy = option
	}
	if err := policy.Validate(); err != nil {
		return IntervalPolicy{}, err
	}
	return policy, nil
}

func (p IntervalPolicy) HorizonEnd(now time.Time) time.Time { return now.UTC().Add(p.Horizon) }

func IsOnGrid(t time.Time, grid time.Duration) bool { return isOnGrid(t, grid) }

func HorizonEndWithPolicy(now time.Time, options ...IntervalPolicy) (time.Time, error) {
	policy, err := resolveIntervalPolicy(options...)
	if err != nil {
		return time.Time{}, err
	}
	return policy.HorizonEnd(now), nil
}

func isOnGrid(t time.Time, grid time.Duration) bool {
	if t.IsZero() || grid <= 0 {
		return false
	}
	u := t.UTC()
	return u.Equal(u.Truncate(grid))
}
