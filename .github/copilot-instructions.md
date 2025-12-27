# Copilot Instructions for mjr.wtf

## Project Overview

**mjr.wtf** is a URL shortener written in Go (1.24.2) following hexagonal architecture principles. The project is in early development with ~5,600 lines of Go code across 32 files. It supports both SQLite (development) and PostgreSQL (production) databases.

**Key Technologies:**
- Go 1.24.2
- SQLite 3 and PostgreSQL for persistence
- sqlc v1.30.0+ for type-safe SQL code generation
- goose v3.26.0 for database migrations
- golangci-lint v1.56.0 for linting

**Architecture:** Hexagonal (Ports & Adapters)
- `internal/domain/` - Core business logic, entities, and repository interfaces
- `internal/adapters/` - Repository implementations using sqlc-generated code
- `internal/infrastructure/` - Configuration and infrastructure concerns
- `internal/migrations/` - Database migrations (SQLite and PostgreSQL)
- `cmd/migrate/` - Migration CLI tool
- `cmd/server/` - HTTP server application
- `docs/` - Database schema documentation

## Preferred Copilot CLI Workflow (RPI-V)

For non-trivial work, follow **Research → Plan → Implement → Validate** and manage context deliberately:

- Keep context under ~60% (use `/context`).
- Clear between phases (`/clear`) after saving artifacts.
- Persist research/plan/validation notes via `/share` into a local `thoughts/` directory.

See the `rpi-workflow` skill for templates and a concrete flow.

## Critical Build Prerequisites

**ALWAYS run these commands in this exact order before building or testing:**

1. **Generate sqlc code first** (required for compilation):
   ```bash
   sqlc generate
   ```
   - This generates type-safe database code in `internal/adapters/repository/sqlc/{sqlite,postgres}/`
   - Without this, compilation will fail with "undefined: postgresrepo" and "undefined: sqliterepo" errors
   - Takes ~0.1 seconds

2. **Build the migrate tool** (if needed):
   ```bash
   make build-migrate
   # or: go build -o bin/migrate ./cmd/migrate
   ```
   - Takes ~18 seconds on first build (downloads dependencies)
   - Takes <1 second on subsequent builds
   - Creates `bin/migrate` binary

3. **Set DATABASE_URL environment variable** (for migration commands):
   ```bash
   # For SQLite (development)
   export DATABASE_URL=./database.db
   
   # For PostgreSQL (production)
   export DATABASE_URL=postgresql://user:password@localhost:5432/mjrwtf
   ```
   - Copy `.env.example` to `.env` and configure if using dotenv
   - Migration commands will fail without this

## Build Commands

```bash
# Clean build artifacts
make clean                    # Removes bin/ directory

# Build all binaries (server + migrate)
make build                    # Builds both bin/server and bin/migrate

# Build server binary (alternative: go build -o bin/server ./cmd/server)
make build-server             # Builds the HTTP server (bin/server)

# Build migrate tool (alternative: go build -o bin/migrate ./cmd/migrate)
make build-migrate            # Builds the migration tool (bin/migrate)

# Format code
make fmt                      # Takes ~0.08s, runs go fmt ./...

# Run go vet
make vet                      # Takes ~1.2s

# Run linter
make lint                     # Takes ~2.7s
# WARNING: Currently shows false positives about undefined postgresrepo/sqliterepo
# These can be ignored if sqlc generate was run and tests pass

# Run tests
make test                     # Takes ~2.5s
go test -v ./...              # Same as above
# PostgreSQL tests will SKIP if PostgreSQL not running (expected)
# SQLite tests run in-memory and should always pass

# Run all checks (fmt + vet + lint + test)
make check                    # Takes ~4-5s total
# Currently exits with error due to lint false positives - can be ignored if tests pass
```

## Test Execution Details

**Test timing:** ~2.5 seconds total
- PostgreSQL tests SKIP automatically if database not available (connection refused)
- SQLite tests run in temporary in-memory databases
- Tests create temporary databases with goose migrations automatically
- All tests should pass if sqlc generate was run

**Key test packages:**
- `internal/adapters/repository/` - Repository implementations (8 test files)
- `internal/domain/click/` - Click entity validation
- `internal/domain/url/` - URL entity validation
- `internal/infrastructure/config/` - Configuration loading

## Go Code Style Standards

### Naming Conventions
- **Interfaces**: Suffix with "er" (e.g., `URLRepository`, `ClickRepository`)
- **Errors**: Prefix with "Err" (e.g., `ErrURLNotFound`, `ErrDuplicateShortCode`)
- **Test functions**: `Test<Type>_<Method>_<Scenario>` (e.g., `TestURL_Validate_InvalidShortCode`)

