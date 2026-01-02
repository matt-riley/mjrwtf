# Repository Adapter Instructions

Repository implementations using sqlc-generated code.

## Critical Rules

1. **Never Edit Generated Code**: Files in `sqlc/sqlite/` are auto-generated
2. **Modify Queries**: Edit `queries.sql` files, then run `sqlc generate`
3. **Test Thoroughly**: Every repository must have comprehensive tests using in-memory SQLite
4. **Error Mapping**: Map database errors to domain errors

## Repository Implementation Pattern

```go
type urlRepository struct {
    queries *sqliterepo.Queries
}

func (r *urlRepository) Create(ctx context.Context, url *url.URL) error {
    // Map domain entity to database params
    params := sqliterepo.CreateURLParams{
        ID:        url.ID,
        ShortCode: url.ShortCode,
        // ...
    }
    
    if err := r.queries.CreateURL(ctx, params); err != nil {
        // Map database errors to domain errors
        if strings.Contains(err.Error(), "UNIQUE constraint") {
            return url.ErrDuplicateShortCode
        }
        return fmt.Errorf("failed to create URL: %w", err)
    }
    
    return nil
}
```

## Testing Pattern

- Use in-memory SQLite for fast, isolated tests
- Apply migrations in test setup using goose
- Clean up test data in teardown

```go
func TestURLRepository(t *testing.T) {
    db := setupTestSQLite(t)
    defer db.Close()
    
    repo := NewURLRepository(db)
    // test cases...
}
```
