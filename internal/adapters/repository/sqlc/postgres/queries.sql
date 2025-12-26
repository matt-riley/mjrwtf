-- ============================================================================
-- URL Queries
-- ============================================================================

-- name: CreateURL :one
INSERT INTO urls (short_code, original_url, created_at, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, short_code, original_url, created_at, created_by;

-- name: FindURLByShortCode :one
SELECT id, short_code, original_url, created_at, created_by
FROM urls
WHERE short_code = $1;

-- name: DeleteURLByShortCode :exec
DELETE FROM urls
WHERE short_code = $1;

-- name: ListURLs :many
SELECT id, short_code, original_url, created_at, created_by
FROM urls
WHERE ($1 = '' OR created_by = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListURLsByCreatedByAndTimeRange :many
SELECT id, short_code, original_url, created_at, created_by
FROM urls
WHERE created_by = $1
  AND created_at >= $2
  AND created_at <= $3
ORDER BY created_at DESC;

-- name: CountURLsByCreatedBy :one
-- Parameters: created_by_filter (pass empty string to count all URLs)
SELECT COUNT(*) as count
FROM urls
WHERE (sqlc.arg(created_by_filter) = '' OR created_by = sqlc.arg(created_by_filter));

-- ============================================================================
-- Click Queries
-- ============================================================================

-- name: RecordClick :one
INSERT INTO clicks (url_id, clicked_at, referrer, referrer_domain, country, user_agent)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, url_id, clicked_at, referrer, referrer_domain, country, user_agent;

-- name: GetTotalClickCount :one
SELECT COUNT(*) as count
FROM clicks
WHERE url_id = $1;

-- name: GetClicksByCountry :many
SELECT country, COUNT(*) as count
FROM clicks
WHERE url_id = $1
  AND country IS NOT NULL
  AND country != ''
GROUP BY country
ORDER BY count DESC;

-- name: GetClicksByReferrer :many
SELECT referrer, COUNT(*) as count
FROM clicks
WHERE url_id = $1
  AND referrer IS NOT NULL
  AND referrer != ''
GROUP BY referrer
ORDER BY count DESC
LIMIT 10;

-- name: GetClicksByDate :many
SELECT DATE(clicked_at) as date, COUNT(*) as count
FROM clicks
WHERE url_id = $1
GROUP BY date
ORDER BY date DESC;

-- name: GetTotalClickCountInTimeRange :one
SELECT COUNT(*) as count
FROM clicks
WHERE url_id = $1
  AND clicked_at >= $2
  AND clicked_at <= $3;

-- name: GetClicksByCountryInTimeRange :many
SELECT country, COUNT(*) as count
FROM clicks
WHERE url_id = $1
  AND clicked_at >= $2
  AND clicked_at <= $3
  AND country IS NOT NULL
  AND country != ''
GROUP BY country
ORDER BY count DESC;

-- name: GetClicksByReferrerInTimeRange :many
SELECT referrer, COUNT(*) as count
FROM clicks
WHERE url_id = $1
  AND clicked_at >= $2
  AND clicked_at <= $3
  AND referrer IS NOT NULL
  AND referrer != ''
GROUP BY referrer
ORDER BY count DESC
LIMIT 10;
