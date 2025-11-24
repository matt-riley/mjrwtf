// Package geolocation provides adapters for IP geolocation services.
// This package implements the geolocation.LookupService interface defined
// in the domain layer using MaxMind GeoIP2 databases.
package geolocation

import (
	"context"
	"net"

	"github.com/matt-riley/mjrwtf/internal/domain/geolocation"
	"github.com/oschwald/geoip2-golang"
)

// geoIP2Service implements the geolocation.LookupService interface using MaxMind GeoIP2.
type geoIP2Service struct {
	db *geoip2.Reader
}

// NewGeoIP2Service creates a new GeoIP2 lookup service with the specified database path.
// Returns an error if the database cannot be opened.
func NewGeoIP2Service(databasePath string) (geolocation.LookupService, error) {
	db, err := geoip2.Open(databasePath)
	if err != nil {
		return nil, err
	}

	return &geoIP2Service{db: db}, nil
}

// LookupCountry returns the ISO 3166-1 alpha-2 country code for the given IP address.
// Returns an empty string if the lookup fails or the IP is invalid.
// This is a best-effort operation - failures result in empty string, not errors.
func (s *geoIP2Service) LookupCountry(_ context.Context, ipAddress string) (string, error) {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		// Invalid IP address format, return empty string
		return "", nil
	}

	record, err := s.db.Country(ip)
	if err != nil {
		// Lookup failed, return empty string (best effort)
		return "", nil
	}

	return record.Country.IsoCode, nil
}

// Close releases the GeoIP2 database resources.
func (s *geoIP2Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
