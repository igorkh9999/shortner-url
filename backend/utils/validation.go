package utils

import (
	"net/http"
	"net/url"
	"strings"
)

// IsValidURL validates if a string is a valid URL
func IsValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// ExtractIP extracts the client IP address from the request
// Handles X-Forwarded-For header for proxied requests
func ExtractIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// ExtractShortCode extracts the short code from the URL path
// Expects path format: /{shortCode}
func ExtractShortCode(path string) string {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")
	// Split by / and take first part
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

