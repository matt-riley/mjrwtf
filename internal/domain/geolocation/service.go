// Package geolocation provides domain interfaces for IP geolocation services.
// This package defines the port (interface) for geolocation lookup following
// hexagonal architecture principles.
package geolocation

import "context"

// LookupService defines the interface for IP geolocation lookup operations.
// Following hexagonal architecture, this interface is defined in the domain layer
// and implemented by adapters (e.g., MaxMind GeoIP2).
type LookupService interface {
	// LookupCountry returns the ISO 3166-1 alpha-2 country code for the given IP address.
	// Returns an empty string if the lookup fails or the IP is invalid.
	// This is a best-effort operation - lookup failures are handled gracefully
	// by returning an empty string rather than propagating errors.
	//
	// Note: The context parameter is included for API consistency and future
	// extensibility, but current implementations may not use it for cancellation
	// since local database lookups are expected to be fast.
	LookupCountry(ctx context.Context, ipAddress string) string

	// Close releases any resources held by the service.
	Close() error
}
