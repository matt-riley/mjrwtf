---
title: Local development
description: Run mjr.wtf locally without Docker.
---

## Prerequisites

- Go toolchain (see `go.mod`)
- Code generation tools:
  - `sqlc` (v1.30.0+)
  - `templ`

```bash
# Install sqlc (for database code generation)
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

# Install templ (for template code generation)
go install github.com/a-h/templ/cmd/templ@latest
```

## Quick start

```bash
# Install dependencies
go mod download

# Generate code (sqlc + templ)
make generate

# Run database migrations
export DATABASE_URL=./database.db
make migrate-up

# Build and run the server
make build-server
./bin/server
```

## Verify

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```
