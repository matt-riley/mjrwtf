package geolocation

import (
	"context"
	"testing"
)

func TestNoopService_LookupCountry(t *testing.T) {
	service := NewNoopService()
	defer service.Close()

	tests := []struct {
		name      string
		ipAddress string
		want      string
	}{
		{
			name:      "valid IPv4 address",
			ipAddress: "8.8.8.8",
			want:      "",
		},
		{
			name:      "valid IPv6 address",
			ipAddress: "2001:4860:4860::8888",
			want:      "",
		},
		{
			name:      "localhost",
			ipAddress: "127.0.0.1",
			want:      "",
		},
		{
			name:      "empty address",
			ipAddress: "",
			want:      "",
		},
		{
			name:      "invalid address",
			ipAddress: "not-an-ip",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.LookupCountry(context.Background(), tt.ipAddress)
			if err != nil {
				t.Errorf("LookupCountry() error = %v, want no error", err)
				return
			}
			if got != tt.want {
				t.Errorf("LookupCountry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoopService_Close(t *testing.T) {
	service := NewNoopService()

	err := service.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want no error", err)
	}

	// Should be safe to call Close multiple times
	err = service.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v, want no error", err)
	}
}

func TestNewNoopService_ReturnsLookupService(t *testing.T) {
	service := NewNoopService()
	if service == nil {
		t.Error("NewNoopService() returned nil, want non-nil LookupService")
	}
}
