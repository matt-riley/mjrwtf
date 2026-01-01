package tui

import "testing"

func TestValidateHTTPURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{name: "empty", in: "", wantErr: true},
		{name: "spaces", in: "   ", wantErr: true},
		{name: "valid https", in: "https://example.com", wantErr: false},
		{name: "valid http", in: "http://example.com", wantErr: false},
		{name: "invalid scheme", in: "ftp://example.com", wantErr: true},
		{name: "missing scheme", in: "example.com", wantErr: true},
		{name: "malformed", in: "http://", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateHTTPURL(tc.in)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
