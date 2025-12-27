package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/mattn/go-sqlite3"
)

func TestIsSQLiteUniqueConstraintError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "generic error returns false",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "SQLite unique constraint error returns true",
			err:  sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique},
			want: true,
		},
		{
			name: "SQLite constraint error without unique returns false",
			err:  sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintNotNull},
			want: false,
		},
		{
			name: "SQLite non-constraint error returns false",
			err:  sqlite3.Error{Code: sqlite3.ErrError},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSQLiteUniqueConstraintError(tt.err)
			if got != tt.want {
				t.Errorf("IsSQLiteUniqueConstraintError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUniqueConstraintError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "SQLite unique constraint error returns true",
			err:  sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique},
			want: true,
		},
		{
			name: "generic error returns false",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUniqueConstraintError(tt.err)
			if got != tt.want {
				t.Errorf("IsUniqueConstraintError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNoRowsError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "sql.ErrNoRows returns true",
			err:  sql.ErrNoRows,
			want: true,
		},
		{
			name: "wrapped sql.ErrNoRows returns true",
			err:  errors.Join(sql.ErrNoRows, errors.New("context")),
			want: true,
		},
		{
			name: "generic error returns false",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNoRowsError(tt.err)
			if got != tt.want {
				t.Errorf("IsNoRowsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapSQLError(t *testing.T) {
	var (
		errNotFound  = errors.New("not found")
		errDuplicate = errors.New("duplicate")
	)

	tests := []struct {
		name         string
		err          error
		notFoundErr  error
		duplicateErr error
		want         error
		wantWrapped  error
	}{
		{
			name:         "nil error returns nil",
			err:          nil,
			notFoundErr:  errNotFound,
			duplicateErr: errDuplicate,
			want:         nil,
		},
		{
			name:         "sql.ErrNoRows mapped to notFoundErr",
			err:          sql.ErrNoRows,
			notFoundErr:  errNotFound,
			duplicateErr: errDuplicate,
			want:         errNotFound,
		},
		{
			name:         "sql.ErrNoRows with nil notFoundErr wraps as database error",
			err:          sql.ErrNoRows,
			notFoundErr:  nil,
			duplicateErr: errDuplicate,
			wantWrapped:  sql.ErrNoRows,
		},
		{
			name:         "SQLite unique constraint mapped to duplicateErr",
			err:          sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique},
			notFoundErr:  errNotFound,
			duplicateErr: errDuplicate,
			want:         errDuplicate,
		},
		{
			name:         "generic error wraps as database error",
			err:          errors.New("connection failed"),
			notFoundErr:  errNotFound,
			duplicateErr: errDuplicate,
			wantWrapped:  errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapSQLError(tt.err, tt.notFoundErr, tt.duplicateErr)

			if tt.want != nil {
				if got != tt.want {
					t.Errorf("MapSQLError() = %v, want %v", got, tt.want)
				}
			} else if tt.wantWrapped != nil {
				// For wrapped errors, just check that an error was returned
				if got == nil {
					t.Errorf("MapSQLError() returned nil, expected wrapped error")
				}
			}
		})
	}
}

func TestMapSQLError_BothNil(t *testing.T) {
	// Test the case where both notFoundErr and duplicateErr are nil
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "sql.ErrNoRows with both nil",
			err:  sql.ErrNoRows,
		},
		{
			name: "unique constraint with both nil",
			err:  sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapSQLError(tt.err, nil, nil)
			if !errors.Is(got, tt.err) {
				t.Errorf("MapSQLError() should wrap original error, got %v", got)
			}
		})
	}
}
