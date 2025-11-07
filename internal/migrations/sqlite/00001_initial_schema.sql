-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- URLs Table
-- ============================================================================
-- Stores shortened URLs with their original destinations
--
-- Note: Foreign key support must be enabled in application code when establishing
-- database connections using: PRAGMA foreign_keys = ON;
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
    
    -- HTTP "Referer" header (where the user came from)
    -- Note: "Referer" is the correct spelling of the HTTP header name per RFC 2616
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
-- Note: This index may be inefficient if country data is sparse (many NULL values).
-- Consider whether country-based filtering is a common query pattern before enabling.
-- CREATE INDEX IF NOT EXISTS idx_clicks_country ON clicks(country);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS clicks;
DROP TABLE IF EXISTS urls;
-- +goose StatementEnd
