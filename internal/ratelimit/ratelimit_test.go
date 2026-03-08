package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	ratelimit "github.com/onsomlem/cocopilot/internal/ratelimit"
)

func mockClock(start time.Time) (nowFn func() time.Time, advance func(d time.Duration)) {
	mu := sync.Mutex{}
	cur := start
	nowFn = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return cur
	}
	advance = func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		cur = cur.Add(d)
	}
	return
}

func TestRateLimitAllowsWithinLimit(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()
	now, _ := mockClock(time.Now())
	rl.NowFunc = now

	for i := 0; i < 5; i++ {
		if !rl.CheckRateLimit("proj1", "agent1", 5, time.Minute) {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}
}

func TestRateLimitBlocksWhenExceeded(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()
	now, _ := mockClock(time.Now())
	rl.NowFunc = now

	limit := 3
	window := time.Minute

	for i := 0; i < limit; i++ {
		if !rl.CheckRateLimit("proj1", "agent1", limit, window) {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}

	if rl.CheckRateLimit("proj1", "agent1", limit, window) {
		t.Fatal("request should have been rate limited")
	}
}

func TestRateLimitResetClearsState(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()
	now, _ := mockClock(time.Now())
	rl.NowFunc = now

	limit := 2
	window := time.Minute

	for i := 0; i < limit; i++ {
		rl.CheckRateLimit("proj1", "agent1", limit, window)
	}
	if rl.CheckRateLimit("proj1", "agent1", limit, window) {
		t.Fatal("should be rate limited before reset")
	}

	rl.Reset("proj1", "agent1")

	if !rl.CheckRateLimit("proj1", "agent1", limit, window) {
		t.Fatal("request should be allowed after reset")
	}
}

func TestRateLimitWindowExpiration(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()
	now, advance := mockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	rl.NowFunc = now

	limit := 2
	window := time.Minute

	for i := 0; i < limit; i++ {
		rl.CheckRateLimit("proj1", "agent1", limit, window)
	}
	if rl.CheckRateLimit("proj1", "agent1", limit, window) {
		t.Fatal("should be rate limited")
	}

	advance(window + time.Second)

	if !rl.CheckRateLimit("proj1", "agent1", limit, window) {
		t.Fatal("request should be allowed after window expires")
	}
}

func TestRateLimitThreadSafety(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()

	limit := 100
	window := time.Minute
	goroutines := 50
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	allowed := make(chan int, goroutines*requestsPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count := 0
			for r := 0; r < requestsPerGoroutine; r++ {
				if rl.CheckRateLimit("proj1", "agent1", limit, window) {
					count++
				}
			}
			allowed <- count
		}()
	}

	wg.Wait()
	close(allowed)

	total := 0
	for c := range allowed {
		total += c
	}

	if total > limit {
		t.Fatalf("total allowed %d exceeded limit %d", total, limit)
	}
	if total < 1 {
		t.Fatal("at least some requests should have been allowed")
	}
}

func TestRateLimitDifferentKeys(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()
	now, _ := mockClock(time.Now())
	rl.NowFunc = now

	limit := 1
	window := time.Minute

	if !rl.CheckRateLimit("projA", "agent1", limit, window) {
		t.Fatal("projA:agent1 should be allowed")
	}
	if rl.CheckRateLimit("projA", "agent1", limit, window) {
		t.Fatal("projA:agent1 should be limited")
	}

	if !rl.CheckRateLimit("projB", "agent2", limit, window) {
		t.Fatal("projB:agent2 should be allowed (different key)")
	}
}

func TestRateLimitCleanup(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()
	now, advance := mockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	rl.NowFunc = now

	rl.CheckRateLimit("proj1", "agent1", 10, time.Minute)
	rl.CheckRateLimit("proj2", "agent2", 10, time.Minute)

	advance(2 * time.Minute)
	rl.Cleanup(time.Minute)

	// Verify entries are empty by checking CountInWindow returns 0 for both keys.
	if c := rl.CountInWindow("proj1", "agent1", time.Minute); c != 0 {
		t.Fatalf("expected 0 entries for proj1:agent1 after cleanup, got %d", c)
	}
	if c := rl.CountInWindow("proj2", "agent2", time.Minute); c != 0 {
		t.Fatalf("expected 0 entries for proj2:agent2 after cleanup, got %d", c)
	}
}

var _ ratelimit.RateLimiter = (*ratelimit.SlidingWindowRateLimiter)(nil)
