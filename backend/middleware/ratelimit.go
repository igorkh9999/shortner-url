package middleware

import (
	"fmt"
	"link-analytics-service/db"
	"link-analytics-service/utils"
	"log"
	"net/http"
	"strconv"
	"time"
)

// RateLimit middleware implements rate limiting using Redis
func RateLimit(redisDB *db.RedisDB, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for redirect endpoint (performance)
			if r.URL.Path != "/" && len(r.URL.Path) <= 10 && r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Skip rate limiting for SSE stream endpoint
			if r.URL.Path != "" && len(r.URL.Path) > 0 && r.URL.Path[len(r.URL.Path)-7:] == "/stream" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract IP address
			ip := utils.ExtractIP(r)

			key := fmt.Sprintf("ratelimit:%s:%s", ip, r.URL.Path)

			ctx := r.Context()
			count, err := redisDB.Incr(ctx, key)
			if err != nil {
				// If Redis fails, allow the request (fail open)
				log.Printf("Rate limit check failed: %v", err)
				next.ServeHTTP(w, r)
				return
			}

			// Set TTL on first request
			if count == 1 {
				redisDB.Set(ctx, key, strconv.FormatInt(count, 10), window)
			}

			if count > int64(limit) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error":"Rate limit exceeded"}`)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

