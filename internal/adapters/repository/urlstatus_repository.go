package repository

import (
	"database/sql"

	"github.com/matt-riley/mjrwtf/internal/domain/urlstatus"
)

// URLStatusRepository is an interface implemented by this repo's SQLite repository adapters.
type URLStatusRepository interface {
	urlstatus.Repository
}

// urlStatusRepositoryBase provides common functionality for URL status repositories.
type urlStatusRepositoryBase struct {
	db *sql.DB
}
