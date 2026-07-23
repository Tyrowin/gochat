// Package server implements a token bucket rate limiter for per-connection
// throttling that protects the hub from abuse.
package server

import "time"

// rateLimiter is a token bucket sized for a single connection.
//
// It is deliberately lock-free: each limiter is embedded by value in a Client
// and only touched by that client's read pump, so the bucket needs no
// synchronization. Do not share a limiter across goroutines.
type rateLimiter struct {
	tokens   float64
	capacity float64
	perNano  float64 // tokens replenished per nanosecond
	last     time.Time
}

func newRateLimiter(capacity int, interval time.Duration) rateLimiter {
	if capacity <= 0 {
		capacity = 1
	}
	if interval <= 0 {
		interval = time.Second
	}

	perNano := float64(capacity) / float64(interval)
	if perNano <= 0 {
		perNano = float64(capacity)
	}

	return rateLimiter{
		tokens:   float64(capacity),
		capacity: float64(capacity),
		perNano:  perNano,
		last:     time.Now(),
	}
}

// allow consumes a token and reports whether the caller may proceed.
func (rl *rateLimiter) allow() bool {
	now := time.Now()
	if elapsed := now.Sub(rl.last); elapsed > 0 {
		rl.last = now
		rl.tokens = min(rl.capacity, rl.tokens+float64(elapsed)*rl.perNano)
	}

	if rl.tokens < 1 {
		return false
	}

	rl.tokens--
	return true
}
