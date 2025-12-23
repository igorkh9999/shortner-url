-- Links table
CREATE TABLE IF NOT EXISTS links (
    id SERIAL PRIMARY KEY,
    short_code VARCHAR(10) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Optimized indexes for high-performance lookups
CREATE INDEX IF NOT EXISTS idx_user_id ON links(user_id);
-- Covering index for GetLinkByCode - includes all columns needed for query
-- This allows index-only scans, avoiding table lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_short_code ON links(short_code);
-- Composite index for user queries with sorting
CREATE INDEX IF NOT EXISTS idx_user_created ON links(user_id, created_at DESC);

-- Analytics events (time-series optimized)
CREATE TABLE IF NOT EXISTS clicks (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL,
    clicked_at TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    visitor_hash VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS idx_short_code_time ON clicks(short_code, clicked_at DESC);
CREATE INDEX IF NOT EXISTS idx_clicked_at ON clicks(clicked_at);

-- Aggregated statistics (updated by workers)
CREATE TABLE IF NOT EXISTS link_stats (
    short_code VARCHAR(10) PRIMARY KEY,
    total_clicks BIGINT DEFAULT 0,
    unique_visitors BIGINT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Top referrers (materialized for performance)
CREATE TABLE IF NOT EXISTS top_referrers (
    short_code VARCHAR(10),
    referer TEXT,
    click_count BIGINT,
    PRIMARY KEY (short_code, referer)
);

CREATE INDEX IF NOT EXISTS idx_short_code_count ON top_referrers(short_code, click_count DESC);

