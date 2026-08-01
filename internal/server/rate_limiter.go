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

// newRateLimiter builds a full bucket of capacity tokens that refills over
// interval. Both must be positive: [resolveConfig] substitutes a default for
// anything else before a hub — and therefore a Client — ever sees it.
func newRateLimiter(capacity int, interval time.Duration) rateLimiter {
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
//
// The zero value permits everything. A limiter only throttles once
// [newRateLimiter] has given it a capacity, so a Client assembled without one
// — as tests do — is unlimited rather than silently blocked.
func (rl *rateLimiter) allow() bool {
	if rl.capacity <= 0 {
		return true
	}

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
