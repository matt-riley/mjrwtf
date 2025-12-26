package database

import (
	"fmt"
	"strings"
)

const (
	SQLiteMaxOpenConns  = 1
	SQLiteBusyTimeoutMs = 5000
)

// NormalizeSQLiteDSN ensures our default SQLite settings are applied via DSN parameters.
// If a caller already specifies _journal_mode, we don't override it.
func NormalizeSQLiteDSN(dsn string) string {
	parts := strings.SplitN(dsn, "?", 2)
	hasJournalMode := len(parts) == 2 && strings.Contains(parts[1], "_journal_mode")
	if hasJournalMode {
		return dsn
	}

	if len(parts) == 2 {
		return dsn + fmt.Sprintf("&_journal_mode=WAL&_busy_timeout=%d", SQLiteBusyTimeoutMs)
	}
	return dsn + fmt.Sprintf("?_journal_mode=WAL&_busy_timeout=%d", SQLiteBusyTimeoutMs)
}
