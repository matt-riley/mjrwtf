# Database Migration Instructions

Goose migration files for SQLite and PostgreSQL.

## Critical Rules

1. **Dual Migrations**: Create both SQLite and PostgreSQL versions
2. **Reversible**: Every migration must have UP and DOWN sections
3. **Test Both Directions**: Verify up and down migrations work
4. **Idempotent**: Use IF NOT EXISTS where appropriate

## Migration Structure

```sql
-- +goose Up
-- SQL in this section is executed when this migration is applied
CREATE TABLE IF NOT EXISTS table_name (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- SQL in this section is executed when this migration is rolled back
DROP TABLE IF EXISTS table_name;
```

## Database-Specific Considerations

### SQLite
- Use `TEXT` for UUIDs and strings
- Use `INTEGER` for timestamps (Unix epoch)
- Use `AUTOINCREMENT` for auto-increment IDs

### PostgreSQL
- Use `UUID` type for IDs
- Use `TIMESTAMP WITH TIME ZONE` for timestamps
- Use `SERIAL` or `BIGSERIAL` for auto-increment IDs

## Testing Migrations

```bash
# Apply migration
make migrate-up

# Verify schema
sqlite3 database.db ".schema"

# Test rollback
make migrate-down

# Verify rollback worked
```

## After Creating Migration

1. Rebuild migrate tool: `make build-migrate`
2. Test migration: `make migrate-up`
3. Update schema documentation in `docs/`
