package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter tracks failed login attempts and blocks accounts that exceed the threshold.
// This interface provides brute force protection by limiting authentication attempts.
type RateLimiter interface {
	// Allow checks if a login attempt is permitted for the given username.
	// Returns false if the account is blocked, along with the remaining block duration.
	Allow(username string) (allowed bool, retryAfter time.Duration)

	// RecordFailure increments the failure count for the username.
	// If the count exceeds the threshold, the account is blocked.
	RecordFailure(username string)

	// RecordSuccess clears all failure records for the username.
	// This resets the counter after a successful login.
	RecordSuccess(username string)
}

// loginAttempt holds per-user attempt state for rate limiting.
// Each user has their own loginAttempt record tracked independently.
type loginAttempt struct {
	failureCount int       // Number of failed attempts in current time window
	firstAttempt time.Time // Timestamp of first failure in current window
	blockedUntil time.Time // Timestamp when block expires (zero if not blocked)
}

// LoginAttemptTracker implements RateLimiter using in-memory storage.
type LoginAttemptTracker struct {
	attempts      sync.Map      // Map[string]*loginAttempt - thread-safe storage
	maxAttempts   int           // Failure threshold before blocking
	timeWindow    time.Duration // Window for counting failures
	blockDuration time.Duration // Duration to block after threshold
}

// NewLoginAttemptTracker creates a new rate limiter with the specified configuration.
func NewLoginAttemptTracker(maxAttempts int, timeWindow, blockDuration time.Duration) *LoginAttemptTracker {
	return &LoginAttemptTracker{
		maxAttempts:   maxAttempts,
		timeWindow:    timeWindow,
		blockDuration: blockDuration,
	}
}

// Allow checks if a login attempt is permitted for the given username.
// This method implements the core rate limiting logic with several checks:
func (t *LoginAttemptTracker) Allow(username string) (bool, time.Duration) {
	now := time.Now()

	// Load existing attempt record or create new one
	// LoadOrStore is atomic - ensures thread safety
	val, _ := t.attempts.LoadOrStore(username, &loginAttempt{firstAttempt: now})
	a := val.(*loginAttempt)

	// Check 1: Is account currently blocked?
	if !a.blockedUntil.IsZero() {
		if now.Before(a.blockedUntil) {
			// Still blocked - return remaining wait time
			return false, a.blockedUntil.Sub(now)
		}
		// Block expired - reset state and allow attempt
		a.failureCount = 0
		a.firstAttempt = now
		a.blockedUntil = time.Time{} // Clear block timestamp
	}

	// Check 2: Has the time window expired?
	// If more than timeWindow has passed since first attempt, reset counter
	if now.Sub(a.firstAttempt) > t.timeWindow {
		a.failureCount = 0
		a.firstAttempt = now
	}

	// Check 3: Has threshold been reached?
	// If user already has maxAttempts failures, block them now
	if a.failureCount >= t.maxAttempts {
		a.blockedUntil = now.Add(t.blockDuration)
		return false, t.blockDuration
	}

	// All checks passed - allow the attempt
	return true, 0
}

// RecordFailure increments the failure count for the username.
// Should be called after each failed authentication attempt.
func (t *LoginAttemptTracker) RecordFailure(username string) {
	now := time.Now()

	// Load existing attempt record or create new one
	val, _ := t.attempts.LoadOrStore(username, &loginAttempt{firstAttempt: now})
	a := val.(*loginAttempt)

	// Check if this is first failure or time window has expired
	if a.failureCount == 0 || now.Sub(a.firstAttempt) > t.timeWindow {
		// Start new tracking window
		a.firstAttempt = now
		a.failureCount = 1
	} else {
		// Increment failure count within current window
		a.failureCount++
	}

	// Check if threshold reached - if so, block the account
	if a.failureCount >= t.maxAttempts {
		a.blockedUntil = now.Add(t.blockDuration)
	}
}

// RecordSuccess clears all failure records for the username.
// Should be called after successful authentication to reset the counter.

func (t *LoginAttemptTracker) RecordSuccess(username string) {
	// Delete the entire record - user starts fresh next time
	t.attempts.Delete(username)
}
