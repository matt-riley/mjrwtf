# Test File Instructions

Go test files following project conventions.

## Test Naming Convention

- File: `<source>_test.go` (e.g., `url_test.go` for `url.go`)
- Function: `Test<Type>_<Method>_<Scenario>`
- Subtests: Use `t.Run()` with descriptive names

## Table-Driven Test Pattern

```go
func TestURL_Validate(t *testing.T) {
    tests := []struct {
        name    string
        url     *URL
        wantErr error
    }{
        {
            name: "valid URL with short code",
            url: &URL{
                ShortCode: "abc123",
                OriginalURL: "https://example.com",
            },
            wantErr: nil,
        },
        {
            name: "short code too short",
            url: &URL{
                ShortCode: "ab",
                OriginalURL: "https://example.com",
            },
            wantErr: ErrInvalidShortCode,
        },
        // More cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.url.Validate()
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Repository Test Pattern

### SQLite Test
```go
func TestURLRepository_Create_SQLite(t *testing.T) {
    db := setupTestSQLite(t)
    defer db.Close()
    
    repo := NewURLRepository(db, "sqlite")
    // test implementation
}

func setupTestSQLite(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    
    // Apply migrations
    require.NoError(t, goose.Up(db, "../../migrations/sqlite"))
    
    return db
}
```

### PostgreSQL Test (with skip)
```go
func TestURLRepository_Create_Postgres(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping PostgreSQL integration test")
    }
    
    db := setupTestPostgres(t)
    if db == nil {
        t.Skip("PostgreSQL not available")
        return
    }
    defer db.Close()
    
    // test implementation
}
```

## Test Best Practices

- Use `require` for setup assertions (fails fast)
- Use `assert` for test assertions (shows all failures)
- Clean up resources with `defer`
- Use subtests for related scenarios
- Test happy path AND error cases
