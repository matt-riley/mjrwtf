package click

import (
	"testing"
)

func TestNewClick(t *testing.T) {
	tests := []struct {
		name            string
		urlID           int64
		referrer        string
		country         string
		userAgent       string
		wantErr         error
		expectedCountry string // Expected country after normalization
	}{
		{
			name:            "valid click with all fields",
			urlID:           1,
			referrer:        "https://google.com",
			country:         "US",
			userAgent:       "Mozilla/5.0",
			wantErr:         nil,
			expectedCountry: "US",
		},
		{
			name:            "valid click without referrer",
			urlID:           1,
			referrer:        "",
			country:         "GB",
			userAgent:       "Mozilla/5.0",
			wantErr:         nil,
			expectedCountry: "GB",
		},
		{
			name:            "valid click without country",
			urlID:           1,
			referrer:        "https://google.com",
			country:         "",
			userAgent:       "Mozilla/5.0",
			wantErr:         nil,
			expectedCountry: "",
		},
		{
			name:            "valid click without user agent",
			urlID:           1,
			referrer:        "https://google.com",
			country:         "CA",
			userAgent:       "",
			wantErr:         nil,
			expectedCountry: "CA",
		},
		{
			name:            "valid click with minimum fields",
			urlID:           1,
			referrer:        "",
			country:         "",
			userAgent:       "",
			wantErr:         nil,
			expectedCountry: "",
		},
		{
			name:            "invalid URL ID zero",
			urlID:           0,
			referrer:        "https://google.com",
			country:         "US",
			userAgent:       "Mozilla/5.0",
			wantErr:         ErrInvalidURLID,
			expectedCountry: "",
		},
		{
			name:            "invalid URL ID negative",
			urlID:           -1,
			referrer:        "https://google.com",
			country:         "US",
			userAgent:       "Mozilla/5.0",
			wantErr:         ErrInvalidURLID,
			expectedCountry: "",
		},
		{
			name:            "invalid country code too short",
			urlID:           1,
			referrer:        "https://google.com",
			country:         "U",
			userAgent:       "Mozilla/5.0",
			wantErr:         ErrInvalidCountryCode,
			expectedCountry: "",
		},
		{
			name:            "invalid country code too long",
			urlID:           1,
			referrer:        "https://google.com",
			country:         "USA",
			userAgent:       "Mozilla/5.0",
			wantErr:         ErrInvalidCountryCode,
			expectedCountry: "",
		},
		{
			name:            "invalid country code with spaces",
			urlID:           1,
			referrer:        "https://google.com",
			country:         "U ",
			userAgent:       "Mozilla/5.0",
			wantErr:         ErrInvalidCountryCode,
			expectedCountry: "",
		},
		{
			name:            "country code with leading and trailing spaces gets normalized",
			urlID:           1,
			referrer:        "https://google.com",
			country:         " US ",
			userAgent:       "Mozilla/5.0",
			wantErr:         nil,
			expectedCountry: "US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			click, err := NewClick(tt.urlID, tt.referrer, tt.country, tt.userAgent)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("NewClick() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClick() unexpected error = %v", err)
				return
			}

			if click.URLID != tt.urlID {
				t.Errorf("NewClick() URLID = %v, want %v", click.URLID, tt.urlID)
			}

			if click.Referrer != tt.referrer {
				t.Errorf("NewClick() Referrer = %v, want %v", click.Referrer, tt.referrer)
			}

			if click.Country != tt.expectedCountry {
				t.Errorf("NewClick() Country = %v, want %v", click.Country, tt.expectedCountry)
			}

			if click.UserAgent != tt.userAgent {
				t.Errorf("NewClick() UserAgent = %v, want %v", click.UserAgent, tt.userAgent)
			}

			if click.ClickedAt.IsZero() {
				t.Error("NewClick() ClickedAt should not be zero")
			}
		})
	}
}

