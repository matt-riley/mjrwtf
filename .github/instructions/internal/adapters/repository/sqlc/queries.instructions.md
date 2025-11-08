# sqlc Query Instructions

SQL queries for type-safe code generation.

## Critical Rules

1. **Database Compatibility**: Write separate queries for SQLite and PostgreSQL
2. **Named Queries**: Use `-- name: QueryName :exec|one|many` format
3. **Parameters**: Use `?` for SQLite, `$1, $2` for PostgreSQL
4. **Null Handling**: Use `sqlc.narg()` for nullable parameters

## Query Naming Convention

- `Create<Entity>` - Insert operations
- `Get<Entity>By<Field>` - Select single row
- `List<Entity>` - Select multiple rows
- `Update<Entity>` - Update operations
- `Delete<Entity>` - Delete operations

## Example Patterns

### Insert (SQLite)
```sql
-- name: CreateURL :exec
INSERT INTO urls (id, short_code, original_url, created_at, created_by)
VALUES (?, ?, ?, ?, ?);
```

### Insert (PostgreSQL)
```sql
-- name: CreateURL :exec
INSERT INTO urls (id, short_code, original_url, created_at, created_by)
VALUES ($1, $2, $3, $4, $5);
```

### Select with Join
```sql
-- name: GetURLWithClickCount :one
SELECT u.*, COUNT(c.id) as click_count
FROM urls u
LEFT JOIN clicks c ON c.url_id = u.id
WHERE u.short_code = ?
GROUP BY u.id;
```

## After Modifying Queries

Always run: `sqlc generate`
