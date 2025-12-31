package urlstatus

import "time"

// URLStatus stores periodic destination status + archive lookup results for a URL.
type URLStatus struct {
	URLID            int64
	LastCheckedAt    *time.Time
	LastStatusCode   *int64
	GoneAt           *time.Time
	ArchiveURL       *string
	ArchiveCheckedAt *time.Time
}

func (s *URLStatus) IsGone() bool {
	return s != nil && s.GoneAt != nil
}

// IsGoneStatusCode returns true when status code should be treated as "gone".
func IsGoneStatusCode(code int) bool {
	return code == 404 || code == 410
}
