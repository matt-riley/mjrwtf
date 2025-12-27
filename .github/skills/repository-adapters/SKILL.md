---
name: repository-adapters
description: Implement or modify database repository adapters using sqlc-generated queries for SQLite and PostgreSQL. Use when adding repository methods, changing SQL, or mapping DB errors to domain errors.
license: MIT
compatibility: Requires bash, git, Go, and sqlc v1.30.0+.
metadata:
  repo: mjrwtf
  runner: github-copilot-cli
  version: 1.2
allowed-tools: Bash(git:*) Bash(make:*) Bash(go:*) Bash(sqlc:*) Read
---

## Tooling assumptions

- Use a terminal runner with bash and git available.
- Prefer `make` targets when available; fall back to direct CLI commands when needed.

## Repo conventions

- Domain interfaces live in `internal/domain/**/repository.go`.
- Implementations live in `internal/adapters/repository/`.
- Do **not** edit generated code under `internal/adapters/repository/sqlc/**`.

## Making a repository change (safe path)

1. Update the SQL queries:
   - SQLite: `internal/adapters/repository/sqlc/sqlite/queries.sql`
   - Postgres: `internal/adapters/repository/sqlc/postgres/queries.sql`
2. Regenerate code:

```bash
make generate
```

3. Update adapter implementation(s) in `internal/adapters/repository/*.go`.
4. Ensure you have tests for both:
   - SQLite tests should always run.
   - Postgres tests should **skip** if Postgres isn’t available.

## Error mapping guidance

- Map uniqueness/constraint violations to domain errors (e.g., duplicate short code).
- Wrap unknown DB errors with context (operation + entity) while preserving `%w`.
- Avoid leaking sensitive inputs (e.g., full original URLs) in error strings/logs.

## Verification

```bash
make test
```
