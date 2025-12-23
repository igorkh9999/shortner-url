package middleware

import (
	"net/http"
	"os"
)

// CORS middleware handles Cross-Origin Resource Sharing
// Allowed origin is configurable via FRONTEND_URL environment variable
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Get allowed origin from environment, default to localhost:3000 for development
		allowedOrigin := os.Getenv("FRONTEND_URL")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000"
		}
		
		// Allow requests from configured frontend URL
		if origin == allowedOrigin || origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		} else {
			// For other origins in development, echo back the origin
			// In production, you may want to reject unknown origins
			if os.Getenv("ENV") != "production" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

