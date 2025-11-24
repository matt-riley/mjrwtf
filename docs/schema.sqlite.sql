-- Database Schema for mjr.wtf URL Shortener
-- SQLite Version
--
-- This schema supports:
-- - Shortened URL storage and management
-- - Click analytics and tracking
-- - Fast lookups and efficient queries
--
-- Usage:
-- sqlite3 database.db < schema.sqlite.sql
--
-- Note: Foreign key constraints must be enabled in SQLite:
-- PRAGMA foreign_keys = ON;

-- Enable foreign key support (must be set for each connection)
PRAGMA foreign_keys = ON;

-- ============================================================================
-- URLs Table
-- ============================================================================
-- Stores shortened URLs with their original destinations
CREATE TABLE IF NOT EXISTS urls (
    -- Primary key: auto-incrementing integer
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
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
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
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
    
    -- Parsed domain from referrer URL (e.g., "google.com")
    -- Extracted from the referrer field for efficient domain-level analytics
    -- NULL if no referer or URL is malformed
    -- Note: Added via migration, appears at end of table
    referrer_domain VARCHAR(255),
    
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

-- Index for referrer domain analytics
CREATE INDEX IF NOT EXISTS idx_clicks_referrer_domain ON clicks(referrer_domain);


-- ============================================================================
-- Example Queries
-- ============================================================================
-- 
-- Get a URL by short code (redirect query):
--   SELECT original_url FROM urls WHERE short_code = 'abc123';
--
-- Record a click:
--   INSERT INTO clicks (url_id, referrer, referrer_domain, country, user_agent) 
--   VALUES (1, 'https://google.com/search', 'google.com', 'US', 'Mozilla/5.0...');
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
-- Get top 10 referrers for a URL:
--   SELECT referrer, COUNT(*) as click_count 
--   FROM clicks 
--   WHERE url_id = 1 AND referrer IS NOT NULL
--   GROUP BY referrer
--   ORDER BY click_count DESC
--   LIMIT 10;
--
-- Get top referrer domains:
--   SELECT referrer_domain, COUNT(*) as click_count 
--   FROM clicks 
--   WHERE url_id = 1 AND referrer_domain IS NOT NULL
--   GROUP BY referrer_domain
--   ORDER BY click_count DESC
--   LIMIT 10;
--
-- Get daily click analytics:
--   SELECT DATE(clicked_at) as date, COUNT(*) as clicks
--   FROM clicks
--   WHERE url_id = 1
--   GROUP BY DATE(clicked_at)
--   ORDER BY date DESC;
--
-- Get recent clicks (last 7 days):
--   SELECT * FROM clicks
--   WHERE url_id = 1
--   AND clicked_at > datetime('now', '-7 days')
--   ORDER BY clicked_at DESC;
