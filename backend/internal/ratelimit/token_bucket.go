package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	rate       float64    // tokens per second
	burst      int        // max tokens
	available  float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{rate: rate, burst: burst, available: float64(burst), lastRefill: time.Now()}
}

func (tb *TokenBucket) refillLocked(now time.Time) {
	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed <= 0 { return }
	tb.available += elapsed * tb.rate
	if tb.available > float64(tb.burst) {
		tb.available = float64(tb.burst)
	}
	tb.lastRefill = now
}

// Allow consumes n tokens if available and returns true, otherwise false.
func (tb *TokenBucket) Allow(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refillLocked(time.Now())
	if tb.available >= float64(n) {
		tb.available -= float64(n)
		return true
	}
	return false
}

// Wait blocks until n tokens are available or ctx is done.
func (tb *TokenBucket) Wait(n int) {
	for {
		if tb.Allow(n) { return }
		time.Sleep(10 * time.Millisecond)
	}
}
