package url

import (
	"errors"
	"testing"
)

func TestNewURL(t *testing.T) {
	tests := []struct {
		name        string
		shortCode   string
		originalURL string
		createdBy   string
		wantErr     error
	}{
		{
			name:        "valid URL",
			shortCode:   "abc123",
			originalURL: "https://example.com",
			createdBy:   "user1",
			wantErr:     nil,
		},
		{
			name:        "valid URL with path",
			shortCode:   "test-url",
			originalURL: "https://example.com/path/to/page",
			createdBy:   "api-key-123",
			wantErr:     nil,
		},
		{
			name:        "valid URL with query params",
			shortCode:   "query_test",
			originalURL: "https://example.com/page?foo=bar&baz=qux",
			createdBy:   "system",
			wantErr:     nil,
		},
		{
			name:        "empty short code",
			shortCode:   "",
			originalURL: "https://example.com",
			createdBy:   "user1",
			wantErr:     ErrEmptyShortCode,
		},
		{
			name:        "short code too short",
			shortCode:   "ab",
			originalURL: "https://example.com",
			createdBy:   "user1",
			wantErr:     ErrInvalidShortCode,
		},
		{
			name:        "short code too long",
			shortCode:   "abcdefghijklmnopqrstuvwxyz",
			originalURL: "https://example.com",
			createdBy:   "user1",
			wantErr:     ErrInvalidShortCode,
		},
		{
			name:        "short code with invalid characters",
			shortCode:   "abc@123",
			originalURL: "https://example.com",
			createdBy:   "user1",
			wantErr:     ErrInvalidShortCode,
		},
		{
			name:        "short code with spaces",
			shortCode:   "abc 123",
			originalURL: "https://example.com",
			createdBy:   "user1",
			wantErr:     ErrInvalidShortCode,
		},
		{
			name:        "empty original URL",
			shortCode:   "abc123",
			originalURL: "",
			createdBy:   "user1",
			wantErr:     ErrEmptyOriginalURL,
		},
		{
			name:        "invalid original URL format",
			shortCode:   "abc123",
			originalURL: "not-a-url",
			createdBy:   "user1",
			wantErr:     ErrMissingURLScheme,
		},
		{
			name:        "URL without scheme",
			shortCode:   "abc123",
			originalURL: "example.com",
			createdBy:   "user1",
			wantErr:     ErrMissingURLScheme,
		},
		{
			name:        "URL with invalid scheme",
			shortCode:   "abc123",
			originalURL: "ftp://example.com",
			createdBy:   "user1",
			wantErr:     ErrInvalidURLScheme,
		},
		{
			name:        "URL without host",
			shortCode:   "abc123",
			originalURL: "https://",
			createdBy:   "user1",
			wantErr:     ErrMissingURLHost,
		},
		{
			name:        "empty created by",
			shortCode:   "abc123",
			originalURL: "https://example.com",
			createdBy:   "",
			wantErr:     ErrInvalidCreatedBy,
		},
		{
			name:        "whitespace only created by",
			shortCode:   "abc123",
			originalURL: "https://example.com",
			createdBy:   "   ",
			wantErr:     ErrInvalidCreatedBy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := NewURL(tt.shortCode, tt.originalURL, tt.createdBy)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("NewURL() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewURL() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewURL() unexpected error = %v", err)
				return
			}

			if url.ShortCode != tt.shortCode {
				t.Errorf("NewURL() ShortCode = %v, want %v", url.ShortCode, tt.shortCode)
			}

			if url.OriginalURL != tt.originalURL {
				t.Errorf("NewURL() OriginalURL = %v, want %v", url.OriginalURL, tt.originalURL)
			}

			if url.CreatedBy != tt.createdBy {
				t.Errorf("NewURL() CreatedBy = %v, want %v", url.CreatedBy, tt.createdBy)
			}

			if url.CreatedAt.IsZero() {
				t.Error("NewURL() CreatedAt should not be zero")
			}
		})
	}
}

