---
name: code-generation
description: Run and troubleshoot project code generation (sqlc + templ). Use when changing SQL queries, migrations/schema inputs, or Templ templates, and before building/tests if generated code might be stale.
license: MIT
compatibility: Requires bash, git, Go, make, sqlc v1.30.0+, and templ.
metadata:
  repo: mjrwtf
  runner: github-copilot-cli
  version: 1.2
allowed-tools: Bash(git:*) Bash(make:*) Bash(go:*) Bash(sqlc:*) Bash(templ:*) Bash(curl:*) Read
---

## Tooling assumptions

- Use a terminal runner with bash and git available.
- Prefer `make` targets when available; fall back to direct CLI commands when needed.

## What this skill covers

This repo relies on generated code for:

- **sqlc**: generates DB access code into `internal/adapters/repository/sqlc/{sqlite,postgres}/`
- **templ**: generates Go code from `.templ` templates (if present)

## Standard commands

- Generate everything:

```bash
make generate
```

- Equivalent manual commands:

```bash
sqlc generate
templ generate
```

## When to run generation

Run `make generate` when you change:

- `internal/adapters/repository/sqlc/**/queries.sql`
- `sqlc.yaml`
- migration files referenced by `sqlc.yaml` under `schema:`
- any `.templ` files

## Troubleshooting

- `sqlc not installed`:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
```

- `templ not installed`:

```bash
go install github.com/a-h/templ/cmd/templ@latest
```

- Build/test failures mentioning missing `postgresrepo` / `sqliterepo` usually mean generation didn’t run. Re-run `make generate` and retry.
