-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Sessions Table
-- ============================================================================
-- Stores active user sessions for authentication and authorization
--
-- Note: Foreign key support must be enabled in application code when establishing
-- database connections using: PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS sessions (
    -- Primary key: session token (64 character random string)
    -- This is the actual token value that will be sent in cookies/headers
    -- Example: "a1b2c3d4e5f6...64chars" (securely generated random token)
    id TEXT PRIMARY KEY,
    
    -- User identifier associated with this session
    -- Initially stores "authenticated-user" as a placeholder
    -- Will be converted to a foreign key when users table is implemented
    user_id TEXT NOT NULL,
    
    -- Timestamp when the session was created
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Absolute expiration time (24 hours from creation)
    -- Session is invalid after this time regardless of activity
    expires_at TIMESTAMP NOT NULL,
    
    -- Last activity timestamp (updated on each request)
    -- Used to implement idle timeout (e.g., 30 minutes of inactivity)
    last_activity_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- IP address of the client when session was created
    -- Optional field for security monitoring and anomaly detection
    -- NULL if IP tracking is disabled
    ip_address TEXT,
    
    -- User-Agent header from the client
    -- Optional field for security monitoring (detect device changes)
    -- Limited to 500 characters to prevent abuse
    -- NULL if User-Agent tracking is disabled
    user_agent TEXT
);

-- Index for listing all sessions for a specific user
-- Supports queries like "show all active sessions for user X"
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- Index for session cleanup queries
-- Supports queries like "delete all expired sessions"
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Composite index for active session queries
-- Supports queries like "find active sessions for user X ordered by last activity"
-- The leading column (user_id) also makes this efficient for user_id-only queries
CREATE INDEX IF NOT EXISTS idx_sessions_user_id_last_activity ON sessions(user_id, last_activity_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd
