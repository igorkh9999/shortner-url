package handlers

import (
	"encoding/json"
	"link-analytics-service/db"
	"link-analytics-service/models"
	"link-analytics-service/utils"
	"log"
	"net/http"
	"strings"
	"time"
)

type CreateLinkRequest struct {
	URL    string `json:"url"`
	UserID string `json:"user_id"`
}

type CreateLinkResponse struct {
	ShortCode   string    `json:"short_code"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type LinkResponse struct {
	ShortCode   string         `json:"short_code"`
	OriginalURL string         `json:"original_url"`
	CreatedAt   time.Time      `json:"created_at"`
	Stats       *models.LinkStats `json:"stats"`
}

type ListLinksResponse struct {
	Links []LinkInfo `json:"links"`
}

type LinkInfo struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	TotalClicks int64     `json:"total_clicks"`
}

// CreateLink handles POST /api/links
func CreateLink(pgDB *db.PostgresDB, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate URL
		if !utils.IsValidURL(req.URL) {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		// Generate short code (retry on collision)
		var link *models.Link
		maxRetries := 5
		for i := 0; i < maxRetries; i++ {
			shortCode := utils.GenerateShortCode()
			link = &models.Link{
				ShortCode:   shortCode,
				OriginalURL: req.URL,
				UserID:      req.UserID,
			}

			err := pgDB.CreateLink(r.Context(), link)
			if err == nil {
				break
			}

			// Check if it's a unique constraint violation
			if i == maxRetries-1 {
				log.Printf("Failed to create link after %d retries: %v", maxRetries, err)
				http.Error(w, "Failed to create link", http.StatusInternalServerError)
				return
			}
		}

		// Use provided baseURL (frontend URL) for short links
		// The frontend will handle the redirect
		response := CreateLinkResponse{
			ShortCode:   link.ShortCode,
			ShortURL:    baseURL + "/" + link.ShortCode,
			OriginalURL: link.OriginalURL,
			CreatedAt:   link.CreatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// GetLink handles GET /api/links/{short_code}
func GetLink(pgDB *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract short code from path like /api/links/{shortCode} or /links/{shortCode}
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		var shortCode string
		// Handle both /api/links/{shortCode} and /links/{shortCode}
		if len(pathParts) >= 3 && pathParts[0] == "api" && pathParts[1] == "links" {
			shortCode = pathParts[2]
		} else if len(pathParts) >= 2 && pathParts[0] == "links" {
			shortCode = pathParts[1]
		} else {
			http.Error(w, "Short code required", http.StatusBadRequest)
			return
		}

		link, err := pgDB.GetLinkByCode(r.Context(), shortCode)
		if err != nil {
			if _, ok := err.(*models.NotFoundError); ok {
				http.Error(w, "Link not found", http.StatusNotFound)
				return
			}
			log.Printf("Error getting link: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Get stats
		stats, err := pgDB.GetLinkStats(r.Context(), shortCode)
		if err != nil {
			log.Printf("Error getting stats: %v", err)
			stats = &models.LinkStats{
				ShortCode:      shortCode,
				TotalClicks:    0,
				UniqueVisitors: 0,
			}
		}

		response := LinkResponse{
			ShortCode:   link.ShortCode,
			OriginalURL: link.OriginalURL,
			CreatedAt:   link.CreatedAt,
			Stats:       stats,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ListLinks handles GET /api/links?user_id=...
func ListLinks(pgDB *db.PostgresDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id parameter required", http.StatusBadRequest)
			return
		}

		links, err := pgDB.GetLinksByUser(r.Context(), userID)
		if err != nil {
			log.Printf("Error getting links: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Get stats for each link
		linkInfos := make([]LinkInfo, 0, len(links))
		for _, link := range links {
			stats, err := pgDB.GetLinkStats(r.Context(), link.ShortCode)
			if err != nil {
				log.Printf("Error getting stats for %s: %v", link.ShortCode, err)
				stats = &models.LinkStats{
					ShortCode:      link.ShortCode,
					TotalClicks:    0,
					UniqueVisitors: 0,
				}
			}

			linkInfos = append(linkInfos, LinkInfo{
				ShortCode:   link.ShortCode,
				OriginalURL: link.OriginalURL,
				CreatedAt:   link.CreatedAt,
				TotalClicks: stats.TotalClicks,
			})
		}

		response := ListLinksResponse{Links: linkInfos}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

