package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type logEntry struct {
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	Status     int           `json:"status"`
	Duration   time.Duration `json:"duration_ms"`
	RemoteAddr string        `json:"remote_addr"`
}

// Logger middleware logs HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for redirect endpoint and health checks (performance)
		path := r.URL.Path
		if (path != "/" && len(path) <= 10 && r.Method == http.MethodGet) ||
			path == "/health" || path == "/ready" || path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		entry := logEntry{
			Method:     r.Method,
			Path:       r.URL.Path,
			Status:     wrapped.statusCode,
			Duration:   duration,
			RemoteAddr: r.RemoteAddr,
		}

		jsonData, _ := json.Marshal(entry)
		log.Println(string(jsonData))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

