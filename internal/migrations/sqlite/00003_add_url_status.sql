-- +goose Up
-- +goose StatementBegin
-- Stores periodic destination status + archive lookup results for each URL.
-- This keeps operational metadata separate from the core urls table.
CREATE TABLE IF NOT EXISTS url_status (
    url_id INTEGER PRIMARY KEY,

    last_checked_at TIMESTAMP,
    last_status_code INTEGER,

    -- Non-NULL means the destination was confirmed gone (HTTP 404/410).
    gone_at TIMESTAMP,

    -- Archived snapshot URL (e.g. Wayback closest snapshot).
    archive_url TEXT,
    archive_checked_at TIMESTAMP,

    FOREIGN KEY (url_id) REFERENCES urls(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_url_status_gone_at ON url_status(gone_at);
CREATE INDEX IF NOT EXISTS idx_url_status_last_checked_at ON url_status(last_checked_at);

-- archive_checked_at is only populated for URLs that are confirmed gone and have
-- archive lookups enabled. We still index it because checker queries filter/order
-- on archive_checked_at to find never-checked or oldest-checked archive rows.
CREATE INDEX IF NOT EXISTS idx_url_status_archive_checked_at ON url_status(archive_checked_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_url_status_archive_checked_at;
DROP INDEX IF EXISTS idx_url_status_last_checked_at;
DROP INDEX IF EXISTS idx_url_status_gone_at;
DROP TABLE IF EXISTS url_status;
-- +goose StatementEnd
