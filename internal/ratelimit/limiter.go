package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter tracks failed login attempts and blocks accounts that exceed the threshold.
type RateLimiter interface {
	Allow(username string) (allowed bool, retryAfter time.Duration)
	RecordFailure(username string)
	RecordSuccess(username string)
}

// loginAttempt holds per-user attempt state.
type loginAttempt struct {
	failureCount int
	firstAttempt time.Time
	blockedUntil time.Time
}

type LoginAttemptTracker struct {
	attempts      sync.Map
	maxAttempts   int
	timeWindow    time.Duration
	blockDuration time.Duration
}

func NewLoginAttemptTracker(maxAttempts int, timeWindow, blockDuration time.Duration) *LoginAttemptTracker {
	return &LoginAttemptTracker{
		maxAttempts:   maxAttempts,
		timeWindow:    timeWindow,
		blockDuration: blockDuration,
	}
}

// Allow returns whether a login attempt is permitted. If the account is blocked,
// it returns false along with the remaining wait duration.
func (t *LoginAttemptTracker) Allow(username string) (bool, time.Duration) {
	now := time.Now()

	val, _ := t.attempts.LoadOrStore(username, &loginAttempt{firstAttempt: now})
	a := val.(*loginAttempt)

	if !a.blockedUntil.IsZero() {
		if now.Before(a.blockedUntil) {
			return false, a.blockedUntil.Sub(now)
		}
		a.failureCount = 0
		a.firstAttempt = now
		a.blockedUntil = time.Time{}
	}

	if now.Sub(a.firstAttempt) > t.timeWindow {
		a.failureCount = 0
		a.firstAttempt = now
	}

	if a.failureCount >= t.maxAttempts {
		a.blockedUntil = now.Add(t.blockDuration)
		return false, t.blockDuration
	}

	return true, 0
}

func (t *LoginAttemptTracker) RecordFailure(username string) {
	now := time.Now()

	val, _ := t.attempts.LoadOrStore(username, &loginAttempt{firstAttempt: now})
	a := val.(*loginAttempt)

	if a.failureCount == 0 || now.Sub(a.firstAttempt) > t.timeWindow {
		a.firstAttempt = now
		a.failureCount = 1
	} else {
		a.failureCount++
	}

	if a.failureCount >= t.maxAttempts {
		a.blockedUntil = now.Add(t.blockDuration)
	}
}

// RecordSuccess clears the attempt record so the counter resets on next login.
func (t *LoginAttemptTracker) RecordSuccess(username string) {
	t.attempts.Delete(username)
}
