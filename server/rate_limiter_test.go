package server

import (
	"sync"
	"time"
)

// mockClock returns a controllable clock for tests.
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
