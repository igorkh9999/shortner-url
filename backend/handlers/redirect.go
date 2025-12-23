package handlers

import (
	"link-analytics-service/db"
	"link-analytics-service/models"
	"link-analytics-service/utils"
	"log"
	"net/http"
	"time"
)

// AnalyticsQueue is the channel for async analytics processing
var AnalyticsQueue = make(chan models.ClickEvent, 10000)

// HandleRedirect handles the redirect request (critical path - optimized for performance)
func HandleRedirect(pgDB *db.PostgresDB, redisDB *db.RedisDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		shortCode := utils.ExtractShortCode(r.URL.Path)

		if shortCode == "" {
			http.NotFound(w, r)
			return
		}

		// 1. Try Redis cache first (< 1ms target)
		cacheKey := "link:" + shortCode
		originalURL, err := redisDB.Get(ctx, cacheKey)
		if err != nil {
			// 2. Cache miss - query PostgreSQL
			link, err := pgDB.GetLinkByCode(ctx, shortCode)
			if err != nil {
				if _, ok := err.(*models.NotFoundError); ok {
					http.NotFound(w, r)
					return
				}
				log.Printf("Error getting link: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			originalURL = link.OriginalURL

			// 3. Cache for future requests (TTL: 1 hour)
			if err := redisDB.Set(ctx, cacheKey, originalURL, 1*time.Hour); err != nil {
				log.Printf("Warning: failed to cache link: %v", err)
			}
		}

		// 4. Fire async analytics event (non-blocking)
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

		// 5. Increment Redis counter for real-time updates
		counterKey := "clicks:realtime:" + shortCode
		if _, err := redisDB.Incr(ctx, counterKey); err != nil {
			log.Printf("Warning: failed to increment counter: %v", err)
		}

		// 6. Redirect immediately with 301 (Permanent Redirect)
		http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
	}
}

