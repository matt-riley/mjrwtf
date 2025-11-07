# mjr.wtf - url shortener

A simple URL shortener, written in Go.

## Database Migrations

This project uses [goose](https://github.com/pressly/goose) for database migrations, supporting both SQLite and PostgreSQL.

### Prerequisites

Set the `DATABASE_URL` environment variable to your database connection string:

```bash
# For SQLite
export DATABASE_URL=./database.db

# For PostgreSQL
export DATABASE_URL=postgresql://user:password@localhost:5432/mjrwtf
```

Alternatively, you can copy `.env.example` to `.env` and configure it there.

### Migration Commands

The following Makefile targets are available for managing migrations:

```bash
# Apply all pending migrations
make migrate-up

# Rollback the most recent migration
make migrate-down

# Show migration status
make migrate-status

# Create a new migration file
make migrate-create NAME=add_new_feature

# Reset all migrations (caution: destroys data)
make migrate-reset
```

### Manual Migration Management

You can also use the migrate tool directly:

```bash
# Build the migrate tool
go build -o bin/migrate ./cmd/migrate

# Run migrations with explicit driver and URL
./bin/migrate -driver sqlite3 -url ./database.db up

# Run migrations for PostgreSQL
./bin/migrate -driver postgres -url "postgresql://user:pass@localhost/dbname" up

# Show help
./bin/migrate
```

### Migration Files

Migration files are located in:
- `internal/migrations/sqlite/` - SQLite-specific migrations
- `internal/migrations/postgres/` - PostgreSQL-specific migrations

Each migration consists of:
- An `.sql` file with `-- +goose Up` and `-- +goose Down` sections
- The "Up" section applies the migration
- The "Down" section reverts the migration

### Creating New Migrations

To create a new migration:

```bash
# For SQLite (default)
make migrate-create NAME=add_users_table

# For PostgreSQL
DATABASE_URL=postgresql://... make migrate-create NAME=add_users_table
```

This will create a new timestamped migration file in the appropriate directory.

### Embedded Migrations

Migrations are embedded in the binary at build time, so the migrate tool is self-contained and doesn't require external migration files at runtime.
