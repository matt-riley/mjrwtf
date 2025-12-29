# Copilot Instructions for mjr.wtf

## Project Overview

**mjr.wtf** is a URL shortener written in Go (see `go.mod` for the Go version). It currently uses **SQLite** for persistence.

Key technologies:
- Go
- SQLite (via `github.com/mattn/go-sqlite3`)
- sqlc (DB query codegen)
- templ (HTML template codegen)
- goose (migrations, embedded in `bin/migrate`)
- OpenAPI spec in `openapi.yaml`

## Critical workflows

### Code generation (required before build/test)

Run one of:

```bash
make generate
# or:
sqlc generate
templ generate
```

If generated code is stale, CI will fail the “Generate & verify generated code” job.

### Build / test

```bash
make test
# or:
go test ./...
```

### Migrations

Migrations live in `internal/migrations/sqlite/` and are embedded into the migrate binary.

```bash
make build-migrate
export DATABASE_URL=./database.db
./bin/migrate up
./bin/migrate status
```

### OpenAPI

`openapi.yaml` is the API contract.

```bash
make validate-openapi
```

## Auth & security notes

- Configure auth tokens via `AUTH_TOKENS` (preferred; comma-separated) or `AUTH_TOKEN` (legacy).
- `/metrics` is public by default; set `METRICS_AUTH_ENABLED=true` to require Bearer auth.
- Rate limiting applies to `/{shortCode}` and `/api/*`; limits are controlled by `REDIRECT_RATE_LIMIT_PER_MINUTE` and `API_RATE_LIMIT_PER_MINUTE`.
- Don’t log secrets (Authorization headers, tokens) or full URLs.
