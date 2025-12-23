package handlers

import (
	"link-analytics-service/db"
	"link-analytics-service/models"
	"link-analytics-service/utils"
	"log"
	"net/http"
	"strings"
	"time"
)

// TrackClick handles POST /api/track/{shortCode} - dedicated endpoint for tracking clicks
// This is called by the frontend before redirecting
func TrackClick(pgDB *db.PostgresDB, redisDB *db.RedisDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		
		// Extract short code from path like /api/track/{shortCode}
		// The path will already have /api stripped by the router, so it's /track/{shortCode}
		path := strings.TrimPrefix(r.URL.Path, "/api")
		path = strings.TrimPrefix(path, "/track/")
		path = strings.Trim(path, "/")
		
		shortCode := path
		if shortCode == "" {
			http.Error(w, "Short code required", http.StatusBadRequest)
			return
		}
		
		// Remove any query parameters or fragments
		if idx := strings.Index(shortCode, "?"); idx != -1 {
			shortCode = shortCode[:idx]
		}
		if idx := strings.Index(shortCode, "#"); idx != -1 {
			shortCode = shortCode[:idx]
		}

		// Verify link exists
		_, err := pgDB.GetLinkByCode(ctx, shortCode)
		if err != nil {
			if _, ok := err.(*models.NotFoundError); ok {
				http.Error(w, "Link not found", http.StatusNotFound)
				return
			}
			log.Printf("Error getting link for tracking: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Fire async analytics event (non-blocking)
		select {
		case AnalyticsQueue <- models.ClickEvent{
			ShortCode:   shortCode,
			Timestamp:   time.Now(),
			IPAddress:   utils.ExtractIP(r),
			UserAgent:   r.UserAgent(),
			Referer:     r.Referer(),
			VisitorHash: utils.HashVisitor(utils.ExtractIP(r), r.UserAgent()),
		}:
		default:
			// Queue is full, log but don't block
			log.Printf("Warning: analytics queue full, dropping event for %s", shortCode)
		}

		// Increment Redis counter for real-time updates
		counterKey := "clicks:realtime:" + shortCode
		if _, err := redisDB.Incr(ctx, counterKey); err != nil {
			log.Printf("Warning: failed to increment counter: %v", err)
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"tracked"}`))
	}
}

