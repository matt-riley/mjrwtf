package geolocation

import (
	"context"

	"github.com/matt-riley/mjrwtf/internal/domain/geolocation"
)

// noopService implements the geolocation.LookupService interface as a no-op.
// This is used when GeoIP is disabled.
type noopService struct{}

// NewNoopService creates a new no-op lookup service that always returns empty strings.
// Use this when GeoIP is disabled.
func NewNoopService() geolocation.LookupService {
	return &noopService{}
}

// LookupCountry always returns an empty string since this is a no-op implementation.
func (s *noopService) LookupCountry(_ context.Context, _ string) string {
	return ""
}

// Close is a no-op for this implementation.
func (s *noopService) Close() error {
	return nil
}
