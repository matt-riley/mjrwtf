package database

import "testing"

func TestNormalizeSQLiteDSN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no query params",
			in:   "./db.sqlite",
			want: "./db.sqlite?_journal_mode=WAL&_busy_timeout=5000",
		},
		{
			name: "existing query params (no journal mode)",
			in:   "./db.sqlite?cache=shared",
			want: "./db.sqlite?cache=shared&_journal_mode=WAL&_busy_timeout=5000",
		},
		{
			name: "journal mode already set (single param)",
			in:   "./db.sqlite?_journal_mode=DELETE",
			want: "./db.sqlite?_journal_mode=DELETE",
		},
		{
			name: "journal mode already set (multiple params)",
			in:   "./db.sqlite?cache=shared&_journal_mode=WAL",
			want: "./db.sqlite?cache=shared&_journal_mode=WAL",
		},
		{
			name: "substring false positive in value",
			in:   "./db.sqlite?x=_journal_mode",
			want: "./db.sqlite?x=_journal_mode&_journal_mode=WAL&_busy_timeout=5000",
		},
		{
			name: "substring false positive in filename",
			in:   "./db__journal_mode.sqlite",
			want: "./db__journal_mode.sqlite?_journal_mode=WAL&_busy_timeout=5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeSQLiteDSN(tt.in); got != tt.want {
				t.Fatalf("NormalizeSQLiteDSN(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
