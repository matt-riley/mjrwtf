package urlstatus

import (
	"context"
	"time"
)

// DueURL represents a URL that should be checked by the periodic status checker.
type DueURL struct {
	URLID       int64
	ShortCode   string
	OriginalURL string

	LastCheckedAt    *time.Time
	LastStatusCode   *int64
	GoneAt           *time.Time
	ArchiveURL       *string
	ArchiveCheckedAt *time.Time
}

// Repository defines persistence operations for URL status metadata.
type Repository interface {
	GetByURLID(ctx context.Context, urlID int64) (*URLStatus, error)
	Upsert(ctx context.Context, status *URLStatus) error
	ListDueForStatusCheck(ctx context.Context, aliveCutoff, goneCutoff time.Time, limit int) ([]*DueURL, error)
}
