package workers

import (
	"context"
	"encoding/json"
	"link-analytics-service/db"
	"link-analytics-service/handlers"
	"link-analytics-service/models"
	"log"
	"sync"
	"time"
)

const (
	NumWorkers     = 10
	BatchSize      = 100
	BatchTimeout   = 5 * time.Second
)

// StartWorkers starts the analytics worker pool
func StartWorkers(ctx context.Context, pgDB *db.PostgresDB, redisDB *db.RedisDB, broker *handlers.SSEBroker) {
	var wg sync.WaitGroup

	for i := 0; i < NumWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(ctx, id, pgDB, redisDB, broker)
		}(i)
	}

	log.Printf("Started %d analytics workers", NumWorkers)

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Stopping analytics workers...")

	// Wait for all workers to finish
	wg.Wait()
	log.Println("All analytics workers stopped")
}

func worker(ctx context.Context, id int, pgDB *db.PostgresDB, redisDB *db.RedisDB, broker *handlers.SSEBroker) {
	batch := make([]models.ClickEvent, 0, BatchSize)
	ticker := time.NewTicker(BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case event := <-handlers.AnalyticsQueue:
			batch = append(batch, event)

			if len(batch) >= BatchSize {
				flushBatch(ctx, pgDB, redisDB, broker, batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				flushBatch(ctx, pgDB, redisDB, broker, batch)
				batch = batch[:0]
			}

		case <-ctx.Done():
			// Flush remaining events before shutdown
			if len(batch) > 0 {
				flushBatch(ctx, pgDB, redisDB, broker, batch)
			}
			return
		}
	}
}

func flushBatch(ctx context.Context, pgDB *db.PostgresDB, redisDB *db.RedisDB, broker *handlers.SSEBroker, events []models.ClickEvent) {
	if len(events) == 0 {
		return
	}

	// Convert to pointers for batch insert
	eventPtrs := make([]*models.ClickEvent, len(events))
	for i := range events {
		eventPtrs[i] = &events[i]
	}

	// Batch insert into clicks table
	if err := pgDB.BatchInsertClickEvents(ctx, eventPtrs); err != nil {
		log.Printf("Error inserting click events: %v", err)
		return
	}

	// Group events by short code for aggregation
	codeStats := make(map[string]*codeStat)
	referrerStats := make(map[string]map[string]int64) // shortCode -> referer -> count

	for _, event := range events {
		// Initialize if needed
		if codeStats[event.ShortCode] == nil {
			codeStats[event.ShortCode] = &codeStat{
				totalClicks:    0,
				uniqueVisitors: make(map[string]bool),
			}
		}
		if referrerStats[event.ShortCode] == nil {
			referrerStats[event.ShortCode] = make(map[string]int64)
		}

		// Update stats
		codeStats[event.ShortCode].totalClicks++
		codeStats[event.ShortCode].uniqueVisitors[event.VisitorHash] = true

		// Update referrer stats
		if event.Referer != "" {
			referrerStats[event.ShortCode][event.Referer]++
		}
	}

	// Update aggregated statistics
	for shortCode, stats := range codeStats {
		// Get current stats
		currentStats, err := pgDB.GetLinkStats(ctx, shortCode)
		if err != nil {
			// If no stats exist, create new
			currentStats = &models.LinkStats{
				ShortCode:      shortCode,
				TotalClicks:    0,
				UniqueVisitors: 0,
			}
		}

		// Calculate actual unique visitors count from database
		// Recalculate from all clicks to get accurate count
		uniqueVisitors, err := pgDB.RecalculateUniqueVisitors(ctx, shortCode)
		if err != nil {
			log.Printf("Error recalculating unique visitors for %s: %v", shortCode, err)
			// Fallback: use approximate count (current + new unique in batch)
			uniqueVisitors = currentStats.UniqueVisitors + int64(len(stats.uniqueVisitors))
		}

		// Update link_stats table
		if err := pgDB.UpdateLinkStats(ctx, shortCode, stats.totalClicks, uniqueVisitors); err != nil {
			log.Printf("Error updating link stats for %s: %v", shortCode, err)
		}

		// Update top_referrers
		if refs, ok := referrerStats[shortCode]; ok {
			for referer, count := range refs {
				if err := pgDB.UpdateTopReferrers(ctx, shortCode, referer, count); err != nil {
					log.Printf("Error updating top referrers for %s: %v", shortCode, err)
				}
			}
		}

		// Broadcast to SSE clients
		stats, err := pgDB.GetLinkStats(ctx, shortCode)
		if err == nil {
			data := map[string]interface{}{
				"short_code":   shortCode,
				"timestamp":    time.Now().UTC().Format(time.RFC3339),
				"total_clicks": stats.TotalClicks,
			}
			if jsonData, err := json.Marshal(data); err == nil {
				broker.Broadcast(shortCode, jsonData)
			}
		}
	}
}

type codeStat struct {
	totalClicks    int64
	uniqueVisitors map[string]bool
}

