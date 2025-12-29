---
title: Database migrations
description: Running and managing SQLite migrations.
---

This project uses goose for SQLite migrations.

## Quick start

```bash
# Build the migrate tool
make build-migrate

# Apply migrations
export DATABASE_URL=./database.db
make migrate-up

# Check status
make migrate-status

# Roll back last migration
make migrate-down
```

Migration files live in:

- `internal/migrations/sqlite/`
