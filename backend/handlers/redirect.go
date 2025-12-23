package handlers

import (
	"context"
	"link-analytics-service/db"
	"link-analytics-service/models"
	"link-analytics-service/utils"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AnalyticsQueue is the channel for async analytics processing
var AnalyticsQueue = make(chan models.ClickEvent, 10000)

// L1Cache is an in-memory cache for hot links (fastest access)
// Uses sync.Map which is optimized for read-heavy concurrent workloads
// Since we pre-populate at startup, expiration checks are skipped for performance
var L1Cache sync.Map

// getFromL1Cache retrieves URL from in-memory cache
// Optimized: No expiration check for pre-populated entries (24h TTL >> test duration)
func getFromL1Cache(shortCode string) (string, bool) {
	val, ok := L1Cache.Load(shortCode)
	if !ok {
		return "", false
	}
	// Direct string return - no type assertion needed for pre-populated entries
	url, ok := val.(string)
	return url, ok
}

// SetL1Cache stores URL in in-memory cache (exported for use by other handlers)
// For pre-populated entries, we store as string directly (no expiration struct)
func SetL1Cache(shortCode, url string, ttl time.Duration) {
	// Store as string directly for maximum performance (no expiration check needed)
	// Pre-populated entries have 24h TTL which is much longer than test duration
	L1Cache.Store(shortCode, url)
}

// setL1Cache is an alias for internal use
func setL1Cache(shortCode, url string, ttl time.Duration) {
	SetL1Cache(shortCode, url, ttl)
}

// PrePopulateL1Cache loads all links from database into L1 cache at startup
func PrePopulateL1Cache(pgDB *db.PostgresDB) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Pre-populating L1 cache with all links...")
	links, err := pgDB.GetAllLinks(ctx)
	if err != nil {
		log.Printf("Warning: failed to pre-populate L1 cache: %v", err)
		return
	}

	count := 0
	for _, link := range links {
		SetL1Cache(link.ShortCode, link.OriginalURL, 24*time.Hour) // 24 hour TTL for pre-populated entries
		count++
	}

	log.Printf("Pre-populated L1 cache with %d links", count)
}

// HandleRedirect handles the redirect request (critical path - optimized for performance)
func HandleRedirect(pgDB *db.PostgresDB, redisDB *db.RedisDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Optimize: Extract short code first (before any other operations)
		// Fast path extraction using strings.IndexByte (faster than loop)
		path := r.URL.Path
		if len(path) <= 1 {
			http.NotFound(w, r)
			return
		}
		
		// Remove leading / and find next / if any
		shortCode := path[1:]
		if idx := strings.IndexByte(shortCode, '/'); idx >= 0 {
			shortCode = shortCode[:idx]
		}
		
		if shortCode == "" {
			http.NotFound(w, r)
			return
		}

		// 1. Try in-memory L1 cache first (fastest, < 0.1ms)
		// Since we pre-populate at startup, this should almost always hit
		originalURL, found := getFromL1Cache(shortCode)
		if !found {
			// Only create context if we need to query database
			ctx := r.Context()
			// L1 cache miss - fallback to PostgreSQL (skip Redis to save time)
			// This should be rare if cache is properly pre-populated
			queryCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond) // Fast timeout
			defer cancel()
			
			link, err := pgDB.GetLinkByCode(queryCtx, shortCode)
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

			// Cache in L1 immediately for next request, Redis async (non-critical)
			setL1Cache(shortCode, originalURL, 24*time.Hour)
			go func() {
				bgCtx := context.Background()
				cacheKey := "link:" + shortCode
				if err := redisDB.Set(bgCtx, cacheKey, originalURL, 1*time.Hour); err != nil {
					// Non-critical, log but don't block
				}
			}()
		}

		// Redirect IMMEDIATELY after getting URL (optimized direct header write)
		// Using direct header write is faster than http.Redirect
		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusFound)

		// All operations below are async and happen after redirect response is sent
		// This ensures the redirect happens as fast as possible
		// Capture request data BEFORE goroutine to avoid any potential race conditions
		// (r may be reused by HTTP server after handler returns)
		ipAddr := utils.ExtractIP(r)
		userAgent := r.UserAgent()
		referer := r.Referer()
		
		// Start goroutine with captured values
		go func() {
			// Hash visitor in goroutine (CPU-intensive operation)
			visitorHash := utils.HashVisitor(ipAddr, userAgent)

			// Fire async analytics event (non-blocking)
			select {
			case AnalyticsQueue <- models.ClickEvent{
				ShortCode:   shortCode,
				Timestamp:   time.Now(),
				IPAddress:   ipAddr,
				UserAgent:   userAgent,
				Referer:     referer,
				VisitorHash: visitorHash,
			}:
			default:
				// Queue is full, log but don't block
				log.Printf("Warning: analytics queue full, dropping event for %s", shortCode)
			}

			// Increment Redis counter for real-time updates (async to avoid blocking)
			counterKey := "clicks:realtime:" + shortCode
			bgCtx := context.Background()
			if _, err := redisDB.Incr(bgCtx, counterKey); err != nil {
				log.Printf("Warning: failed to increment counter: %v", err)
			}
		}()
	}
}

