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
// interval, starting the clock at now. Capacity and interval must be positive:
// [resolveConfig] substitutes a default for anything else before a hub — and
// therefore a Client — ever sees it.
func newRateLimiter(capacity int, interval time.Duration, now time.Time) rateLimiter {
	perNano := float64(capacity) / float64(interval)
	if perNano <= 0 {
		perNano = float64(capacity)
	}

	return rateLimiter{
		tokens:   float64(capacity),
		capacity: float64(capacity),
		perNano:  perNano,
		last:     now,
	}
}

// allow consumes a token and reports whether the caller may proceed. The caller
// supplies the current instant, which is what makes refill observable without
// spending wall-clock time; production passes time.Now(). A now that does not
// advance refills nothing, and one that goes backwards is ignored rather than
// draining the bucket.
//
// The clock is a parameter rather than a field on purpose: a limiter is
// embedded by value in every Client and this is a per-message path, so a
// func-valued field would cost an indirect call per message and grow every
// Client.
//
// The zero value permits everything. A limiter only throttles once
// [newRateLimiter] has given it a capacity, so a Client assembled without one
// — as tests do — is unlimited rather than silently blocked.
func (rl *rateLimiter) allow(now time.Time) bool {
	if rl.capacity <= 0 {
		return true
	}

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
