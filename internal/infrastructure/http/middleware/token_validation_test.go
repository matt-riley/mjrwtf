package middleware

import "testing"

func TestValidateTokenConstantTime(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		valid      []string
		wantMatch  bool
		wantConfig bool
	}{
		{
			name:       "no tokens configured",
			token:      "anything",
			valid:      nil,
			wantMatch:  false,
			wantConfig: false,
		},
		{
			name:       "single token match",
			token:      "a",
			valid:      []string{"a"},
			wantMatch:  true,
			wantConfig: true,
		},
		{
			name:       "single token no match",
			token:      "b",
			valid:      []string{"a"},
			wantMatch:  false,
			wantConfig: true,
		},
		{
			name:       "multi token match first",
			token:      "a",
			valid:      []string{"a", "b", "c"},
			wantMatch:  true,
			wantConfig: true,
		},
		{
			name:       "multi token match middle",
			token:      "b",
			valid:      []string{"a", "b", "c"},
			wantMatch:  true,
			wantConfig: true,
		},
		{
			name:       "multi token match last",
			token:      "c",
			valid:      []string{"a", "b", "c"},
			wantMatch:  true,
			wantConfig: true,
		},
		{
			name:       "multi token no match",
			token:      "d",
			valid:      []string{"a", "b", "c"},
			wantMatch:  false,
			wantConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch, gotConfigured := ValidateTokenConstantTime(tt.token, tt.valid)
			if gotMatch != tt.wantMatch {
				t.Fatalf("match = %v, want %v", gotMatch, tt.wantMatch)
			}
			if gotConfigured != tt.wantConfig {
				t.Fatalf("configured = %v, want %v", gotConfigured, tt.wantConfig)
			}
		})
	}
}
