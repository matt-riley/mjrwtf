# GitHub Copilot Workspace Guide for mjr.wtf

This guide helps you work effectively with GitHub Copilot on this repo.

## Quick reference

Before making changes:

```bash
make generate
make test
```

If you changed OpenAPI:

```bash
make validate-openapi
```

## Repo realities (important)

- Database: **SQLite only** (migrations in `internal/migrations/sqlite/`).
- Codegen is required: sqlc + templ.
- Auth is bearer-token based (`AUTH_TOKENS` preferred), with optional session auth for the dashboard.

## How to write good issues for Copilot

Include:
- Clear acceptance criteria
- Files likely to change
- Commands to run (`make generate`, `make test`)

Example (good):

```markdown
Title: Add foo to URL analytics

## Acceptance Criteria
- [ ] Given an existing short URL, when requesting /api/urls/{shortCode}/analytics, then foo is included in the JSON response.

## Files
- internal/application/get_analytics.go
- internal/infrastructure/http/handlers/analytics_handler.go
- openapi.yaml
- tests under internal/infrastructure/http/server/

## Verification
- make generate
- make test
- make validate-openapi
```
