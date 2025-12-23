package db

import (
	"context"
	"database/sql"
	"fmt"
	"link-analytics-service/models"
	"time"

	_ "github.com/lib/pq"
)

type PostgresDB struct {
	db *sql.DB
}

func NewPostgresDB(databaseURL string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Increase pool size for high concurrency (1000+ RPS)
	// Most requests should hit L1 cache, but we need capacity for cache misses
	db.SetMaxOpenConns(200)  // Increased for high load
	db.SetMaxIdleConns(50)   // Increased idle connections for faster reuse
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute) // Close idle connections faster

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{db: db}, nil
}

func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// Ping checks database connectivity
func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresDB) CreateLink(ctx context.Context, link *models.Link) error {
	query := `INSERT INTO links (short_code, original_url, user_id, created_at) 
	          VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	
	err := p.db.QueryRowContext(ctx, query, link.ShortCode, link.OriginalURL, link.UserID, time.Now()).
		Scan(&link.ID, &link.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	}
	return nil
}

func (p *PostgresDB) GetLinkByCode(ctx context.Context, shortCode string) (*models.Link, error) {
	query := `SELECT id, short_code, original_url, user_id, created_at 
	          FROM links WHERE short_code = $1`
	
	link := &models.Link{}
	err := p.db.QueryRowContext(ctx, query, shortCode).
		Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.UserID, &link.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, &models.NotFoundError{Message: "link not found"}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get link: %w", err)
	}
	return link, nil
}

func (p *PostgresDB) GetLinksByUser(ctx context.Context, userID string) ([]*models.Link, error) {
	query := `SELECT id, short_code, original_url, user_id, created_at 
	          FROM links WHERE user_id = $1 ORDER BY created_at DESC`
	
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query links: %w", err)
	}
	defer rows.Close()

	var links []*models.Link
	for rows.Next() {
		link := &models.Link{}
		if err := rows.Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.UserID, &link.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return links, nil
}

// GetAllLinks retrieves all links from the database (for cache pre-population)
func (p *PostgresDB) GetAllLinks(ctx context.Context) ([]*models.Link, error) {
	query := `SELECT id, short_code, original_url, user_id, created_at 
	          FROM links ORDER BY created_at DESC`
	
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all links: %w", err)
	}
	defer rows.Close()

	var links []*models.Link
	for rows.Next() {
		link := &models.Link{}
		if err := rows.Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.UserID, &link.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return links, nil
}

func (p *PostgresDB) InsertClickEvent(ctx context.Context, event *models.ClickEvent) error {
	query := `INSERT INTO clicks (short_code, clicked_at, ip_address, user_agent, referer, visitor_hash)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err := p.db.ExecContext(ctx, query, event.ShortCode, event.Timestamp, event.IPAddress, 
		event.UserAgent, event.Referer, event.VisitorHash)
	if err != nil {
		return fmt.Errorf("failed to insert click event: %w", err)
	}
	return nil
}

