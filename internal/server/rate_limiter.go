// Package server implements a token bucket rate limiter for per-connection
// throttling that protects the hub from abuse.
package server

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu        sync.Mutex
	tokens    float64
	capacity  float64
	rate      float64
	lastCheck time.Time
}

func newRateLimiter(capacity int, interval time.Duration) *rateLimiter {
	if capacity <= 0 {
		capacity = 1
	}
	if interval <= 0 {
		interval = time.Second
	}

	rate := float64(capacity) / interval.Seconds()
	if rate <= 0 {
		rate = float64(capacity)
	}

	return &rateLimiter{
		tokens:    float64(capacity),
		capacity:  float64(capacity),
		rate:      rate,
		lastCheck: time.Now(),
	}
}

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastCheck).Seconds()
	rl.lastCheck = now

	if elapsed > 0 {
		rl.tokens += elapsed * rl.rate
		if rl.tokens > rl.capacity {
			rl.tokens = rl.capacity
		}
	}

	if rl.tokens < 1 {
		return false
	}

	rl.tokens--
	return true
}
