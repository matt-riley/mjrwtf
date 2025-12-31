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
WHERE created_by = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAllURLs :many
SELECT id, short_code, original_url, created_at, created_by
FROM urls
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListURLsByCreatedByAndTimeRange :many
SELECT id, short_code, original_url, created_at, created_by
FROM urls
WHERE created_by = ?
  AND created_at >= ?
  AND created_at <= ?
ORDER BY created_at DESC;

-- name: CountURLs :one
SELECT COUNT(*) as count
FROM urls;

-- name: CountURLsByCreatedBy :one
SELECT COUNT(*) as count
FROM urls
WHERE created_by = ?;

-- ============================================================================
-- URL Status Queries
-- ============================================================================

-- name: GetURLStatusByURLID :one
SELECT url_id, last_checked_at, last_status_code, gone_at, archive_url, archive_checked_at
FROM url_status
WHERE url_id = ?;

-- name: UpsertURLStatus :exec
INSERT INTO url_status (
    url_id,
    last_checked_at,
    last_status_code,
    gone_at,
    archive_url,
    archive_checked_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(url_id) DO UPDATE SET
    last_checked_at = excluded.last_checked_at,
    last_status_code = excluded.last_status_code,
    gone_at = excluded.gone_at,
    archive_url = excluded.archive_url,
    archive_checked_at = excluded.archive_checked_at;

-- name: ListURLsDueForStatusCheck :many
SELECT
    u.id AS url_id,
    u.short_code,
    u.original_url,
    us.last_checked_at,
    us.last_status_code,
    us.gone_at,
    us.archive_url,
    us.archive_checked_at
FROM urls u
LEFT JOIN url_status us ON us.url_id = u.id
WHERE us.last_checked_at IS NULL
   OR (us.gone_at IS NULL AND us.last_checked_at <= ?)
   OR (us.gone_at IS NOT NULL AND us.last_checked_at <= ?)
ORDER BY COALESCE(us.last_checked_at, u.created_at) ASC
LIMIT ?;

-- ============================================================================
-- Click Queries
-- ============================================================================

-- name: RecordClick :one
INSERT INTO clicks (url_id, clicked_at, referrer, referrer_domain, country, user_agent)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, url_id, clicked_at, referrer, referrer_domain, country, user_agent;

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
ORDER BY count DESC
LIMIT 10;

-- name: GetClicksByDate :many
SELECT CAST(DATE(clicked_at) AS TEXT) as date, COUNT(*) as count
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
ORDER BY count DESC
LIMIT 10;
