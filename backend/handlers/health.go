package handlers

import (
	"encoding/json"
	"fmt"
	"link-analytics-service/db"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Health metrics
var (
	requestCount    atomic.Int64
	errorCount      atomic.Int64
	startTime       = time.Now()
)

// Health handles GET /health - simple health check
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// Readiness handles GET /ready - readiness check with dependencies
func Readiness(pgDB *db.PostgresDB, redisDB *db.RedisDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Check database
		dbHealthy := false
		if err := pgDB.Ping(ctx); err == nil {
			dbHealthy = true
		}
		
		// Check Redis
		redisHealthy := false
		if err := redisDB.Ping(ctx); err == nil {
			redisHealthy = true
		}
		
		status := http.StatusOK
		if !dbHealthy || !redisHealthy {
			status = http.StatusServiceUnavailable
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   map[string]bool{"database": dbHealthy, "redis": redisHealthy},
			"ready":    dbHealthy && redisHealthy,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// Metrics handles GET /metrics - application metrics
func Metrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		uptime := time.Since(startTime).Seconds()
		totalRequests := requestCount.Load()
		totalErrors := errorCount.Load()
		
		errorRate := 0.0
		if totalRequests > 0 {
			errorRate = float64(totalErrors) / float64(totalRequests) * 100
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uptime": map[string]interface{}{
				"seconds": int64(uptime),
				"formatted": formatDuration(time.Since(startTime)),
			},
			"requests": map[string]interface{}{
				"total": totalRequests,
				"errors": totalErrors,
				"error_rate_percent": errorRate,
			},
			"memory": map[string]interface{}{
				"alloc_mb":      bToMb(m.Alloc),
				"total_alloc_mb": bToMb(m.TotalAlloc),
				"sys_mb":        bToMb(m.Sys),
				"num_gc":        m.NumGC,
			},
			"runtime": map[string]interface{}{
				"goroutines": runtime.NumGoroutine(),
				"cpu_count":  runtime.NumCPU(),
			},
			"cache": map[string]interface{}{
				"l1_size": getL1CacheSize(),
			},
		})
	}
}

// IncrementRequestCount increments the request counter
func IncrementRequestCount() {
	requestCount.Add(1)
}

// IncrementErrorCount increments the error counter
func IncrementErrorCount() {
	errorCount.Add(1)
}

// Helper functions
func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
}

func getL1CacheSize() int {
	count := 0
	L1Cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

