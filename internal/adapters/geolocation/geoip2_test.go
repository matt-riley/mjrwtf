package geolocation

import (
	"context"
	"os"
	"testing"
)

func TestGeoIP2Service_LookupCountry_InvalidIP(t *testing.T) {
	// Skip if GeoIP database is not available
	dbPath := os.Getenv("GEOIP_TEST_DATABASE")
	if dbPath == "" {
		t.Skip("GEOIP_TEST_DATABASE environment variable not set, skipping GeoIP2 tests")
	}

	service, err := NewGeoIP2Service(dbPath)
	if err != nil {
		t.Skipf("Could not open GeoIP database at %s: %v", dbPath, err)
	}
	defer service.Close()

	tests := []struct {
		name      string
		ipAddress string
		want      string
	}{
		{
			name:      "empty IP address",
			ipAddress: "",
			want:      "",
		},
		{
			name:      "invalid IP address format",
			ipAddress: "not-an-ip",
			want:      "",
		},
		{
			name:      "incomplete IP address",
			ipAddress: "192.168",
			want:      "",
		},
		{
			name:      "IP with invalid characters",
			ipAddress: "192.168.1.abc",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.LookupCountry(context.Background(), tt.ipAddress)
			if got != tt.want {
				t.Errorf("LookupCountry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeoIP2Service_OpenInvalidDatabase(t *testing.T) {
	// Test that opening a non-existent database returns an error
	_, err := NewGeoIP2Service("/path/to/nonexistent/database.mmdb")
	if err == nil {
		t.Error("NewGeoIP2Service() expected error for non-existent database, got nil")
	}
}

func TestGeoIP2Service_OpenInvalidFile(t *testing.T) {
	// Create a temporary file that is not a valid MaxMind database
	tmpFile, err := os.CreateTemp("", "invalid-geoip-*.mmdb")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some invalid content
	_, err = tmpFile.WriteString("this is not a valid maxmind database")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Try to open it as a GeoIP database
	_, err = NewGeoIP2Service(tmpFile.Name())
	if err == nil {
		t.Error("NewGeoIP2Service() expected error for invalid database file, got nil")
	}
}

func TestGeoIP2Service_LookupCountry_WithDatabase(t *testing.T) {
	// Skip if GeoIP database is not available
	dbPath := os.Getenv("GEOIP_TEST_DATABASE")
	if dbPath == "" {
		t.Skip("GEOIP_TEST_DATABASE environment variable not set, skipping GeoIP2 tests")
	}

	service, err := NewGeoIP2Service(dbPath)
	if err != nil {
		t.Skipf("Could not open GeoIP database at %s: %v", dbPath, err)
	}
	defer service.Close()

	// Test with localhost IP - should return empty (private IP)
	t.Run("localhost IPv4", func(t *testing.T) {
		got := service.LookupCountry(context.Background(), "127.0.0.1")
		// Localhost typically doesn't have a country code in GeoIP databases
		// We just verify it doesn't error
		t.Logf("Localhost returned country: %q", got)
	})

	// Test with private IP - should return empty
	t.Run("private IPv4", func(t *testing.T) {
		got := service.LookupCountry(context.Background(), "192.168.1.1")
		// Private IPs typically don't have a country code in GeoIP databases
		t.Logf("Private IP returned country: %q", got)
	})

	// Test with valid IPv6 localhost
	t.Run("localhost IPv6", func(t *testing.T) {
		got := service.LookupCountry(context.Background(), "::1")
		// Localhost typically doesn't have a country code
		t.Logf("IPv6 localhost returned country: %q", got)
	})
}
