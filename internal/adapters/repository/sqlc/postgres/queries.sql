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

-- ============================================================================
-- Click Queries
-- ============================================================================

-- name: RecordClick :one
INSERT INTO clicks (url_id, clicked_at, referrer, country, user_agent)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, url_id, clicked_at, referrer, country, user_agent;

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
ORDER BY count DESC;

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
ORDER BY count DESC;

-- ============================================================================
-- Session Queries
-- ============================================================================

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent;

-- name: GetSessionByID :one
SELECT id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent
FROM sessions
WHERE id = $1;

-- name: ListSessionsByUserID :many
SELECT id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent
FROM sessions
WHERE user_id = $1
ORDER BY last_activity_at DESC;

-- name: UpdateSessionActivity :exec
UPDATE sessions
SET last_activity_at = $1
WHERE id = $2;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions
WHERE user_id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at < NOW();

-- name: DeleteIdleSessions :execrows
DELETE FROM sessions
WHERE last_activity_at < $1;
