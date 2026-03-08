// Rate Limiting Infrastructure - B1.2
package ratelimit

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// RateLimiter interface
// ---------------------------------------------------------------------------

// RateLimiter defines an interface for checking and resetting rate limits.
type RateLimiter interface {
	// CheckRateLimit returns true if the action is allowed, false if rate limited.
	CheckRateLimit(projectID, agentID string, limit int, window time.Duration) bool
	// Reset clears rate limit state for a given key.
	Reset(projectID, agentID string)
}

// ---------------------------------------------------------------------------
// SlidingWindowRateLimiter
// ---------------------------------------------------------------------------

// slidingWindowEntry tracks request timestamps for a single composite key.
type slidingWindowEntry struct {
	timestamps []time.Time
}

// SlidingWindowRateLimiter implements RateLimiter using an in-memory sliding
// window algorithm. It is safe for concurrent use.
type SlidingWindowRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*slidingWindowEntry
	// NowFunc is used for obtaining the current time; overridable in tests.
	NowFunc func() time.Time
}

// NewSlidingWindowRateLimiter creates a new SlidingWindowRateLimiter.
func NewSlidingWindowRateLimiter() *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		entries: make(map[string]*slidingWindowEntry),
		NowFunc: time.Now,
	}
}

// rateLimitKey builds the composite map key from projectID and agentID.
func rateLimitKey(projectID, agentID string) string {
	return projectID + ":" + agentID
}

// CheckRateLimit records the current request and returns true if the number of
// requests within the sliding window has not exceeded limit. If the limit is
// already reached, the request is NOT recorded and false is returned.
func (rl *SlidingWindowRateLimiter) CheckRateLimit(projectID, agentID string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := rateLimitKey(projectID, agentID)
	now := rl.NowFunc()

	e, ok := rl.entries[key]
	if !ok {
		e = &slidingWindowEntry{}
		rl.entries[key] = e
	}

	// Prune timestamps outside the window.
	cutoff := now.Add(-window)
	pruned := e.timestamps[:0]
	for _, t := range e.timestamps {
		if !t.Before(cutoff) {
			pruned = append(pruned, t)
		}
	}
	e.timestamps = pruned

	// Check against limit.
	if len(pruned) >= limit {
		return false
	}

	// Record the new request timestamp.
	e.timestamps = append(e.timestamps, now)
	return true
}

// Reset clears all recorded timestamps for the given projectID and agentID.
func (rl *SlidingWindowRateLimiter) Reset(projectID, agentID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.entries, rateLimitKey(projectID, agentID))
}

// CountInWindow returns the number of requests recorded within the given
// window for the specified projectID and agentID, without recording a new
// request. It is safe for concurrent use.
func (rl *SlidingWindowRateLimiter) CountInWindow(projectID, agentID string, window time.Duration) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := rateLimitKey(projectID, agentID)
	now := rl.NowFunc()

	e, ok := rl.entries[key]
	if !ok {
		return 0
	}

	cutoff := now.Add(-window)
	count := 0
	for _, t := range e.timestamps {
		if !t.Before(cutoff) {
			count++
		}
	}
	return count
}

// Cleanup removes all entries whose timestamps have all expired relative to
// now. This can be called periodically to reclaim memory.
func (rl *SlidingWindowRateLimiter) Cleanup(window time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.NowFunc()
	cutoff := now.Add(-window)

	for key, e := range rl.entries {
		pruned := e.timestamps[:0]
		for _, t := range e.timestamps {
			if !t.Before(cutoff) {
				pruned = append(pruned, t)
			}
		}
		if len(pruned) == 0 {
			delete(rl.entries, key)
		} else {
			e.timestamps = pruned
		}
	}
}