func TestValidateShortCode(t *testing.T) {
	tests := []struct {
		name      string
		shortCode string
		wantErr   error
	}{
		{
			name:      "valid alphanumeric",
			shortCode: "abc123",
			wantErr:   nil,
		},
		{
			name:      "valid with underscores",
			shortCode: "test_url",
			wantErr:   nil,
		},
		{
			name:      "valid with hyphens",
			shortCode: "test-url",
			wantErr:   nil,
		},
		{
			name:      "valid mixed",
			shortCode: "test_url-123",
			wantErr:   nil,
		},
		{
			name:      "valid minimum length",
			shortCode: "abc",
			wantErr:   nil,
		},
		{
			name:      "valid maximum length",
			shortCode: "12345678901234567890",
			wantErr:   nil,
		},
		{
			name:      "empty",
			shortCode: "",
			wantErr:   ErrEmptyShortCode,
		},
		{
			name:      "too short",
			shortCode: "ab",
			wantErr:   ErrInvalidShortCode,
		},
		{
			name:      "too long",
			shortCode: "123456789012345678901",
			wantErr:   ErrInvalidShortCode,
		},
		{
			name:      "with spaces",
			shortCode: "abc 123",
			wantErr:   ErrInvalidShortCode,
		},
		{
			name:      "with special characters",
			shortCode: "abc@123",
			wantErr:   ErrInvalidShortCode,
		},
		{
			name:      "with dots",
			shortCode: "abc.123",
			wantErr:   ErrInvalidShortCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShortCode(tt.shortCode)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidateShortCode() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateShortCode() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateOriginalURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		wantErr     error
	}{
		{
			name:        "valid https URL",
			originalURL: "https://example.com",
			wantErr:     nil,
		},
		{
			name:        "valid http URL",
			originalURL: "http://example.com",
			wantErr:     nil,
		},
		{
			name:        "valid URL with path",
			originalURL: "https://example.com/path/to/page",
			wantErr:     nil,
		},
		{
			name:        "valid URL with query params",
			originalURL: "https://example.com/page?foo=bar",
			wantErr:     nil,
		},
		{
			name:        "valid URL with fragment",
			originalURL: "https://example.com/page#section",
			wantErr:     nil,
		},
		{
			name:        "valid URL with port",
			originalURL: "https://example.com:8080",
			wantErr:     nil,
		},
		{
			name:        "empty URL",
			originalURL: "",
			wantErr:     ErrEmptyOriginalURL,
		},
		{
			name:        "URL without scheme",
			originalURL: "example.com",
			wantErr:     ErrMissingURLScheme,
		},
		{
			name:        "URL with invalid scheme",
			originalURL: "ftp://example.com",
			wantErr:     ErrInvalidURLScheme,
		},
		{
			name:        "URL with javascript scheme",
			originalURL: "javascript:alert('xss')",
			wantErr:     ErrInvalidURLScheme,
		},
		{
			name:        "URL with parse error",
			originalURL: "http://example.com/%zz",
			wantErr:     ErrInvalidOriginalURL,
		},
		{
			name:        "URL without host",
			originalURL: "https://",
			wantErr:     ErrMissingURLHost,
		},
		{
			name:        "URL with only scheme",
			originalURL: "http://",
			wantErr:     ErrMissingURLHost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOriginalURL(tt.originalURL)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateOriginalURL() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateOriginalURL() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateOriginalURL() unexpected error = %v", err)
			}
		})
	}
}

func TestURL_Validate(t *testing.T) {
	tests := []struct {
		name    string
		url     *URL
		wantErr error
	}{
		{
			name: "valid URL",
			url: &URL{
				ShortCode:   "abc123",
				OriginalURL: "https://example.com",
				CreatedBy:   "user1",
			},
			wantErr: nil,
		},
		{
			name: "invalid short code",
			url: &URL{
				ShortCode:   "ab",
				OriginalURL: "https://example.com",
				CreatedBy:   "user1",
			},
			wantErr: ErrInvalidShortCode,
		},
		{
			name: "invalid original URL",
			url: &URL{
				ShortCode:   "abc123",
				OriginalURL: "not-a-url",
				CreatedBy:   "user1",
			},
			wantErr: ErrMissingURLScheme,
		},
		{
			name: "empty created by",
			url: &URL{
				ShortCode:   "abc123",
				OriginalURL: "https://example.com",
				CreatedBy:   "",
			},
			wantErr: ErrInvalidCreatedBy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.url.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("URL.Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("URL.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("URL.Validate() unexpected error = %v", err)
			}
		})
	}
}