func (p *PostgresDB) BatchInsertClickEvents(ctx context.Context, events []*models.ClickEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO clicks (short_code, clicked_at, ip_address, user_agent, referer, visitor_hash)
	                                      VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		_, err := stmt.ExecContext(ctx, event.ShortCode, event.Timestamp, event.IPAddress,
			event.UserAgent, event.Referer, event.VisitorHash)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (p *PostgresDB) GetLinkStats(ctx context.Context, shortCode string) (*models.LinkStats, error) {
	query := `SELECT short_code, total_clicks, unique_visitors 
	          FROM link_stats WHERE short_code = $1`
	
	stats := &models.LinkStats{}
	err := p.db.QueryRowContext(ctx, query, shortCode).
		Scan(&stats.ShortCode, &stats.TotalClicks, &stats.UniqueVisitors)
	if err == sql.ErrNoRows {
		return &models.LinkStats{
			ShortCode:      shortCode,
			TotalClicks:    0,
			UniqueVisitors: 0,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get link stats: %w", err)
	}
	return stats, nil
}

func (p *PostgresDB) GetClicksOverTime(ctx context.Context, shortCode string, period time.Duration) ([]models.TimePoint, error) {
	startTime := time.Now().Add(-period)
	
	var query string
	
	if period <= 24*time.Hour {
		query = `SELECT DATE_TRUNC('hour', clicked_at) as time_bucket, COUNT(*) as count
		         FROM clicks
		         WHERE short_code = $1 AND clicked_at >= $2
		         GROUP BY time_bucket
		         ORDER BY time_bucket ASC`
	} else {
		query = `SELECT DATE_TRUNC('day', clicked_at) as time_bucket, COUNT(*) as count
		         FROM clicks
		         WHERE short_code = $1 AND clicked_at >= $2
		         GROUP BY time_bucket
		         ORDER BY time_bucket ASC`
	}

	rows, err := p.db.QueryContext(ctx, query, shortCode, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query clicks over time: %w", err)
	}
	defer rows.Close()

	var points []models.TimePoint
	for rows.Next() {
		var point models.TimePoint
		if err := rows.Scan(&point.Timestamp, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan time point: %w", err)
		}
		points = append(points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return points, nil
}

func (p *PostgresDB) GetTopReferrers(ctx context.Context, shortCode string, limit int) ([]models.Referrer, error) {
	query := `SELECT referer, click_count 
	          FROM top_referrers 
	          WHERE short_code = $1 
	          ORDER BY click_count DESC 
	          LIMIT $2`
	
	rows, err := p.db.QueryContext(ctx, query, shortCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top referrers: %w", err)
	}
	defer rows.Close()

	var referrers []models.Referrer
	for rows.Next() {
		var ref models.Referrer
		if err := rows.Scan(&ref.Referer, &ref.ClickCount); err != nil {
			return nil, fmt.Errorf("failed to scan referrer: %w", err)
		}
		referrers = append(referrers, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return referrers, nil
}

func (p *PostgresDB) UpdateLinkStats(ctx context.Context, shortCode string, totalClicks int64, uniqueVisitors int64) error {
	query := `INSERT INTO link_stats (short_code, total_clicks, unique_visitors, last_updated)
	          VALUES ($1, $2, $3, NOW())
	          ON CONFLICT (short_code) 
	          DO UPDATE SET 
	            total_clicks = link_stats.total_clicks + $2,
	            unique_visitors = $3,
	            last_updated = NOW()`
	
	_, err := p.db.ExecContext(ctx, query, shortCode, totalClicks, uniqueVisitors)
	if err != nil {
		return fmt.Errorf("failed to update link stats: %w", err)
	}
	return nil
}

// RecalculateUniqueVisitors recalculates unique visitors count from clicks table
func (p *PostgresDB) RecalculateUniqueVisitors(ctx context.Context, shortCode string) (int64, error) {
	query := `SELECT COUNT(DISTINCT visitor_hash) 
	          FROM clicks 
	          WHERE short_code = $1`
	
	var count int64
	err := p.db.QueryRowContext(ctx, query, shortCode).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to recalculate unique visitors: %w", err)
	}
	return count, nil
}

func (p *PostgresDB) UpdateTopReferrers(ctx context.Context, shortCode string, referer string, count int64) error {
	query := `INSERT INTO top_referrers (short_code, referer, click_count)
	          VALUES ($1, $2, $3)
	          ON CONFLICT (short_code, referer)
	          DO UPDATE SET click_count = top_referrers.click_count + $3`
	
	_, err := p.db.ExecContext(ctx, query, shortCode, referer, count)
	if err != nil {
		return fmt.Errorf("failed to update top referrers: %w", err)
	}
	return nil
}

func (p *PostgresDB) GetUniqueVisitors(ctx context.Context, shortCode string, startTime time.Time) (int64, error) {
	query := `SELECT COUNT(DISTINCT visitor_hash) 
	          FROM clicks 
	          WHERE short_code = $1 AND clicked_at >= $2`
	
	var count int64
	err := p.db.QueryRowContext(ctx, query, shortCode, startTime).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unique visitors: %w", err)
	}
	return count, nil
}

