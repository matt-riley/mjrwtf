-- Database Schema for mjr.wtf URL Shortener
-- PostgreSQL Version
--
-- This schema supports:
-- - Shortened URL storage and management
-- - Click analytics and tracking
-- - Fast lookups and efficient queries
--
-- Usage:
-- psql -U username -d database -f schema.postgres.sql

-- ============================================================================
-- URLs Table
-- ============================================================================
-- Stores shortened URLs with their original destinations
CREATE TABLE IF NOT EXISTS urls (
    -- Primary key: auto-incrementing integer
    id SERIAL PRIMARY KEY,
    
    -- Short code: unique identifier for the shortened URL
    -- Example: "abc123" for https://mjr.wtf/abc123
    short_code VARCHAR(255) NOT NULL UNIQUE,
    
    -- Original URL: the destination URL
    -- Example: "https://example.com/very/long/path"
    original_url TEXT NOT NULL,
    
    -- Timestamp when the URL was created (timezone-aware)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- User/system that created this short URL
    -- Can be an API key identifier, user ID, or system name
    created_by VARCHAR(255) NOT NULL
);

-- Note: The UNIQUE constraint on short_code automatically creates an index
-- No additional index needed for short_code lookups

-- Index for filtering by creator
CREATE INDEX IF NOT EXISTS idx_urls_created_by ON urls(created_by);

-- Index for sorting/filtering by creation time
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);


-- ============================================================================
-- Clicks Table
-- ============================================================================
-- Stores analytics data for each click on a shortened URL
CREATE TABLE IF NOT EXISTS clicks (
    -- Primary key: auto-incrementing integer
    id SERIAL PRIMARY KEY,
    
    -- Foreign key reference to the URLs table
    -- Indicates which shortened URL was clicked
    url_id INTEGER NOT NULL,
    
    -- Timestamp when the click occurred (timezone-aware)
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- HTTP Referer header (where the user came from)
    -- NULL if no referer or direct access
    referrer TEXT,
    
    -- Country code derived from IP address
    -- Example: "US", "GB", "CA" (ISO 3166-1 alpha-2)
    -- NULL if GeoIP is disabled or lookup fails
    country VARCHAR(2),
    
    -- User-Agent header from the request
    -- Useful for device/browser analytics
    user_agent TEXT,
    
    -- Foreign key constraint linking to urls table
    -- ON DELETE CASCADE: when a URL is deleted, all its clicks are deleted too
    FOREIGN KEY (url_id) REFERENCES urls(id) ON DELETE CASCADE
);

-- Composite index for time-based analytics queries
-- Note: This composite index with url_id as the leading column also efficiently
-- serves queries filtering only on url_id, making a separate single-column index unnecessary
-- Supports queries like "clicks per day" or "clicks in date range"
CREATE INDEX IF NOT EXISTS idx_clicks_url_id_clicked_at ON clicks(url_id, clicked_at);

-- Index for time-based filtering and sorting
CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at);

-- Index for country-based analytics
CREATE INDEX IF NOT EXISTS idx_clicks_country ON clicks(country);


-- ============================================================================
-- Optional: Performance Optimizations for High-Volume Deployments
-- ============================================================================

-- Partial index for recent clicks (improves query performance for recent data)
-- Uncomment if you frequently query recent clicks:
-- CREATE INDEX idx_recent_clicks ON clicks(url_id, clicked_at) 
--   WHERE clicked_at > NOW() - INTERVAL '30 days';

-- Consider using BIGSERIAL for id fields if expecting very high volume (>2 billion records)
-- ALTER TABLE urls ALTER COLUMN id TYPE BIGINT;
-- ALTER TABLE clicks ALTER COLUMN id TYPE BIGINT;

-- Consider table partitioning for clicks table with high volume
-- Example: partition by month
-- 
-- To use partitioning, first create the clicks table as a partitioned table:
--   CREATE TABLE clicks (
--     id SERIAL PRIMARY KEY,
--     url_id INTEGER NOT NULL REFERENCES urls(id),
--     clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     referrer TEXT,
--     country VARCHAR(2),
--     user_agent TEXT
--   ) PARTITION BY RANGE (clicked_at);
--
-- Then create partitions, e.g.:
--   CREATE TABLE clicks_2024_01 PARTITION OF clicks
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');


-- ============================================================================
-- Example Queries
-- ============================================================================
-- 
-- Get a URL by short code (redirect query):
--   SELECT original_url FROM urls WHERE short_code = 'abc123';
--
-- Record a click:
--   INSERT INTO clicks (url_id, referrer, country, user_agent) 
--   VALUES (1, 'https://google.com', 'US', 'Mozilla/5.0...');
--
-- Get click count for a URL:
--   SELECT COUNT(*) FROM clicks WHERE url_id = 1;
--
-- Get clicks by country for a URL:
--   SELECT country, COUNT(*) as click_count 
--   FROM clicks 
--   WHERE url_id = 1 AND country IS NOT NULL
--   GROUP BY country
--   ORDER BY click_count DESC;
--
-- Get daily click analytics:
--   SELECT DATE_TRUNC('day', clicked_at) as date, COUNT(*) as clicks
--   FROM clicks
--   WHERE url_id = 1
--   GROUP BY DATE_TRUNC('day', clicked_at)
--   ORDER BY date DESC;
--
-- Get click trends over time with timezone:
--   SELECT 
--     DATE_TRUNC('day', clicked_at AT TIME ZONE 'UTC') as date,
--     COUNT(*) as clicks
--   FROM clicks
--   WHERE url_id = 1
--   AND clicked_at > NOW() - INTERVAL '7 days'
--   GROUP BY DATE_TRUNC('day', clicked_at AT TIME ZONE 'UTC')
--   ORDER BY date DESC;
