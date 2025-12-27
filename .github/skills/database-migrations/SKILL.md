---
name: database-migrations
description: Create, apply, and rollback goose migrations for both SQLite and PostgreSQL using the embedded migrate tool. Use when changing persistent schema or adding indexes/constraints.
license: MIT
compatibility: Requires bash, git, Go, make, and DATABASE_URL set for target DB.
metadata:
  repo: mjrwtf
  runner: github-copilot-cli
  version: 1.2
allowed-tools: Bash(git:*) Bash(make:*) Bash(go:*) Bash(curl:*) Read
---

## Tooling assumptions

- Use a terminal runner with bash and git available.
- Prefer `make` targets when available; fall back to direct CLI commands when needed.

## Key facts for this repo

- Migrations live in:
  - SQLite: `internal/migrations/sqlite/`
  - Postgres: `internal/migrations/postgres/`
- Migrations are **embedded** into the `bin/migrate` tool at build time.
- `DATABASE_URL` must point at the database you intend to migrate.

## Common workflows

### Apply migrations

```bash
export DATABASE_URL=./database.db
make migrate-up
```

For Postgres (often via docker compose):

```bash
export DATABASE_URL='postgresql://mjrwtf:INSECURE_CHANGE_ME@localhost:5432/mjrwtf'
make migrate-up
```

### Check migration status

```bash
make migrate-status
```

### Roll back one migration

```bash
make migrate-down
```

### Create a new migration

Create a migration (name should describe the schema change):

```bash
make migrate-create NAME=add_feature_x
```

Then implement **both** versions:

- `internal/migrations/sqlite/<timestamp>_add_feature_x.sql`
- `internal/migrations/postgres/<timestamp>_add_feature_x.sql`

Each file must include `-- +goose Up` and `-- +goose Down` sections.

## Checklist for new schema changes

1. Add/modify migrations for **both** databases.
2. If the schema affects sqlc models, ensure `sqlc.yaml` references the needed schema files.
3. Run:

```bash
make generate
make test
```

## Common pitfalls

- Forgetting to implement the Postgres variant (or the Down section).
- Changing migration files but not rebuilding `bin/migrate` (they’re embedded).
