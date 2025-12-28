package database

import (
	"fmt"
	"strings"
)

const (
	// SQLiteMaxOpenConns is the recommended maximum number of open connections for SQLite.
	SQLiteMaxOpenConns = 1

	// SQLiteBusyTimeoutMs is the busy timeout, in milliseconds, applied via the SQLite DSN.
	SQLiteBusyTimeoutMs = 5000
)

// NormalizeSQLiteDSN ensures our default SQLite settings are applied via DSN parameters.
// If a caller already specifies _journal_mode, we don't override it.
func NormalizeSQLiteDSN(dsn string) string {
	parts := strings.SplitN(dsn, "?", 2)
	if len(parts) == 2 {
		for _, kv := range strings.Split(parts[1], "&") {
			key, _, _ := strings.Cut(kv, "=")
			if key == "_journal_mode" {
				return dsn
			}
		}
	}

	if len(parts) == 2 {
		return dsn + fmt.Sprintf("&_journal_mode=WAL&_busy_timeout=%d", SQLiteBusyTimeoutMs)
	}
	return dsn + fmt.Sprintf("?_journal_mode=WAL&_busy_timeout=%d", SQLiteBusyTimeoutMs)
}
