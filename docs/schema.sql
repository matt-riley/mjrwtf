-- Database Schema for mjr.wtf URL Shortener
-- Compatible with both SQLite and PostgreSQL
--
-- This schema supports:
-- - Shortened URL storage and management
-- - Click analytics and tracking
-- - Fast lookups and efficient queries
--
-- Usage:
-- PostgreSQL: psql -U username -d database -f schema.sql
-- SQLite: sqlite3 database.db < schema.sql

-- ============================================================================
-- URLs Table
-- ============================================================================
-- Stores shortened URLs with their original destinations
--
-- Note: For PostgreSQL, change "INTEGER PRIMARY KEY AUTOINCREMENT" to "SERIAL PRIMARY KEY"
-- SQLite uses: INTEGER PRIMARY KEY AUTOINCREMENT
-- PostgreSQL uses: SERIAL PRIMARY KEY or BIGSERIAL PRIMARY KEY
CREATE TABLE IF NOT EXISTS urls (
    -- Primary key: auto-incrementing integer
    -- SQLite: INTEGER PRIMARY KEY AUTOINCREMENT
    -- PostgreSQL: SERIAL PRIMARY KEY (uncomment line below for PostgreSQL)
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- id SERIAL PRIMARY KEY,  -- Use this line for PostgreSQL instead
    
    -- Short code: unique identifier for the shortened URL
    -- Example: "abc123" for https://mjr.wtf/abc123
    short_code VARCHAR(255) NOT NULL UNIQUE,
    
    -- Original URL: the destination URL
    -- Example: "https://example.com/very/long/path"
    original_url TEXT NOT NULL,
    
    -- Timestamp when the URL was created
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- User/system that created this short URL
    -- Can be an API key identifier, user ID, or system name
    created_by VARCHAR(255) NOT NULL
);

-- Index for fast lookups by short_code (most common query pattern)
-- This is critical for redirect performance
CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);

-- Index for filtering by creator
CREATE INDEX IF NOT EXISTS idx_urls_created_by ON urls(created_by);

-- Index for sorting/filtering by creation time
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);


-- ============================================================================
-- Clicks Table
-- ============================================================================
-- Stores analytics data for each click on a shortened URL
--
-- Note: For PostgreSQL, change "INTEGER PRIMARY KEY AUTOINCREMENT" to "SERIAL PRIMARY KEY"
CREATE TABLE IF NOT EXISTS clicks (
    -- Primary key: auto-incrementing integer
    -- SQLite: INTEGER PRIMARY KEY AUTOINCREMENT
    -- PostgreSQL: SERIAL PRIMARY KEY (uncomment line below for PostgreSQL)
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- id SERIAL PRIMARY KEY,  -- Use this line for PostgreSQL instead
    
    -- Foreign key reference to the URLs table
    -- Indicates which shortened URL was clicked
    url_id INTEGER NOT NULL,
    
    -- Timestamp when the click occurred
    clicked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
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

-- Index for fast lookups of clicks by URL (most common analytics query)
CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);

-- Composite index for time-based analytics queries
-- Supports queries like "clicks per day" or "clicks in date range"
CREATE INDEX IF NOT EXISTS idx_clicks_url_id_clicked_at ON clicks(url_id, clicked_at);

-- Index for time-based filtering and sorting
CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at);

-- Index for country-based analytics
CREATE INDEX IF NOT EXISTS idx_clicks_country ON clicks(country);


-- ============================================================================
-- PostgreSQL Specific Optimizations
-- ============================================================================
-- The following adjustments are recommended when deploying to PostgreSQL:
--
-- 1. Use SERIAL or BIGSERIAL instead of INTEGER PRIMARY KEY AUTOINCREMENT:
--    id SERIAL PRIMARY KEY  (or BIGSERIAL for larger datasets)
--
-- 2. Use TIMESTAMPTZ for timezone-aware timestamps:
--    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
--    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
--
-- 3. Consider using UUID for primary keys if distributed systems are needed:
--    id UUID PRIMARY KEY DEFAULT gen_random_uuid()
--
-- 4. Add partial indexes for common queries:
--    CREATE INDEX idx_recent_clicks ON clicks(url_id, clicked_at) 
--      WHERE clicked_at > NOW() - INTERVAL '30 days';
--
-- 5. Consider table partitioning for the clicks table if expecting high volume:
--    Partition by clicked_at range (e.g., monthly partitions)


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
--   SELECT DATE(clicked_at) as date, COUNT(*) as clicks
--   FROM clicks
--   WHERE url_id = 1
--   GROUP BY DATE(clicked_at)
--   ORDER BY date DESC;
