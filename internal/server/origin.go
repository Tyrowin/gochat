// Package server normalizes and validates HTTP origins for WebSocket requests
// to enforce configured access control.
package server

import (
	"net/http"
	"net/url"
	"strings"
)

func normalizeOrigins(origins []string) ([]string, bool) {
	if len(origins) == 0 {
		return nil, false
	}

	normalized := make([]string, 0, len(origins))
	allowAll := false

	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}

		if trimmed == "*" {
			allowAll = true
			continue
		}

		normalizedOrigin, ok := normalizeOrigin(trimmed)
		if !ok {
			log().Warn("ignoring invalid origin in configuration", "origin", origin)
			continue
		}

		normalized = append(normalized, normalizedOrigin)
	}

	return normalized, allowAll
}

func normalizeOrigin(origin string) (string, bool) {
	parsed, err := url.Parse(origin)
	if err != nil {
		return "", false
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), true
}

func isOriginAllowed(r *http.Request) bool {
	// A missing Origin header is rejected even when the allow-list contains "*",
	// so non-browser clients cannot bypass the check by omitting it.
	originHeader := r.Header.Get("Origin")
	if originHeader == "" {
		return false
	}

	snap := currentSnapshot()

	// Fast path: browsers send the already-canonical form, so the common case
	// matches the allow-list without parsing a URL.
	if _, exists := snap.origins[originHeader]; exists {
		return true
	}

	normalizedOrigin, ok := normalizeOrigin(originHeader)
	if !ok {
		return false
	}

	if snap.allowAll {
		return true
	}

	_, exists := snap.origins[normalizedOrigin]
	return exists
}

func checkOrigin(r *http.Request) bool {
	if isOriginAllowed(r) {
		return true
	}

	log().Warn("blocked WebSocket connection from disallowed origin", "remote_addr", r.RemoteAddr)
	return false
}
