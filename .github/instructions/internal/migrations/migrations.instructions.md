# Database Migration Instructions (SQLite)

Goose migration files live in `internal/migrations/sqlite/`.

## Critical rules

1. Every migration must have `-- +goose Up` and `-- +goose Down`.
2. Prefer idempotent DDL where possible (`IF NOT EXISTS`).
3. Test `up` and `down` locally.

## Create / apply / rollback

```bash
make migrate-create NAME=add_feature_x
make build-migrate
export DATABASE_URL=./database.db
make migrate-up
make migrate-down
```

Note: Migrations are embedded into `bin/migrate` at build time, so rebuild it after changing migration files.
