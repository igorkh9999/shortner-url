package models

import "time"

// Link represents a shortened URL
type Link struct {
	ID          int       `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// ClickEvent represents a click analytics event
type ClickEvent struct {
	ShortCode   string    `json:"short_code"`
	Timestamp   time.Time `json:"timestamp"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Referer     string    `json:"referer"`
	VisitorHash string    `json:"visitor_hash"`
}

// LinkStats represents aggregated statistics for a link
type LinkStats struct {
	ShortCode      string `json:"short_code"`
	TotalClicks    int64  `json:"total_clicks"`
	UniqueVisitors int64  `json:"unique_visitors"`
}

// TimePoint represents a data point in time-series analytics
type TimePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

// Referrer represents referrer statistics
type Referrer struct {
	Referer    string `json:"referer"`
	ClickCount int64  `json:"count"`
}

// Error types
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

