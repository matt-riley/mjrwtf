-- ============================================================================
-- URL Queries
-- ============================================================================

-- name: CreateURL :one
INSERT INTO urls (short_code, original_url, created_at, created_by)
VALUES (?, ?, ?, ?)
RETURNING id, short_code, original_url, created_at, created_by;

-- name: FindURLByShortCode :one
SELECT id, short_code, original_url, created_at, created_by
FROM urls
WHERE short_code = ?;

-- name: DeleteURLByShortCode :exec
DELETE FROM urls
WHERE short_code = ?;

-- name: ListURLs :many
SELECT id, short_code, original_url, created_at, created_by
FROM urls
WHERE (? = '' OR created_by = ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListURLsByCreatedByAndTimeRange :many
SELECT id, short_code, original_url, created_at, created_by
FROM urls
WHERE created_by = ?
  AND created_at >= ?
  AND created_at <= ?
ORDER BY created_at DESC;

-- ============================================================================
-- Click Queries
-- ============================================================================

-- name: RecordClick :one
INSERT INTO clicks (url_id, clicked_at, referrer, country, user_agent)
VALUES (?, ?, ?, ?, ?)
RETURNING id, url_id, clicked_at, referrer, country, user_agent;

-- name: GetTotalClickCount :one
SELECT COUNT(*) as count
FROM clicks
WHERE url_id = ?;

-- name: GetClicksByCountry :many
SELECT country, COUNT(*) as count
FROM clicks
WHERE url_id = ?
  AND country IS NOT NULL
  AND country != ''
GROUP BY country
ORDER BY count DESC;

-- name: GetClicksByReferrer :many
SELECT referrer, COUNT(*) as count
FROM clicks
WHERE url_id = ?
  AND referrer IS NOT NULL
  AND referrer != ''
GROUP BY referrer
ORDER BY count DESC;

-- name: GetClicksByDate :many
SELECT DATE(clicked_at) as date, COUNT(*) as count
FROM clicks
WHERE url_id = ?
GROUP BY date
ORDER BY date DESC;

-- name: GetTotalClickCountInTimeRange :one
SELECT COUNT(*) as count
FROM clicks
WHERE url_id = ?
  AND clicked_at >= ?
  AND clicked_at <= ?;

-- name: GetClicksByCountryInTimeRange :many
SELECT country, COUNT(*) as count
FROM clicks
WHERE url_id = ?
  AND clicked_at >= ?
  AND clicked_at <= ?
  AND country IS NOT NULL
  AND country != ''
GROUP BY country
ORDER BY count DESC;

-- name: GetClicksByReferrerInTimeRange :many
SELECT referrer, COUNT(*) as count
FROM clicks
WHERE url_id = ?
  AND clicked_at >= ?
  AND clicked_at <= ?
  AND referrer IS NOT NULL
  AND referrer != ''
GROUP BY referrer
ORDER BY count DESC;

-- ============================================================================
-- Session Queries
-- ============================================================================

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent;

-- name: GetSessionByID :one
SELECT id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent
FROM sessions
WHERE id = ?;

-- name: ListSessionsByUserID :many
SELECT id, user_id, created_at, expires_at, last_activity_at, ip_address, user_agent
FROM sessions
WHERE user_id = ?
ORDER BY last_activity_at DESC;

-- name: UpdateSessionActivity :exec
UPDATE sessions
SET last_activity_at = ?
WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions
WHERE user_id = ?;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at < datetime('now');

-- name: DeleteIdleSessions :execrows
DELETE FROM sessions
WHERE last_activity_at < ?;
