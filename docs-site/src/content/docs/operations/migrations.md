---
title: Database migrations
description: Running and managing SQLite migrations.
---

This project uses goose for SQLite migrations. Migrations are embedded into the `migrate` binary at build time.

## Quick start

```bash
# Apply all pending migrations (builds ./bin/migrate first)
export DATABASE_URL=./database.db
make migrate-up

# Check status
make migrate-status

# Roll back last migration
make migrate-down
```

## Key details

- Migration files live in `internal/migrations/sqlite/`.
- The `make migrate-*` targets rebuild `./bin/migrate` first (`make build-migrate`), because migrations are embedded into the binary.
- `DATABASE_URL` must be a **file path** for SQLite (URL-like values containing `://` are rejected).

## Creating a new migration

```bash
make migrate-create NAME=add_feature_x
# edit the generated file in internal/migrations/sqlite/
make migrate-up
```
