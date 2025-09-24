// Package server normalizes and validates HTTP origins for WebSocket requests
// to enforce configured access control.
package server

import (
	"log"
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
			log.Printf("Ignoring invalid origin in configuration: %q", origin)
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

	normalized := strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
	return normalized, true
}

func isOriginAllowed(r *http.Request) bool {
	originHeader := r.Header.Get("Origin")
	if originHeader == "" {
		return false
	}

	normalizedOrigin, ok := normalizeOrigin(originHeader)
	if !ok {
		return false
	}

	configMu.RLock()
	defer configMu.RUnlock()

	if allowAllOrigins {
		return true
	}

	_, exists := allowedOrigins[normalizedOrigin]
	return exists
}

func checkOrigin(r *http.Request) bool {
	if isOriginAllowed(r) {
		return true
	}

	log.Printf("Blocked WebSocket connection from disallowed origin: %q", r.Header.Get("Origin"))
	return false
}
