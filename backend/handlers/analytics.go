package handlers

import (
	"encoding/json"
	"fmt"
	"link-analytics-service/db"
	"link-analytics-service/models"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AnalyticsResponse struct {
	ShortCode      string                `json:"short_code"`
	TotalClicks    int64                 `json:"total_clicks"`
	UniqueVisitors int64                 `json:"unique_visitors"`
	ClicksOverTime []models.TimePoint    `json:"clicks_over_time"`
	TopReferrers   []models.Referrer     `json:"top_referrers"`
	ClickRate      float64               `json:"click_rate"`      // Clicks per hour/day based on period
	PeakHour       *models.TimePoint     `json:"peak_hour"`      // Hour/day with most clicks
}

// SSEBroker manages Server-Sent Events connections
type SSEBroker struct {
	clients map[string]map[chan []byte]bool
	mu      sync.RWMutex
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[string]map[chan []byte]bool),
	}
}

func (b *SSEBroker) AddClient(shortCode string, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.clients[shortCode] == nil {
		b.clients[shortCode] = make(map[chan []byte]bool)
	}
	b.clients[shortCode][ch] = true
}

func (b *SSEBroker) RemoveClient(shortCode string, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if clients, ok := b.clients[shortCode]; ok {
		delete(clients, ch)
		if len(clients) == 0 {
			delete(b.clients, shortCode)
		}
	}
}

func (b *SSEBroker) Broadcast(shortCode string, data []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if clients, ok := b.clients[shortCode]; ok {
		for ch := range clients {
			select {
			case ch <- data:
			default:
				// Channel is full, skip this client
			}
		}
	}
}

// GetAnalytics handles GET /api/analytics/{short_code}?period=24h|7d|30d
func GetAnalytics(pgDB *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract short code from path like /api/analytics/{shortCode} or /analytics/{shortCode}
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		var shortCode string
		// Handle both /api/analytics/{shortCode} and /analytics/{shortCode}
		if len(pathParts) >= 3 && pathParts[0] == "api" && pathParts[1] == "analytics" {
			shortCode = pathParts[2]
		} else if len(pathParts) >= 2 && pathParts[0] == "analytics" {
			shortCode = pathParts[1]
		} else {
			http.Error(w, "Short code required", http.StatusBadRequest)
			return
		}

		periodStr := r.URL.Query().Get("period")
		if periodStr == "" {
			periodStr = "24h"
		}

		var period time.Duration
		switch periodStr {
		case "24h":
			period = 24 * time.Hour
		case "7d":
			period = 7 * 24 * time.Hour
		case "30d":
			period = 30 * 24 * time.Hour
		default:
			http.Error(w, "Invalid period. Use 24h, 7d, or 30d", http.StatusBadRequest)
			return
		}

		// Get overall stats
		stats, err := pgDB.GetLinkStats(r.Context(), shortCode)
		if err != nil {
			log.Printf("Error getting stats: %v", err)
			stats = &models.LinkStats{
				ShortCode:      shortCode,
				TotalClicks:    0,
				UniqueVisitors: 0,
			}
		}

		// Recalculate unique visitors for accuracy (in case worker hasn't updated yet)
		if stats.TotalClicks > 0 {
			uniqueVisitors, err := pgDB.RecalculateUniqueVisitors(r.Context(), shortCode)
			if err == nil {
				stats.UniqueVisitors = uniqueVisitors
			}
		}

		// Get clicks over time
		clicksOverTime, err := pgDB.GetClicksOverTime(r.Context(), shortCode, period)
		if err != nil {
			log.Printf("Error getting clicks over time: %v", err)
			clicksOverTime = []models.TimePoint{}
		}

		// Get top referrers
		topReferrers, err := pgDB.GetTopReferrers(r.Context(), shortCode, 10)
		if err != nil {
			log.Printf("Error getting top referrers: %v", err)
			topReferrers = []models.Referrer{}
		}

		// Ensure arrays are never nil
		if clicksOverTime == nil {
			clicksOverTime = []models.TimePoint{}
		}
		if topReferrers == nil {
			topReferrers = []models.Referrer{}
		}

		// Calculate click rate (clicks per hour or per day based on period)
		var clickRate float64
		if len(clicksOverTime) > 0 {
			if period <= 24*time.Hour {
				// Clicks per hour
				hours := float64(period.Hours())
				if hours > 0 {
					clickRate = float64(stats.TotalClicks) / hours
				}
			} else {
				// Clicks per day
				days := float64(period.Hours()) / 24.0
				if days > 0 {
					clickRate = float64(stats.TotalClicks) / days
				}
			}
		}

		// Find peak hour/day (time bucket with most clicks)
		var peakHour *models.TimePoint
		if len(clicksOverTime) > 0 {
			maxCount := int64(0)
			for i := range clicksOverTime {
				if clicksOverTime[i].Count > maxCount {
					maxCount = clicksOverTime[i].Count
					peakHour = &clicksOverTime[i]
				}
			}
		}

		response := AnalyticsResponse{
			ShortCode:      shortCode,
			TotalClicks:    stats.TotalClicks,
			UniqueVisitors: stats.UniqueVisitors,
			ClicksOverTime: clicksOverTime,
			TopReferrers:   topReferrers,
			ClickRate:      clickRate,
			PeakHour:       peakHour,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// StreamAnalytics handles GET /api/analytics/{short_code}/stream (SSE)
func StreamAnalytics(pgDB *db.PostgresDB, redisDB *db.RedisDB, broker *SSEBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle OPTIONS for CORS preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract short code from path like /api/analytics/{shortCode}/stream
		// The router strips /api, so the path is /analytics/{shortCode}/stream
		path := strings.TrimPrefix(r.URL.Path, "/api")
		path = strings.TrimPrefix(path, "/analytics/")
		path = strings.TrimSuffix(path, "/stream")
		path = strings.Trim(path, "/")
		
		shortCode := path
		if shortCode == "" {
			log.Printf("StreamAnalytics: empty short code from path: %s", r.URL.Path)
			http.Error(w, "Short code required", http.StatusBadRequest)
			return
		}
		
		// Remove any query parameters
		if idx := strings.Index(shortCode, "?"); idx != -1 {
			shortCode = shortCode[:idx]
		}

		log.Printf("StreamAnalytics: starting stream for short code: %s", shortCode)

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // Disable buffering in nginx
		
		// CORS headers for SSE
		origin := r.Header.Get("Origin")
		allowedOrigin := "http://localhost:3000"
		if origin == allowedOrigin || origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Create client channel
		clientChan := make(chan []byte, 10)
		broker.AddClient(shortCode, clientChan)
		defer broker.RemoveClient(shortCode, clientChan)

		// Verify we can flush (required for SSE)
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send initial count
		ctx := r.Context()
		counterKey := "clicks:realtime:" + shortCode
		count, _ := redisDB.GetInt(ctx, counterKey)
		stats, _ := pgDB.GetLinkStats(ctx, shortCode)
		if stats != nil {
			count = stats.TotalClicks
		}

		initialData := map[string]interface{}{
			"short_code":   shortCode,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
			"total_clicks": count,
		}
		if data, err := json.Marshal(initialData); err == nil {
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()
		} else {
			log.Printf("Error marshaling initial data: %v", err)
		}

		// Heartbeat ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg := <-clientChan:
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(msg)); err != nil {
					log.Printf("Error writing SSE message: %v", err)
					return
				}
				flusher.Flush()
			case <-ticker.C:
				// Send heartbeat
				if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
					log.Printf("Error writing heartbeat: %v", err)
					return
				}
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

