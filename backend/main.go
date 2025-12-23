package main

import (
	"context"
	"link-analytics-service/config"
	"link-analytics-service/db"
	"link-analytics-service/handlers"
	"link-analytics-service/middleware"
	"link-analytics-service/workers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Optimize runtime for high concurrency
	// Use all available CPUs for maximum throughput
	runtime.GOMAXPROCS(runtime.NumCPU())

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to PostgreSQL
	pgDB, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgDB.Close()
	log.Println("Connected to PostgreSQL")

	// Connect to Redis
	redisDB, err := db.NewRedisDB(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisDB.Close()
	log.Println("Connected to Redis")

	// Initialize SSE broker
	broker := handlers.NewSSEBroker()

	// Pre-populate L1 cache with all links for maximum performance
	handlers.PrePopulateL1Cache(pgDB)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start analytics workers
	go workers.StartWorkers(ctx, pgDB, redisDB, broker)

	// Setup routes
	mux := http.NewServeMux()

	// API endpoints - wrap handlers with middleware chain
	// Register API routes FIRST so they take precedence
	createLinkHandler := middleware.Chain(
		handlers.CreateLink(pgDB, cfg.FrontendURL),
		middleware.RateLimit(redisDB, 100, time.Minute),
		middleware.Logger,
	)
	getLinkHandler := middleware.Chain(
		handlers.GetLink(pgDB),
		middleware.RateLimit(redisDB, 100, time.Minute),
		middleware.Logger,
	)
	listLinksHandler := middleware.Chain(
		handlers.ListLinks(pgDB),
		middleware.RateLimit(redisDB, 100, time.Minute),
		middleware.Logger,
	)
	getAnalyticsHandler := middleware.Chain(
		handlers.GetAnalytics(pgDB),
		middleware.RateLimit(redisDB, 100, time.Minute),
		middleware.Logger,
	)
	// Stream handler - no logger middleware (SSE streams need immediate response)
	streamAnalyticsHandler := handlers.StreamAnalytics(pgDB, redisDB, broker)
	trackClickHandler := middleware.Chain(
		handlers.TrackClick(pgDB, redisDB),
		middleware.Logger,
	)

	// Health and metrics endpoints (no middleware for performance)
	// Register these directly on mux before the catch-all handler

	// Create a custom API router that manually handles routing
	// This gives us full control over path matching and CORS
	apiRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Remove /api prefix
		path := strings.TrimPrefix(r.URL.Path, "/api")
		if path == "" {
			path = "/"
		}
		
		// Route based on method and path
		switch {
		case r.Method == http.MethodPost && path == "/links":
			createLinkHandler.ServeHTTP(w, r)
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/links/") && path != "/links":
			// Extract shortCode from /links/{shortCode}
			getLinkHandler.ServeHTTP(w, r)
		case r.Method == http.MethodGet && path == "/links":
			listLinksHandler.ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasPrefix(path, "/track/"):
			// Track click endpoint
			trackClickHandler.ServeHTTP(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/stream") && strings.HasPrefix(path, "/analytics/"):
			streamAnalyticsHandler.ServeHTTP(w, r)
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/analytics/"):
			getAnalyticsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	
	// Wrap API router with CORS (handles OPTIONS for all /api/ routes)
	apiHandler := middleware.CORS(apiRouter)
	
	// Mount at /api/ - register BEFORE redirect handler
	mux.Handle("/api/", apiHandler)
	
	// Redirect endpoint (no middleware for performance)
	// Register AFTER API routes as catch-all for short codes
	redirectHandler := handlers.HandleRedirect(pgDB, redisDB)

	// Optimized routing: Check path prefix first to avoid mux.Handler overhead for redirects
	// This is critical for performance - most requests are redirects
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Track request count for metrics (skip for health endpoints)
		if path != "/health" && path != "/ready" && path != "/metrics" {
			handlers.IncrementRequestCount()
		}
		
		// Fast path: Most requests are redirects (not /api/ routes)
		// Check prefix first to avoid expensive mux.Handler call
		if !strings.HasPrefix(path, "/api") && path != "/health" && path != "/ready" && path != "/metrics" {
			// This is likely a redirect request
			if r.Method == http.MethodGet && path != "/" && len(path) > 1 {
				redirectHandler(w, r)
				return
			}
		}
		
		// API routes and health endpoints: Let mux handle it
		mux.ServeHTTP(w, r)
	})

	// Create server with optimized settings for high performance
	server := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        handler,
		ReadTimeout:    5 * time.Second,   // Reduced for faster connection recycling
		WriteTimeout:   5 * time.Second,   // Reduced for faster response
		IdleTimeout:    120 * time.Second, // Increased for connection reuse
		MaxHeaderBytes: 1 << 20,          // 1MB max header size
		// Note: No ReadHeaderTimeout needed for redirects (simple requests)
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel worker context
	cancel()

	// Give workers time to finish processing
	time.Sleep(2 * time.Second)

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
