# Repository Adapter Instructions

Repository implementations using sqlc-generated code.

## Critical Rules

1. **Never Edit Generated Code**: Files in `sqlc/sqlite/` and `sqlc/postgres/` are auto-generated
2. **Modify Queries**: Edit `queries.sql` files, then run `sqlc generate`
3. **Test Both Databases**: Every repository must have SQLite and PostgreSQL tests
4. **Error Mapping**: Map database errors to domain errors

## Repository Implementation Pattern

```go
type urlRepository struct {
    sqliteQueries  *sqliterepo.Queries
    postgresQueries *postgresrepo.Queries
    dbType string
}

func (r *urlRepository) Create(ctx context.Context, url *url.URL) error {
    // Map domain entity to database params
    params := sqliterepo.CreateURLParams{
        ID:        url.ID,
        ShortCode: url.ShortCode,
        // ...
    }
    
    if err := r.sqliteQueries.CreateURL(ctx, params); err != nil {
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
- PostgreSQL tests should skip if database unavailable
- Apply migrations in test setup using goose
- Clean up test data in teardown

```go
func TestURLRepository_SQLite(t *testing.T) {
    db := setupTestSQLite(t)
    defer db.Close()
    
    repo := NewURLRepository(db, "sqlite")
    // test cases...
}
```
