-- +goose Up
-- +goose StatementBegin
-- Add referrer_domain column to store parsed domain from referrer URL
-- This enables domain-level analytics aggregation while preserving full referrer
ALTER TABLE clicks ADD COLUMN referrer_domain VARCHAR(255);

-- Create index for efficient domain-based analytics queries
CREATE INDEX IF NOT EXISTS idx_clicks_referrer_domain ON clicks(referrer_domain);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop the index first
DROP INDEX IF EXISTS idx_clicks_referrer_domain;

-- Drop the referrer_domain column
ALTER TABLE clicks DROP COLUMN referrer_domain;
-- +goose StatementEnd