func TestClick_Validate(t *testing.T) {
	tests := []struct {
		name    string
		click   *Click
		wantErr error
	}{
		{
			name: "valid click",
			click: &Click{
				URLID:   1,
				Country: "US",
			},
			wantErr: nil,
		},
		{
			name: "valid click without country",
			click: &Click{
				URLID:   1,
				Country: "",
			},
			wantErr: nil,
		},
		{
			name: "invalid URL ID zero",
			click: &Click{
				URLID:   0,
				Country: "US",
			},
			wantErr: ErrInvalidURLID,
		},
		{
			name: "invalid URL ID negative",
			click: &Click{
				URLID:   -1,
				Country: "US",
			},
			wantErr: ErrInvalidURLID,
		},
		{
			name: "invalid country code",
			click: &Click{
				URLID:   1,
				Country: "USA",
			},
			wantErr: ErrInvalidCountryCode,
		},
		{
			name: "invalid country code single character",
			click: &Click{
				URLID:   1,
				Country: "U",
			},
			wantErr: ErrInvalidCountryCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.click.Validate()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Click.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Click.Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name           string
		referrer       string
		expectedDomain string
	}{
		{
			name:           "https URL with domain",
			referrer:       "https://google.com",
			expectedDomain: "google.com",
		},
		{
			name:           "https URL with subdomain",
			referrer:       "https://www.google.com",
			expectedDomain: "www.google.com",
		},
		{
			name:           "https URL with path",
			referrer:       "https://google.com/search?q=test",
			expectedDomain: "google.com",
		},
		{
			name:           "http URL",
			referrer:       "http://example.com",
			expectedDomain: "example.com",
		},
		{
			name:           "URL with port",
			referrer:       "https://localhost:8080",
			expectedDomain: "localhost",
		},
		{
			name:           "URL with port and path",
			referrer:       "https://localhost:8080/path",
			expectedDomain: "localhost",
		},
		{
			name:           "empty referrer",
			referrer:       "",
			expectedDomain: "",
		},
		{
			name:           "malformed URL no scheme",
			referrer:       "google.com",
			expectedDomain: "",
		},
		{
			name:           "malformed URL invalid characters",
			referrer:       "ht!tp://invalid",
			expectedDomain: "",
		},
		{
			name:           "URL with fragment",
			referrer:       "https://example.com/page#section",
			expectedDomain: "example.com",
		},
		{
			name:           "URL with query parameters",
			referrer:       "https://example.com?param=value&foo=bar",
			expectedDomain: "example.com",
		},
		{
			name:           "social media URL",
			referrer:       "https://twitter.com/user/status/123",
			expectedDomain: "twitter.com",
		},
		{
			name:           "social media URL with subdomain",
			referrer:       "https://m.facebook.com/page",
			expectedDomain: "m.facebook.com",
		},
		{
			name:           "IP address",
			referrer:       "http://192.168.1.1",
			expectedDomain: "192.168.1.1",
		},
		{
			name:           "IPv6 address",
			referrer:       "http://[2001:db8::1]",
			expectedDomain: "2001:db8::1",
		},
		{
			name:           "URL with username and password",
			referrer:       "https://user:pass@example.com",
			expectedDomain: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain := extractDomain(tt.referrer)
			if domain != tt.expectedDomain {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.referrer, domain, tt.expectedDomain)
			}
		})
	}
}

func TestNewClick_ReferrerDomainExtraction(t *testing.T) {
	tests := []struct {
		name                   string
		referrer               string
		expectedReferrerDomain string
	}{
		{
			name:                   "valid https referrer",
			referrer:               "https://google.com/search",
			expectedReferrerDomain: "google.com",
		},
		{
			name:                   "valid http referrer",
			referrer:               "http://example.com",
			expectedReferrerDomain: "example.com",
		},
		{
			name:                   "referrer with subdomain",
			referrer:               "https://www.reddit.com/r/golang",
			expectedReferrerDomain: "www.reddit.com",
		},
		{
			name:                   "empty referrer (direct navigation)",
			referrer:               "",
			expectedReferrerDomain: "",
		},
		{
			name:                   "malformed referrer",
			referrer:               "not-a-valid-url",
			expectedReferrerDomain: "",
		},
		{
			name:                   "referrer with port",
			referrer:               "https://localhost:3000/page",
			expectedReferrerDomain: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			click, err := NewClick(1, tt.referrer, "US", "Mozilla/5.0")
			if err != nil {
				t.Fatalf("NewClick() unexpected error = %v", err)
			}

			if click.Referrer != tt.referrer {
				t.Errorf("NewClick() Referrer = %q, want %q", click.Referrer, tt.referrer)
			}

			if click.ReferrerDomain != tt.expectedReferrerDomain {
				t.Errorf("NewClick() ReferrerDomain = %q, want %q", click.ReferrerDomain, tt.expectedReferrerDomain)
			}
		})
	}
}
