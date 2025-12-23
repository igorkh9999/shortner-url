package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	Port        string
	BaseURL     string // Base URL for generating short URLs (e.g., http://localhost:8080)
	FrontendURL string // Frontend URL for CORS and short URL generation (e.g., http://localhost:3000)
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	return &Config{
		DatabaseURL: dbURL,
		RedisURL:    redisURL,
		Port:        port,
		BaseURL:     baseURL,
		FrontendURL: frontendURL,
	}, nil
}