### Comment Style
- Only comment non-obvious logic (explain **why**, not **what**)
- Package comments required for all packages
- Exported functions must have GoDoc comments starting with the function name
- Keep comments concise and up-to-date

### File Organization
- One entity type per file (e.g., `url.go`, `click.go`)
- Group related functions together
- Keep files under 300 lines when possible
- Order: types, constructors, methods, helpers

## Pull Request Guidelines

### Pre-PR Checklist
1. Run: `sqlc generate && make check`
2. Verify all tests pass locally
3. Update documentation if behavior changes
4. Add/update tests for new functionality
5. No hardcoded credentials or secrets
6. Breaking changes documented in PR description

### PR Description Template
- **What**: Brief summary of changes (1-2 sentences)
- **Why**: Business/technical motivation
- **Testing**: How changes were verified (tests added, manual testing)
- **Impact**: Breaking changes, migration needs, or "None"

## Security Checklist

Review before committing:

- [ ] No hardcoded credentials or secrets in code
- [ ] All user inputs validated in domain layer
- [ ] SQL injection prevented (sqlc handles this automatically)
- [ ] Sensitive data not logged (e.g., full URLs, user data)
- [ ] Rate limiting considered for public endpoints
- [ ] Authentication required for write operations
- [ ] Error messages don't leak sensitive information

## Version Compatibility

- **Go 1.24.2+** required (uses latest generics features)
- **sqlc 1.30.0+** required (older versions have breaking changes)
- **SQLite 3.x** required (any recent version)
- **PostgreSQL 12+** recommended for production
- **golangci-lint 1.56.0+** (older versions may not support Go 1.24)
- **goose v3.26.0+** for database migrations

## Quick Start for New Contributors

