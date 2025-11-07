# Database Schema Documentation

This directory contains the database schema for the mjr.wtf URL shortener application.

## Schema Files

- **`schema.sql`** - Base schema with comments for both SQLite and PostgreSQL compatibility
- **`schema.sqlite.sql`** - SQLite-specific schema (development)
- **`schema.postgres.sql`** - PostgreSQL-specific schema (production)

## Tables

### urls
Stores shortened URLs with their original destinations.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER/SERIAL | Primary key, auto-incrementing |
| `short_code` | VARCHAR(255) | Unique identifier for the shortened URL (e.g., "abc123") |
| `original_url` | TEXT | The destination URL |
| `created_at` | TIMESTAMP/TIMESTAMPTZ | When the URL was created |
| `created_by` | VARCHAR(255) | User/system that created this URL (API key, user ID, etc.) |

**Constraints:**
- UNIQUE on `short_code`
- NOT NULL on `short_code`, `original_url`, `created_at`, `created_by`

**Indexes:**
- `idx_urls_short_code` on `short_code` - Critical for redirect performance
- `idx_urls_created_by` on `created_by` - For filtering by creator
- `idx_urls_created_at` on `created_at` - For sorting/filtering by creation time

### clicks
Stores analytics data for each click on a shortened URL.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER/SERIAL | Primary key, auto-incrementing |
| `url_id` | INTEGER | Foreign key reference to `urls.id` |
| `clicked_at` | TIMESTAMP/TIMESTAMPTZ | When the click occurred |
| `referrer` | TEXT | HTTP Referer header (nullable) |
| `country` | VARCHAR(2) | ISO 3166-1 alpha-2 country code (nullable) |
| `user_agent` | TEXT | User-Agent header (nullable) |

**Constraints:**
- FOREIGN KEY `url_id` REFERENCES `urls(id)` ON DELETE CASCADE
- NOT NULL on `url_id`, `clicked_at`

**Indexes:**
- `idx_clicks_url_id` on `url_id` - For looking up clicks by URL
- `idx_clicks_url_id_clicked_at` on `(url_id, clicked_at)` - Composite index for time-based analytics
- `idx_clicks_clicked_at` on `clicked_at` - For time-based filtering and sorting
- `idx_clicks_country` on `country` - For country-based analytics

## Usage

### SQLite (Development)

```bash
# Create database with schema
sqlite3 database.db < docs/schema.sqlite.sql

# Or use the base schema
sqlite3 database.db < docs/schema.sql
```

**Important:** Foreign key constraints must be enabled for each connection:
```sql
PRAGMA foreign_keys = ON;
```

### PostgreSQL (Production)

```bash
# Create database
createdb mjrwtf

# Apply schema
psql -U username -d mjrwtf -f docs/schema.postgres.sql

# Or use the base schema (requires manual adjustment)
psql -U username -d mjrwtf -f docs/schema.sql
```

## Common Queries

### Redirect Query (most common)
```sql
SELECT original_url FROM urls WHERE short_code = 'abc123';
```

### Record a Click
```sql
INSERT INTO clicks (url_id, referrer, country, user_agent) 
VALUES (1, 'https://google.com', 'US', 'Mozilla/5.0...');
```

### Get Click Count
```sql
SELECT COUNT(*) FROM clicks WHERE url_id = 1;
```

### Country Analytics
```sql
SELECT country, COUNT(*) as click_count 
FROM clicks 
WHERE url_id = 1 AND country IS NOT NULL
GROUP BY country
ORDER BY click_count DESC;
```

### Daily Click Analytics

**SQLite:**
```sql
SELECT DATE(clicked_at) as date, COUNT(*) as clicks
FROM clicks
WHERE url_id = 1
GROUP BY DATE(clicked_at)
ORDER BY date DESC;
```

**PostgreSQL:**
```sql
SELECT DATE_TRUNC('day', clicked_at) as date, COUNT(*) as clicks
FROM clicks
WHERE url_id = 1
GROUP BY DATE_TRUNC('day', clicked_at)
ORDER BY date DESC;
```

## Performance Considerations

### Indexes
All critical indexes are included in the schema:
- `short_code` lookup is indexed for fast redirects (most common operation)
- `url_id` and composite `(url_id, clicked_at)` indexes support efficient analytics queries
- `clicked_at` index enables time-based filtering
- `country` index supports geographic analytics

### PostgreSQL Optimizations
The PostgreSQL schema includes:
- `SERIAL` type for auto-incrementing IDs
- `TIMESTAMPTZ` for timezone-aware timestamps
- Notes on optional partial indexes for recent data
- Suggestions for table partitioning at high volume

### SQLite Considerations
- Foreign key constraints must be explicitly enabled: `PRAGMA foreign_keys = ON;`
- Uses `INTEGER PRIMARY KEY AUTOINCREMENT` for auto-increment
- `CURRENT_TIMESTAMP` for default timestamps

## Database Compatibility

The base `schema.sql` file is designed to work with both databases with minimal changes:
- Uses SQL standard types where possible
- Includes comments for database-specific syntax
- Provides separate files for each database to avoid manual edits

## Integration with sqlc

These schemas are designed to work with [sqlc](https://github.com/sqlc-dev/sqlc) for type-safe SQL code generation.

Example `sqlc.yaml` configuration:
```yaml
version: "2"
sql:
  - engine: "postgresql"  # or "sqlite"
    queries: "queries/"
    schema: "docs/schema.postgres.sql"  # or schema.sqlite.sql
    gen:
      go:
        package: "db"
        out: "internal/infrastructure/db"
```

## Future Enhancements

Potential improvements for high-scale deployments:
- UUID primary keys for distributed systems
- Table partitioning for clicks (by month or week)
- Materialized views for common analytics queries
- Additional indexes based on actual query patterns
- Retention policies for old click data
