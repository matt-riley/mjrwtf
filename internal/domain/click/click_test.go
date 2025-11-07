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
