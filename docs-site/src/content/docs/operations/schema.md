---
title: Database Schema
description: SQLite database schema reference for mjr.wtf.
---

This document describes the database schema for the mjr.wtf URL shortener application.

Note: The active Goose migrations (and sqlc config) are currently **SQLite-only**.

The schema is defined in migration files located in `internal/migrations/sqlite/`. For information on running migrations, see [Database Migrations](/operations/migrations/).

## Tables

### urls
Stores shortened URLs with their original destinations.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Primary key, auto-incrementing |
| `short_code` | VARCHAR(255) | Unique identifier for the shortened URL (e.g., "abc123") |
| `original_url` | TEXT | The destination URL |
| `created_at` | TIMESTAMP | When the URL was created |
| `created_by` | VARCHAR(255) | User/system that created this URL (API key, user ID, etc.) |

**Constraints:**
- UNIQUE on `short_code`
- NOT NULL on `short_code`, `original_url`, `created_at`, `created_by`

**Indexes:**
- `short_code` is automatically indexed via its UNIQUE constraint - Critical for redirect performance
- `idx_urls_created_by` on `created_by` - For filtering by creator
- `idx_urls_created_at` on `created_at` - For sorting/filtering by creation time

### clicks
Stores analytics data for each click on a shortened URL.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Primary key, auto-incrementing |
| `url_id` | INTEGER | Foreign key reference to `urls.id` |
| `clicked_at` | TIMESTAMP | When the click occurred |
| `referrer` | TEXT | HTTP Referer header (nullable) |
| `country` | VARCHAR(2) | ISO 3166-1 alpha-2 country code (nullable) |
| `user_agent` | TEXT | User-Agent header (nullable) |
| `referrer_domain` | VARCHAR(255) | Parsed domain from `referrer` for domain analytics (nullable) |

**Constraints:**
- FOREIGN KEY `url_id` REFERENCES `urls(id)` ON DELETE CASCADE
- NOT NULL on `url_id`, `clicked_at`

**Indexes:**
- `idx_clicks_url_id_clicked_at` on `(url_id, clicked_at)` - Composite index for time-based analytics (also serves queries filtering only on `url_id`)
- `idx_clicks_clicked_at` on `clicked_at` - For time-based filtering and sorting
- `idx_clicks_referrer_domain` on `referrer_domain` - For referrer domain analytics
- (Optional) `idx_clicks_country` on `country` - Add if country-based analytics is common

## Common Queries

### Redirect Query (most common)
```sql
SELECT original_url FROM urls WHERE short_code = 'abc123';
```

### Record a Click
```sql
INSERT INTO clicks (url_id, referrer, referrer_domain, country, user_agent)
VALUES (1, 'https://google.com/search', 'google.com', 'US', 'Mozilla/5.0...');
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
```sql
SELECT DATE(clicked_at) as date, COUNT(*) as clicks
FROM clicks
WHERE url_id = 1
GROUP BY DATE(clicked_at)
ORDER BY date DESC;
```

## Performance Considerations

### Indexes
All critical indexes are included in the schema:
- `short_code` lookup is automatically indexed via its UNIQUE constraint for fast redirects (most common operation)
- Composite `(url_id, clicked_at)` index supports efficient analytics queries and also serves queries filtering only on `url_id`
- `clicked_at` index enables time-based filtering

### SQLite Considerations
- Foreign key constraints must be explicitly enabled: `PRAGMA foreign_keys = ON;`
- Uses `INTEGER PRIMARY KEY AUTOINCREMENT` for auto-increment
- `CURRENT_TIMESTAMP` for default timestamps

## Integration with sqlc

This repo uses [sqlc](https://github.com/sqlc-dev/sqlc); the schema input for sqlc is the SQLite Goose migration files explicitly listed in the repo root `sqlc.yaml`.

Example configuration:
```yaml
version: "2"
sql:
  - name: "sqlite"
    engine: "sqlite"
    schema:
      - "internal/migrations/sqlite/00001_initial_schema.sql"
      - "internal/migrations/sqlite/00002_add_referrer_domain.sql"
    queries: "internal/adapters/repository/sqlc/sqlite/queries.sql"
    gen:
      go:
        package: "sqliterepo"
        out: "internal/adapters/repository/sqlc/sqlite"
```

Guidance:
- If you add a migration that changes tables used by queries, also add it to the `schema:` list in `sqlc.yaml` (keep migration order).
- Regenerate with `make generate` (or `sqlc generate`).