### Prerequisites
- Go 1.24.2+
- sqlc (install: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0`)
- golangci-lint (optional but recommended)

### Setup Steps
1. Clone repository: `git clone <repo>`
2. Copy environment file: `cp .env.example .env`
3. Generate database code: `sqlc generate`
4. Generate templates: `templ generate` (or install templ if needed)
5. Build binaries: `make build` (builds both server and migrate tool)
6. Run tests: `make test` (PostgreSQL tests will skip if database unavailable)

### Making Changes
1. Create feature branch: `git checkout -b feature/your-feature`
2. Make changes following architecture patterns
3. If you modified SQL queries: `sqlc generate`
4. Before committing: `sqlc generate && make check`
5. Commit with descriptive message
6. Push and create pull request

## Troubleshooting

### "undefined: postgresrepo" or "undefined: sqliterepo"
**Cause:** sqlc code not generated  
**Fix:** Run `sqlc generate`

### "bin/migrate: command not found"
**Cause:** Migration tool not built  
**Fix:** Run `make build-migrate`

### "failed to open database" or migration errors
**Cause:** DATABASE_URL environment variable not set  
**Fix:** `export DATABASE_URL=./database.db` (or use PostgreSQL connection string)

### Linter shows errors but tests pass
**Cause:** Known false positives with golangci-lint not recognizing sqlc-generated packages  
**Fix:** Verify code compiles (`go build ./...`) and tests pass (`make test`) - these are more reliable

### "UNIQUE constraint failed" in tests
**Cause:** Test data collision or missing cleanup  
**Fix:** Ensure each test uses unique data or properly cleans up after itself

## Database Migrations

**Migration commands REQUIRE DATABASE_URL to be set.**

```bash
# View migration status
make migrate-status
# or: ./bin/migrate status

# Apply all pending migrations
make migrate-up
# or: ./bin/migrate up

# Rollback last migration
make migrate-down
# or: ./bin/migrate down

# Create new migration
make migrate-create NAME=add_feature
# Creates timestamped file in internal/migrations/{sqlite,postgres}/

# Reset all migrations (DESTROYS DATA)
make migrate-reset
# or: ./bin/migrate reset
```

**Migration files location:**
- SQLite: `internal/migrations/sqlite/*.sql`
- PostgreSQL: `internal/migrations/postgres/*.sql`
- Migrations are embedded in the binary via `internal/migrations/migrations.go`

## Known Issues and Workarounds

### Linter False Positives
The linter reports "undefined: postgresrepo", "undefined: sqliterepo", and "undefined: goose" errors even when code builds and tests pass. This is a known issue with golangci-lint not properly recognizing the sqlc-generated packages.

**Workaround:** Verify code compiles and tests pass instead:
```bash
sqlc generate    # Ensure generated code is up to date
go build ./...   # Should succeed
go test ./...    # Should pass (with PostgreSQL tests skipped)
```



## Project Structure

```
.
├── .github/
│   └── renovate.json          # Renovate dependency management config
├── cmd/
│   ├── migrate/               # Migration CLI tool
│   │   └── main.go
│   └── server/                # HTTP server application
│       └── main.go
├── docs/
│   ├── README.md              # Detailed database schema documentation
│   ├── schema.sql             # Base schema (requires manual DB-specific edits)
│   ├── schema.sqlite.sql      # SQLite-ready schema
│   └── schema.postgres.sql    # PostgreSQL-ready schema
├── internal/
│   ├── domain/                # Core business logic (hexagonal architecture)
│   │   ├── README.md          # Detailed domain layer documentation
│   │   ├── url/               # URL entity, validation, repository interface
│   │   │   ├── url.go
│   │   │   ├── url_test.go
│   │   │   ├── errors.go
│   │   │   └── repository.go
│   │   └── click/             # Click entity, validation, repository interface
│   │       ├── click.go
│   │       ├── click_test.go
│   │       ├── errors.go
│   │       └── repository.go
│   ├── adapters/
│   │   └── repository/        # Repository implementations using sqlc
│   │       ├── sqlc/
│   │       │   ├── sqlite/    # Generated by sqlc (DO NOT EDIT)
│   │       │   │   ├── db.go
│   │       │   │   ├── models.go
│   │       │   │   ├── queries.sql
│   │       │   │   ├── queries.sql.go
│   │       │   │   └── querier.go
│   │       │   └── postgres/  # Generated by sqlc (DO NOT EDIT)
│   │       │       └── (same files as sqlite)
│   │       ├── url_repository.go
│   │       ├── url_repository_sqlite_test.go
│   │       ├── url_repository_postgres_test.go
│   │       ├── click_repository.go
│   │       ├── click_repository_sqlite_test.go
│   │       └── click_repository_postgres_test.go
│   ├── infrastructure/
│   │   └── config/            # Configuration management
│   │       ├── config.go
│   │       └── config_test.go
│   ├── migrations/
│   │   ├── migrations.go      # Embeds migration files into binary
│   │   ├── sqlite/
│   │   │   └── 00001_initial_schema.sql
│   │   └── postgres/
│   │       └── 00001_initial_schema.sql
│   └── application/           # Application services (empty placeholder)
├── .env.example               # Environment variable template
├── .gitignore                 # Ignores bin/, *.test, .env, go.work
├── go.mod                     # Go 1.24.2, requires goose, godotenv, lib/pq, go-sqlite3
├── Makefile                   # Build, test, lint, migration targets
├── README.md                  # Migration-focused user documentation
└── sqlc.yaml                  # sqlc configuration for both SQLite and PostgreSQL
```

## Domain Model Summary

**URL entity** (`internal/domain/url/`):
- Fields: ID, ShortCode, OriginalURL, CreatedAt, CreatedBy
- Validation: ShortCode must be 3-20 chars (alphanumeric, underscore, hyphen); OriginalURL must be valid http/https URL
- Repository operations: Create, FindByShortCode, Delete, List, ListByCreatedByAndTimeRange

**Click entity** (`internal/domain/click/`):
- Fields: ID, URLID, ClickedAt, Referrer, Country (ISO 3166-1 alpha-2), UserAgent
- Repository operations: Record, GetStatsByURL, GetStatsByURLAndTimeRange, GetTotalClickCount, GetClicksByCountry

**Error handling:** Domain-specific errors defined in `errors.go` files (e.g., `ErrURLNotFound`, `ErrDuplicateShortCode`, `ErrInvalidShortCode`)

## Configuration

Environment variables (see `.env.example`):
- `DATABASE_URL` - Database connection string (REQUIRED for migrations)
- `SERVER_PORT` - HTTP server port (default: 8080)
- `AUTH_TOKEN` - API authentication token
- `DISCORD_WEBHOOK_URL` - Optional Discord notifications
- `GEOIP_ENABLED` - Enable GeoIP tracking (default: false)
- `GEOIP_DATABASE` - GeoIP database path (required if enabled)

## Important Notes for AI Coding Agents

1. **ALWAYS run `sqlc generate` before building or testing** - This is the most common cause of compilation failures
2. **Trust that tests pass even if linter shows errors** - The linter has known false positives
3. **Don't try to run `make build`** - There's no main application yet, only the migrate tool
4. **PostgreSQL tests skipping is normal** - They only run if PostgreSQL is available on localhost:5432
5. **Set DATABASE_URL before migration commands** - Export it or use `.env` file
6. **Files in `internal/adapters/repository/sqlc/` are generated** - Never edit them directly; modify `sqlc.yaml` or query files instead
7. **Migrations are embedded** - Changes to migration files require rebuilding the migrate tool
8. **Use hexagonal architecture patterns** - Keep domain logic in `internal/domain/`, implementations in `internal/adapters/`
9. **No GitHub Actions workflows exist yet** - All validation must be done locally
10. **Only perform searches if these instructions are incomplete or incorrect** - This should provide everything needed for most changes
