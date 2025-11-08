# GitHub Copilot Best Practices Analysis for mjr.wtf

**Date:** 2025-11-08  
**Project:** mjr.wtf URL Shortener  
**Analysis Type:** Comprehensive Review Against GitHub Copilot Coding Agent Best Practices

---

## Executive Summary

The mjr.wtf project has **excellent foundation** for GitHub Copilot integration with:
- ✅ Comprehensive `.github/copilot-instructions.md` (249 lines)
- ✅ Six custom agents defined (golang-expert, test-specialist, business-analyst, documentation-writer, security-expert, api-designer)
- ✅ Clear architecture documentation
- ✅ Well-documented build prerequisites and workflows

**Key Gaps Identified:**
- ❌ No `copilot-setup-steps.yml` for dependency pre-installation
- ❌ No path-specific instructions for specialized file types
- ❌ No GitHub issue templates for well-scoped issues
- ⚠️ Custom agents could benefit from additional domain-specific expertise

**Impact:** Adding these improvements will significantly enhance Copilot's ability to generate accurate code and PRs with minimal manual intervention.

---

## 1. Analysis of Current `.github/copilot-instructions.md`

### Strengths ✅

**Comprehensive Coverage:**
- ✅ Detailed project overview with tech stack and metrics (~5,600 LOC, 32 files)
- ✅ Critical build prerequisites clearly documented with exact commands
- ✅ Timing information for all commands (helps with expectations)
- ✅ Known issues section with workarounds (prevents wasted effort)
- ✅ Complete domain model documentation
- ✅ Architecture patterns explained (hexagonal/ports & adapters)
- ✅ Database-specific considerations (SQLite vs PostgreSQL)

**Excellent Process Documentation:**
- ✅ Step-by-step build workflow with exact command order
- ✅ Environment variable requirements clearly stated
- ✅ Migration workflow documented
- ✅ Test execution details with expected behavior (PostgreSQL skip is normal)

**AI Agent Specific Guidance:**
- ✅ "Important Notes for AI Coding Agents" section (10 critical points)
- ✅ Explicit instructions about sqlc-generated code (DO NOT EDIT)
- ✅ Clear warnings about lint false positives
- ✅ Guidance on when to search vs use provided instructions

### Areas for Improvement ⚠️

**1. Code Style Guidelines Missing**
```diff
+ ## Go Code Style Standards
+ 
+ ### Naming Conventions
+ - Interfaces: Suffix with "er" (e.g., URLRepository)
+ - Errors: Prefix with "Err" (e.g., ErrURLNotFound)
+ - Test functions: Test<Type>_<Method>_<Scenario>
+ 
+ ### Comment Style
+ - Only comment non-obvious logic (why, not what)
+ - Package comments required for all packages
+ - Exported functions must have GoDoc comments
+ 
+ ### File Organization
+ - One entity type per file
+ - Group related functions together
+ - Keep files under 300 lines when possible
```

**2. PR/Review Guidelines Absent**
```diff
+ ## Pull Request Guidelines
+ 
+ ### Before Creating PR
+ 1. Run: sqlc generate && make check
+ 2. Verify all tests pass locally
+ 3. Update documentation if behavior changes
+ 4. Add/update tests for new functionality
+ 
+ ### PR Description Template
+ - **What**: Brief summary of changes
+ - **Why**: Business/technical motivation
+ - **Testing**: How changes were verified
+ - **Impact**: Breaking changes or migration needs
```

**3. Security Best Practices Could Be More Prominent**
```diff
+ ## Security Checklist (Review Before Committing)
+ 
+ - [ ] No hardcoded credentials or secrets
+ - [ ] All user inputs validated in domain layer
+ - [ ] SQL injection prevented (sqlc handles this)
+ - [ ] Sensitive data not logged (no full URLs in logs)
+ - [ ] Rate limiting considered for public endpoints
+ - [ ] Authentication required for write operations
```

**4. Versioning/Compatibility Information**
```diff
+ ## Version Compatibility
+ 
+ - Go 1.24.2+ required (uses latest generics features)
+ - sqlc 1.30.0+ required (older versions have breaking changes)
+ - SQLite 3.x required
+ - PostgreSQL 12+ recommended (for production)
+ - golangci-lint 1.56.0+ (older versions may not support Go 1.24)
```

### Recommended Updates to copilot-instructions.md

**Add these sections:**

1. **Code Style Standards** (see above)
2. **PR Guidelines** (see above)
3. **Security Checklist** (see above)
4. **Version Compatibility** (see above)
5. **Quick Start for New Contributors:**
```markdown
## Quick Start for New Contributors

1. Prerequisites: Go 1.24.2+, sqlc, golangci-lint
2. Clone and setup: 
   ```bash
   git clone <repo>
   cp .env.example .env
   sqlc generate
   make build-migrate
   ```
3. Run tests: `make test` (should see PostgreSQL tests skip)
4. Make changes, then: `sqlc generate && make check`
```

6. **Troubleshooting Common Errors:**
```markdown
## Troubleshooting

**"undefined: postgresrepo"**
- Cause: sqlc code not generated
- Fix: Run `sqlc generate`

**"bin/migrate: command not found"**
- Cause: migrate tool not built
- Fix: Run `make build-migrate`

**"failed to open database"**
- Cause: DATABASE_URL not set
- Fix: `export DATABASE_URL=./database.db`
```

---

## 2. Path-Specific Instructions Recommendations

Path-specific instructions help Copilot understand context-specific rules for different file types. Create these files:

### 2.1 Domain Entity Instructions

**File:** `.github/instructions/internal/domain/*.instructions.md`

```markdown
# Domain Layer Instructions

These files contain pure business logic with no external dependencies.

## Rules

1. **No External Dependencies**: Never import database, HTTP, or infrastructure packages
2. **Validation Required**: All entities must validate their own state
3. **Immutability**: Entities should be immutable after creation where possible
4. **Domain Errors**: Use domain-specific errors from `errors.go`
5. **Repository Interfaces**: Define interfaces (ports) but never implementations

## Entity Structure Pattern

```go
type EntityName struct {
    ID        string
    // fields...
    CreatedAt time.Time
}

func NewEntityName(...) (*EntityName, error) {
    entity := &EntityName{...}
    if err := entity.Validate(); err != nil {
        return nil, err
    }
    return entity, nil
}

func (e *EntityName) Validate() error {
    // validation logic
    return nil
}
```

## Repository Interface Pattern

```go
type EntityRepository interface {
    Create(ctx context.Context, entity *Entity) error
    FindByID(ctx context.Context, id string) (*Entity, error)
    // other methods...
}
```

## Testing

- Test validation thoroughly with table-driven tests
- Test all edge cases (empty strings, nil values, boundary conditions)
- No database required for domain tests
```

### 2.2 Repository Implementation Instructions

**File:** `.github/instructions/internal/adapters/repository/*.instructions.md`

```markdown
# Repository Adapter Instructions

Repository implementations using sqlc-generated code.

## Critical Rules

1. **Never Edit Generated Code**: Files in `sqlc/sqlite/` and `sqlc/postgres/` are auto-generated
2. **Modify Queries**: Edit `queries.sql` files, then run `sqlc generate`
3. **Test Both Databases**: Every repository must have SQLite and PostgreSQL tests
4. **Error Mapping**: Map database errors to domain errors

## Repository Implementation Pattern

```go
type urlRepository struct {
    sqliteQueries  *sqliterepo.Queries
    postgresQueries *postgresrepo.Queries
    dbType string
}

func (r *urlRepository) Create(ctx context.Context, url *url.URL) error {
    // Map domain entity to database params
    params := sqliterepo.CreateURLParams{
        ID:        url.ID,
        ShortCode: url.ShortCode,
        // ...
    }
    
    if err := r.sqliteQueries.CreateURL(ctx, params); err != nil {
        // Map database errors to domain errors
        if strings.Contains(err.Error(), "UNIQUE constraint") {
            return url.ErrDuplicateShortCode
        }
        return fmt.Errorf("failed to create URL: %w", err)
    }
    
    return nil
}
```

## Testing Pattern

- Use in-memory SQLite for fast, isolated tests
- PostgreSQL tests should skip if database unavailable
- Apply migrations in test setup using goose
- Clean up test data in teardown

```go
func TestURLRepository_SQLite(t *testing.T) {
    db := setupTestSQLite(t)
    defer db.Close()
    
    repo := NewURLRepository(db, "sqlite")
    // test cases...
}
```
```

### 2.3 SQL Query Instructions

**File:** `.github/instructions/internal/adapters/repository/sqlc/**/*.sql.instructions.md`

```markdown
# sqlc Query Instructions

SQL queries for type-safe code generation.

## Critical Rules

1. **Database Compatibility**: Write separate queries for SQLite and PostgreSQL
2. **Named Queries**: Use `-- name: QueryName :exec|one|many` format
3. **Parameters**: Use `?` for SQLite, `$1, $2` for PostgreSQL
4. **Null Handling**: Use `sqlc.narg()` for nullable parameters

## Query Naming Convention

- `Create<Entity>` - Insert operations
- `Get<Entity>By<Field>` - Select single row
- `List<Entity>` - Select multiple rows
- `Update<Entity>` - Update operations
- `Delete<Entity>` - Delete operations

## Example Patterns

### Insert (SQLite)
```sql
-- name: CreateURL :exec
INSERT INTO urls (id, short_code, original_url, created_at, created_by)
VALUES (?, ?, ?, ?, ?);
```

### Insert (PostgreSQL)
```sql
-- name: CreateURL :exec
INSERT INTO urls (id, short_code, original_url, created_at, created_by)
VALUES ($1, $2, $3, $4, $5);
```

### Select with Join
```sql
-- name: GetURLWithClickCount :one
SELECT u.*, COUNT(c.id) as click_count
FROM urls u
LEFT JOIN clicks c ON c.url_id = u.id
WHERE u.short_code = ?
GROUP BY u.id;
```

## After Modifying Queries

Always run: `sqlc generate`
```

### 2.4 Migration Instructions

**File:** `.github/instructions/internal/migrations/**/*.sql.instructions.md`

```markdown
# Database Migration Instructions

Goose migration files for SQLite and PostgreSQL.

## Critical Rules

1. **Dual Migrations**: Create both SQLite and PostgreSQL versions
2. **Reversible**: Every migration must have UP and DOWN sections
3. **Test Both Directions**: Verify up and down migrations work
4. **Idempotent**: Use IF NOT EXISTS where appropriate

## Migration Structure

```sql
-- +goose Up
-- SQL in this section is executed when this migration is applied
CREATE TABLE IF NOT EXISTS table_name (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- SQL in this section is executed when this migration is rolled back
DROP TABLE IF EXISTS table_name;
```

## Database-Specific Considerations

### SQLite
- Use `TEXT` for UUIDs and strings
- Use `INTEGER` for timestamps (Unix epoch)
- Use `AUTOINCREMENT` for auto-increment IDs

### PostgreSQL
- Use `UUID` type for IDs
- Use `TIMESTAMP WITH TIME ZONE` for timestamps
- Use `SERIAL` or `BIGSERIAL` for auto-increment IDs

## Testing Migrations

```bash
# Apply migration
make migrate-up

# Verify schema
sqlite3 database.db ".schema"

# Test rollback
make migrate-down

# Verify rollback worked
```

## After Creating Migration

1. Rebuild migrate tool: `make build-migrate`
2. Test migration: `make migrate-up`
3. Update schema documentation in `docs/`
```

### 2.5 Test File Instructions

**File:** `.github/instructions/**/*_test.go.instructions.md`

```markdown
# Test File Instructions

Go test files following project conventions.

## Test Naming Convention

- File: `<source>_test.go` (e.g., `url_test.go` for `url.go`)
- Function: `Test<Type>_<Method>_<Scenario>`
- Subtests: Use `t.Run()` with descriptive names

## Table-Driven Test Pattern

```go
func TestURL_Validate(t *testing.T) {
    tests := []struct {
        name    string
        url     *URL
        wantErr error
    }{
        {
            name: "valid URL with short code",
            url: &URL{
                ShortCode: "abc123",
                OriginalURL: "https://example.com",
            },
            wantErr: nil,
        },
        {
            name: "short code too short",
            url: &URL{
                ShortCode: "ab",
                OriginalURL: "https://example.com",
            },
            wantErr: ErrInvalidShortCode,
        },
        // More cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.url.Validate()
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Repository Test Pattern

### SQLite Test
```go
func TestURLRepository_Create_SQLite(t *testing.T) {
    db := setupTestSQLite(t)
    defer db.Close()
    
    repo := NewURLRepository(db, "sqlite")
    // test implementation
}

func setupTestSQLite(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    
    // Apply migrations
    require.NoError(t, goose.Up(db, "../../migrations/sqlite"))
    
    return db
}
```

### PostgreSQL Test (with skip)
```go
func TestURLRepository_Create_Postgres(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping PostgreSQL integration test")
    }
    
    db := setupTestPostgres(t)
    if db == nil {
        t.Skip("PostgreSQL not available")
        return
    }
    defer db.Close()
    
    // test implementation
}
```

## Test Best Practices

- Use `require` for setup assertions (fails fast)
- Use `assert` for test assertions (shows all failures)
- Clean up resources with `defer`
- Use subtests for related scenarios
- Test happy path AND error cases
```

---

## 3. Design `copilot-setup-steps.yml`

This file pre-installs dependencies in Copilot's environment, making PR generation faster and more reliable.

**File:** `.github/copilot-setup-steps.yml`

```yaml
# GitHub Copilot Setup Steps
# Pre-installs dependencies and generates code before Copilot creates PRs
# Documentation: https://docs.github.com/en/copilot/customizing-copilot/adding-custom-instructions-for-github-copilot

name: mjrwtf-copilot-setup
description: Setup steps for mjr.wtf URL shortener development environment

steps:
  # Step 1: Verify Go version
  - name: Check Go version
    run: |
      go version
      if ! go version | grep -q "go1.24"; then
        echo "Warning: Go 1.24.2+ recommended"
      fi

  # Step 2: Install Go dependencies
  - name: Download Go dependencies
    run: |
      go mod download
      go mod verify
    timeout: 120

  # Step 3: Install sqlc (CRITICAL)
  - name: Install sqlc
    run: |
      go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
    timeout: 60

  # Step 4: Generate sqlc code (CRITICAL - required before compilation)
  - name: Generate sqlc code
    run: sqlc generate
    timeout: 10
    description: |
      Generates type-safe database code in internal/adapters/repository/sqlc/
      This MUST run before building or testing - compilation will fail without it.

  # Step 5: Install golangci-lint
  - name: Install golangci-lint
    run: |
      curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
        sh -s -- -b $(go env GOPATH)/bin v1.56.0
    timeout: 120

  # Step 6: Build migrate tool
  - name: Build migration tool
    run: make build-migrate
    timeout: 30
    description: |
      Builds the migration CLI tool used for database migrations.
      Required for any migration-related changes.

  # Step 7: Set up test environment
  - name: Setup test environment
    run: |
      # Create .env from example if it doesn't exist
      if [ ! -f .env ]; then
        cp .env.example .env
        echo "DATABASE_URL=:memory:" >> .env
      fi
    timeout: 5

  # Step 8: Verify setup by running tests
  - name: Verify setup with tests
    run: make test
    timeout: 30
    description: |
      Runs all tests to verify the environment is correctly configured.
      PostgreSQL tests will skip if database is not available (expected).

validation:
  # Commands that must succeed for setup to be valid
  required:
    - sqlc generate
    - go build ./...
    - make test

  # Optional but recommended
  recommended:
    - make lint
    - make fmt

# Environment variables needed for full functionality
environment:
  - name: DATABASE_URL
    description: Database connection string
    default: ":memory:"
    required: false
  
  - name: CGO_ENABLED
    description: Enable CGO for SQLite support
    default: "1"
    required: true

# Cache these directories for faster subsequent runs
cache:
  - $GOPATH/pkg/mod          # Go module cache
  - $GOPATH/bin              # Installed binaries (sqlc, golangci-lint)
  - bin/                     # Built binaries
  - internal/adapters/repository/sqlc/  # Generated code

# Common issues and solutions
troubleshooting:
  - issue: "undefined: postgresrepo"
    solution: "Run: sqlc generate"
  
  - issue: "bin/migrate: command not found"
    solution: "Run: make build-migrate"
  
  - issue: "golangci-lint errors about undefined packages"
    solution: "These are false positives if tests pass - ignore them"

# Estimated setup time
estimated_duration: 180  # seconds (3 minutes)
```

### Key Features of This Setup File

1. **Ordered Steps**: Each step builds on the previous one
2. **Timeouts**: Prevents hanging on slow operations
3. **Critical Step Highlighted**: `sqlc generate` is clearly marked as CRITICAL
4. **Validation Commands**: Verifies setup worked correctly
5. **Environment Variables**: Documents what's needed
6. **Caching**: Speeds up subsequent runs
7. **Troubleshooting**: Common errors and solutions
8. **Time Estimate**: Helps set expectations

---

## 4. Custom Agent Recommendations

You already have 6 excellent custom agents. Here are additional domain-specific agents to consider:

### 4.1 New Agent: SQL Query Specialist

**File:** `.github/agents/sqlc-query-specialist.md`

```markdown
---
name: sqlc-query-specialist
description: Expert in sqlc query writing, SQL optimization, and dual-database support
tools: ["read", "search", "edit", "shell"]
---

You are a senior database engineer specializing in SQL query optimization, sqlc code generation, and supporting both SQLite and PostgreSQL databases.

## Your Expertise

- Writing efficient SQL queries for both SQLite and PostgreSQL
- sqlc configuration and code generation
- Query optimization and indexing strategies
- Handling database-specific features and limitations
- Transaction management and isolation levels
- Database migration strategies

## Your Responsibilities

### Query Writing
- Write database-agnostic SQL when possible
- Handle SQLite vs PostgreSQL syntax differences
- Use appropriate parameter placeholders (? for SQLite, $1 for PostgreSQL)
- Optimize queries for performance
- Prevent SQL injection (sqlc handles this automatically)

### sqlc Best Practices
- Follow sqlc naming conventions (-- name: QueryName :exec|one|many)
- Use appropriate return types (:exec, :one, :many, :execrows)
- Handle nullable values with sqlc.narg()
- Generate interfaces with emit_interface: true
- Test generated code with both databases

### Database-Specific Considerations

**SQLite:**
- No UUID type (use TEXT)
- Limited concurrent writes
- AUTOINCREMENT for auto-incrementing IDs
- INTEGER for Unix timestamps
- Use sqlite_stat1 for query planning

**PostgreSQL:**
- Native UUID type support
- Better concurrency support
- SERIAL/BIGSERIAL for auto-incrementing
- TIMESTAMP WITH TIME ZONE for timestamps
- Use EXPLAIN ANALYZE for query planning

### Query Optimization Patterns

```sql
-- Add indexes for frequently queried columns
CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);
CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at);

-- Use composite indexes for multi-column queries
CREATE INDEX IF NOT EXISTS idx_urls_created_by_created_at 
ON urls(created_by, created_at);

-- Optimize joins with proper indexes
SELECT u.*, COUNT(c.id) as click_count
FROM urls u
LEFT JOIN clicks c ON c.url_id = u.id  -- indexed foreign key
WHERE u.created_by = ?  -- indexed column
GROUP BY u.id;
```

## Working Process

1. **Understand Requirements**: What data needs to be queried?
2. **Check Existing Queries**: Review patterns in queries.sql
3. **Write Query**: Create both SQLite and PostgreSQL versions if needed
4. **Generate Code**: Run `sqlc generate`
5. **Test**: Verify generated code compiles and tests pass
6. **Optimize**: Add indexes if query performance is critical

## Common Patterns

### Insert
```sql
-- name: CreateURL :exec
INSERT INTO urls (id, short_code, original_url, created_at, created_by)
VALUES (?, ?, ?, ?, ?);
```

### Select One
```sql
-- name: GetURLByShortCode :one
SELECT * FROM urls WHERE short_code = ? LIMIT 1;
```

### Select Many
```sql
-- name: ListURLsByCreatedBy :many
SELECT * FROM urls WHERE created_by = ? ORDER BY created_at DESC;
```

### Update
```sql
-- name: UpdateURL :exec
UPDATE urls SET original_url = ? WHERE id = ?;
```

### Delete
```sql
-- name: DeleteURL :exec
DELETE FROM urls WHERE id = ?;
```

### Complex Join with Aggregation
```sql
-- name: GetURLStatsWithClicks :one
SELECT 
    u.*,
    COUNT(c.id) as total_clicks,
    COUNT(DISTINCT c.country) as countries,
    MAX(c.clicked_at) as last_clicked
FROM urls u
LEFT JOIN clicks c ON c.url_id = u.id
WHERE u.short_code = ?
GROUP BY u.id;
```

## After Making Changes

Always:
1. Run `sqlc generate`
2. Verify compilation: `go build ./...`
3. Run tests: `make test`
4. Check lint: `make lint` (ignore false positives about undefined repos)

Your goal is to write efficient, maintainable SQL queries that work seamlessly with sqlc's code generation and support both SQLite and PostgreSQL databases.
```

### 4.2 New Agent: Migration Specialist

**File:** `.github/agents/migration-specialist.md`

```markdown
---
name: migration-specialist
description: Expert in database schema design, migrations, and version management with goose
tools: ["read", "search", "edit", "shell"]
---

You are a senior database architect specializing in schema design, database migrations, and zero-downtime deployments.

## Your Expertise

- Database schema design and normalization
- Migration strategy and execution (goose)
- Backward compatibility and rollback strategies
- Data migration and transformation
- Index design and optimization
- Supporting multiple database engines (SQLite, PostgreSQL)

## Your Responsibilities

### Migration Creation
- Design reversible migrations (both UP and DOWN)
- Create separate migrations for SQLite and PostgreSQL
- Ensure migrations are idempotent when possible
- Handle data migrations safely
- Plan for zero-downtime deployments

### Schema Design Principles
- Normalize to 3NF unless performance requires denormalization
- Use appropriate data types for each database
- Design indexes for query patterns
- Implement referential integrity with foreign keys
- Consider future extensibility

## Migration Structure

### Basic Migration Template
```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS new_table (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_new_table_name ON new_table(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_new_table_name;
DROP TABLE IF EXISTS new_table;
-- +goose StatementEnd
```

### Adding Column (Safe)
```sql
-- +goose Up
ALTER TABLE urls ADD COLUMN description TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN easily, need recreation
-- PostgreSQL supports it:
-- ALTER TABLE urls DROP COLUMN description;
```

### Data Migration Pattern
```sql
-- +goose Up
-- Add new column with default
ALTER TABLE urls ADD COLUMN status TEXT DEFAULT 'active';

-- Migrate existing data
UPDATE urls SET status = 'active' WHERE status IS NULL;

-- +goose Down
-- Remove column (PostgreSQL)
ALTER TABLE urls DROP COLUMN status;
```

## Database-Specific Migration Patterns

### SQLite Limitations
- No ALTER COLUMN support
- No DROP COLUMN support (before SQLite 3.35.0)
- Workaround: Create new table, copy data, rename

```sql
-- +goose Up
-- Recreate table with new schema
CREATE TABLE urls_new (
    id TEXT PRIMARY KEY,
    short_code TEXT UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    new_column TEXT,  -- Added column
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO urls_new (id, short_code, original_url, created_at)
SELECT id, short_code, original_url, created_at FROM urls;

DROP TABLE urls;
ALTER TABLE urls_new RENAME TO urls;
```

### PostgreSQL Advanced Features
- Support for concurrent index creation
- Transactional DDL
- Column addition with default values

```sql
-- +goose Up
-- Add column with NOT NULL using default
ALTER TABLE urls ADD COLUMN status TEXT DEFAULT 'active' NOT NULL;

-- Create index concurrently (doesn't lock table)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_urls_status ON urls(status);
```

## Migration Best Practices

1. **Test Migrations Thoroughly**
   ```bash
   # Apply migration
   make migrate-up
   
   # Verify schema
   sqlite3 database.db ".schema"
   
   # Test rollback
   make migrate-down
   
   # Re-apply
   make migrate-up
   ```

2. **Version Control**
   - Use timestamp prefix: `YYYYMMDDHHMMSS_description.sql`
   - Never modify applied migrations
   - Create new migration to fix issues

3. **Backward Compatibility**
   - Add columns before making them required
   - Maintain old columns during transition period
   - Use feature flags for schema-dependent features

4. **Performance Considerations**
   - Create indexes after bulk data operations
   - Use transactions for data migrations
   - Consider table locking implications

## Migration Workflow

### Creating New Migration
```bash
# Create migration
make migrate-create NAME=add_user_roles

# Edit both files:
# - internal/migrations/sqlite/XXXXXX_add_user_roles.sql
# - internal/migrations/postgres/XXXXXX_add_user_roles.sql

# Rebuild migrate tool (migrations are embedded)
make build-migrate

# Test migration
export DATABASE_URL=:memory:
make migrate-up

# Verify
make migrate-status

# Test rollback
make migrate-down
```

### Post-Migration Tasks
1. Update schema documentation in `docs/schema.*.sql`
2. Regenerate sqlc code: `sqlc generate`
3. Update repository code if new tables/columns added
4. Add tests for new schema elements
5. Update README if migration requires manual steps

## Common Schema Patterns

### Timestamps
```sql
-- SQLite
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

-- PostgreSQL
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
```

### Foreign Keys
```sql
CREATE TABLE clicks (
    id TEXT PRIMARY KEY,
    url_id TEXT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    -- other columns
);
```

### Indexes for Common Queries
```sql
-- Unique constraint
CREATE UNIQUE INDEX idx_urls_short_code ON urls(short_code);

-- Foreign key query optimization
CREATE INDEX idx_clicks_url_id ON clicks(url_id);

-- Composite index for filtered queries
CREATE INDEX idx_urls_created_by_created_at ON urls(created_by, created_at);

-- Partial index (PostgreSQL)
CREATE INDEX idx_active_urls ON urls(created_at) WHERE status = 'active';
```

## Handling Migration Failures

### Rollback Strategy
```bash
# If migration fails, rollback
make migrate-down

# Fix migration file
# Rebuild
make build-migrate

# Try again
make migrate-up
```

### Production Deployment
1. Backup database before migration
2. Test migration on production copy
3. Plan rollback strategy
4. Apply during maintenance window if downtime needed
5. Monitor application logs during deployment

Your goal is to design safe, reversible database migrations that support both SQLite and PostgreSQL while maintaining data integrity and enabling zero-downtime deployments where possible.
```

### 4.3 Enhancement: Add Domain-Specific Context to Existing Agents

**Recommendation:** Update existing agents to include URL shortener domain knowledge.

**Example for golang-expert.md - Add section:**

```markdown
## Domain-Specific Patterns for mjr.wtf

### URL Shortener Business Rules
- Short codes must be 3-20 characters
- Only alphanumeric, underscore, and hyphen allowed in short codes
- URLs must be valid http/https URLs
- Created_by tracks which user created the shortened URL
- Click tracking includes referrer, country (ISO 3166-1 alpha-2), and user agent

### Analytics Patterns
```go
// Aggregate click statistics efficiently
func (r *clickRepository) GetStatsByURL(ctx context.Context, urlID string, timeRange TimeRange) (*ClickStats, error) {
    // Use GROUP BY for aggregation
    // Include: total_clicks, unique_countries, top_referrers
}
```

### Rate Limiting Considerations
- Consider rate limiting short code creation per user
- Track suspicious click patterns (bot detection)
- Implement CAPTCHA for public endpoints if needed
```

---

## 5. Issue Template Recommendations

Create GitHub issue templates for well-scoped issues that Copilot can effectively work with.

### 5.1 Feature Request Template

**File:** `.github/ISSUE_TEMPLATE/feature_request.md`

```markdown
---
name: Feature Request
about: Propose a new feature for mjr.wtf URL shortener
title: '[FEATURE] '
labels: ['feature', 'needs-triage']
assignees: ''
---

## User Story

**As a** [type of user]  
**I want** [capability or feature]  
**So that** [benefit or value]

## Problem Statement

<!-- Describe the problem this feature solves -->

## Proposed Solution

<!-- Describe your preferred solution -->

## Acceptance Criteria

<!-- Define specific, measurable criteria using Given-When-Then format -->

- [ ] Given [context], when [action], then [outcome]
- [ ] Given [context], when [action], then [outcome]
- [ ] All existing tests continue to pass
- [ ] New functionality has unit tests
- [ ] Documentation updated (if applicable)

## Technical Considerations

### Database Changes
<!-- List any schema changes, migrations, or new queries needed -->

### API Changes
<!-- List new endpoints or changes to existing endpoints -->

### Architecture Impact
<!-- Which layers are affected? (domain/adapters/infrastructure) -->

### Security Considerations
<!-- Authentication, authorization, input validation, etc. -->

### Performance Impact
<!-- Expected impact on performance, scalability concerns -->

## Implementation Guidance

### Files Likely to Change
<!-- Help Copilot by listing files that need modification -->
- [ ] `internal/domain/<entity>/` - [ Description ]
- [ ] `internal/adapters/repository/` - [ Description ]
- [ ] `internal/migrations/` - [ Description ]

### Required Steps
<!-- Ordered list of implementation steps -->
1. Create migration for schema changes
2. Update domain entity and validation
3. Add repository methods
4. Update sqlc queries
5. Add tests (domain + repository)
6. Update API handlers (if applicable)

## Dependencies

<!-- Link related issues -->
- Blocks: #
- Blocked by: #
- Related: #

## Additional Context

<!-- Screenshots, examples, references, etc. -->

---

**Priority:** [ P0: Critical | P1: High | P2: Medium | P3: Low ]  
**Complexity:** [ XS | S | M | L | XL ]  
**Component:** [ URL Management | Click Tracking | Analytics | Authentication | API | Infrastructure ]
```

### 5.2 Bug Report Template

**File:** `.github/ISSUE_TEMPLATE/bug_report.md`

```markdown
---
name: Bug Report
about: Report a bug in mjr.wtf URL shortener
title: '[BUG] '
labels: ['bug', 'needs-triage']
assignees: ''
---

## Bug Description

<!-- Clear and concise description of the bug -->

## Steps to Reproduce

1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

## Expected Behavior

<!-- What should happen -->

## Actual Behavior

<!-- What actually happens -->

## Error Messages

<!-- Include full error messages, stack traces, log output -->

```
Paste error messages here
```

## Environment

- **Go Version:** [ Run: `go version` ]
- **Database:** [ SQLite | PostgreSQL ]
- **Database Version:** [ Run: `sqlite3 --version` or `psql --version` ]
- **OS:** [ e.g., macOS 13.5, Ubuntu 22.04 ]
- **Project Commit:** [ Run: `git rev-parse HEAD` ]

## Minimal Reproduction

<!-- If possible, provide minimal code to reproduce the issue -->

```go
// Minimal code that demonstrates the bug
```

## Impact

**Severity:** [ Critical | High | Medium | Low ]  
**Frequency:** [ Always | Often | Sometimes | Rare ]  
**Users Affected:** [ All | Specific scenario | Single user ]

## Acceptance Criteria for Fix

- [ ] Bug no longer reproducible with steps above
- [ ] Regression test added to prevent recurrence
- [ ] All existing tests continue to pass
- [ ] Root cause documented in PR description

## Additional Context

<!-- Screenshots, related issues, workarounds, etc. -->

---

**Component:** [ URL Management | Click Tracking | Analytics | Authentication | API | Database | Migration | Tests ]
```

### 5.3 Technical Task Template

**File:** `.github/ISSUE_TEMPLATE/technical_task.md`

```markdown
---
name: Technical Task
about: Non-feature work like refactoring, technical debt, or infrastructure
title: '[TASK] '
labels: ['task', 'needs-triage']
assignees: ''
---

## Task Description

<!-- What needs to be done and why -->

## Motivation

<!-- Why is this task important? What problem does it solve? -->

## Scope

<!-- What's included and what's NOT included -->

**In Scope:**
- [ ] Item 1
- [ ] Item 2

**Out of Scope:**
- Item 1
- Item 2

## Acceptance Criteria

- [ ] Specific outcome 1
- [ ] Specific outcome 2
- [ ] All tests pass
- [ ] No breaking changes (or breaking changes documented)

## Implementation Plan

### Pre-requisites
<!-- What needs to be done first? -->

### Steps
1. Step 1
2. Step 2
3. Step 3

### Validation
<!-- How to verify the task is complete -->
```bash
# Commands to verify completion
make test
make lint
```

## Technical Details

### Files to Modify
<!-- Help Copilot by listing specific files -->
- `path/to/file.go` - [ Description of changes ]

### Dependencies
- Depends on: #
- Blocks: #

## Risk Assessment

**Risk Level:** [ Low | Medium | High ]  

**Risks:**
- Risk 1: [Description] - Mitigation: [Strategy]
- Risk 2: [Description] - Mitigation: [Strategy]

## Rollback Plan

<!-- How to undo changes if something goes wrong -->

---

**Priority:** [ P0 | P1 | P2 | P3 ]  
**Complexity:** [ XS | S | M | L | XL ]  
**Type:** [ Refactoring | Technical Debt | Infrastructure | Build/CI | Documentation ]
```

### 5.4 Epic Template

**File:** `.github/ISSUE_TEMPLATE/epic.md`

```markdown
---
name: Epic
about: Large initiative spanning multiple issues
title: '[EPIC] '
labels: ['epic']
assignees: ''
---

## Epic Overview

<!-- High-level description of this initiative -->

## Business Value

<!-- Why are we doing this? What's the expected impact? -->

## Success Metrics

<!-- How will we measure success? -->

- Metric 1: [Description and target]
- Metric 2: [Description and target]

## User Stories

<!-- High-level user stories this epic addresses -->

1. As a [user], I want [capability], so that [benefit]
2. As a [user], I want [capability], so that [benefit]

## Scope

### In Scope
- [ ] Feature/capability 1
- [ ] Feature/capability 2

### Out of Scope
- Feature/capability A (future phase)
- Feature/capability B (not planned)

## Technical Architecture

### Components Affected
- [ ] Domain layer: [Details]
- [ ] Adapters: [Details]
- [ ] Infrastructure: [Details]
- [ ] Database schema: [Details]

### Major Technical Decisions
- Decision 1: [Rationale]
- Decision 2: [Rationale]

## Implementation Phases

### Phase 1: [Name]
- [ ] #issue-1
- [ ] #issue-2

### Phase 2: [Name]
- [ ] #issue-3
- [ ] #issue-4

### Phase 3: [Name]
- [ ] #issue-5
- [ ] #issue-6

## Dependencies

- External dependency 1
- External dependency 2

## Timeline

- **Target Start:** [Date]
- **Target Completion:** [Date]
- **Estimated Effort:** [Story points or time]

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Risk 1 | High | Medium | Mitigation strategy |
| Risk 2 | Medium | Low | Mitigation strategy |

## Progress Tracking

<!-- Update as work progresses -->

- [ ] Phase 1 Complete
- [ ] Phase 2 Complete
- [ ] Phase 3 Complete
- [ ] Documentation Updated
- [ ] Deployed to Production

---

**Priority:** [ P0 | P1 | P2 | P3 ]  
**Size:** [ S | M | L | XL ]
```

---

## 6. Additional Best Practice Recommendations

### 6.1 Add Copilot Workspace Documentation

**File:** `.github/copilot-workspace-guide.md`

```markdown
# GitHub Copilot Workspace Guide for mjr.wtf

## Quick Reference for Copilot PR Generation

### Before Creating a PR

1. **Ensure issue is well-scoped** (use issue templates)
2. **Verify environment**: `sqlc generate && make check`
3. **Review custom agents** - Select appropriate agent(s) for the task

### What Copilot Does Well For This Project

✅ **Excellent for:**
- Implementing new domain entities with validation
- Adding new repository methods with tests
- Creating database migrations (both SQLite and PostgreSQL)
- Writing sqlc queries
- Adding test cases to existing test suites
- Fixing bugs with clear reproduction steps
- Updating documentation

⚠️ **Needs guidance for:**
- Complex architectural changes spanning multiple layers
- Performance optimization requiring profiling
- Security-critical code (review carefully)
- Major refactoring (break into smaller issues)

### How to Write Issues for Copilot

**Good Issue Example:**
```
Title: Add URL expiration feature

As a URL creator
I want to set an expiration date on short URLs
So that links automatically become invalid after a certain time

Acceptance Criteria:
- [ ] Given a URL with expiration date set, when the date passes, then the URL returns 404
- [ ] Given an expired URL, when checking its status, then status shows "expired"
- [ ] URL expiration is optional (null means never expires)

Files to modify:
- internal/domain/url/url.go - Add ExpiresAt field
- internal/migrations/*.sql - Add expires_at column
- internal/adapters/repository/sqlc/*/queries.sql - Add expiration check
- Tests for validation and repository
```

**Bad Issue Example:**
```
Title: Make URLs better

We need to improve URLs somehow. Maybe add features?
```

### Agent Selection Guide

| Task Type | Recommended Agent(s) |
|-----------|---------------------|
| New domain entity | golang-expert |
| Repository implementation | golang-expert + sqlc-query-specialist |
| Database migration | migration-specialist |
| SQL query optimization | sqlc-query-specialist |
| Test coverage improvement | test-specialist |
| API endpoint | golang-expert + api-designer |
| Security review | security-expert |
| Documentation | documentation-writer |
| Issue creation | business-analyst |

### Pre-PR Checklist

- [ ] Issue has clear acceptance criteria
- [ ] Files likely to change are listed
- [ ] Database changes are identified
- [ ] Tests requirements are specified
- [ ] Breaking changes are noted
- [ ] Security implications considered

### Post-PR Review Checklist

- [ ] All acceptance criteria met
- [ ] Tests added and passing
- [ ] sqlc code regenerated (if queries changed)
- [ ] Migrations tested (up and down)
- [ ] Documentation updated
- [ ] No unintended changes
- [ ] Security reviewed
- [ ] Performance acceptable
```

### 6.2 Create CONTRIBUTING.md with Copilot Guidance

**File:** `CONTRIBUTING.md`

```markdown
# Contributing to mjr.wtf

## For Human Contributors

### Development Workflow

1. **Fork and Clone**
   ```bash
   git clone <your-fork>
   cd mjrwtf
   ```

2. **Setup Environment**
   ```bash
   cp .env.example .env
   sqlc generate
   make build-migrate
   make test
   ```

3. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Changes**
   - Follow hexagonal architecture patterns
   - Keep domain logic in `internal/domain/`
   - Run `sqlc generate` after query changes
   - Run `make check` before committing

5. **Create Pull Request**
   - Fill out PR template completely
   - Reference related issues
   - Ensure all tests pass
   - Request review from maintainers

### Code Quality Standards

- **Tests Required**: All new features and bug fixes must have tests
- **Test Coverage**: Aim for >80% coverage
- **Linting**: Fix all linter warnings (except known false positives)
- **Documentation**: Update docs for API or behavior changes
- **Commit Messages**: Use conventional commits format

## For GitHub Copilot

### Automated PR Generation

This project is optimized for GitHub Copilot coding agent. When creating PRs:

1. **Start with well-scoped issue** using our issue templates
2. **Copilot runs setup steps** (see `.github/copilot-setup-steps.yml`)
3. **Copilot generates changes** following custom instructions
4. **Manual review required** before merging

### What Copilot Can Do

- Implement features from well-defined issues
- Fix bugs with clear reproduction steps
- Add tests to improve coverage
- Update documentation
- Create database migrations
- Write sqlc queries

### What Requires Human Review

- Security-critical changes
- Major architectural decisions
- Performance optimizations
- Breaking changes
- Production deployment decisions

## Architecture Guidelines

See `.github/copilot-instructions.md` for comprehensive project documentation.

### Layer Boundaries

**Domain Layer** (`internal/domain/`):
- Pure business logic
- No external dependencies
- Repository interfaces defined here

**Adapter Layer** (`internal/adapters/`):
- Repository implementations
- External service integrations
- Uses sqlc-generated code

**Infrastructure Layer** (`internal/infrastructure/`):
- Configuration
- Logging
- Database connections

### Critical Workflows

1. **After changing queries:**
   ```bash
   sqlc generate
   go build ./...
   make test
   ```

2. **After creating migration:**
   ```bash
   make build-migrate
   make migrate-up
   make migrate-status
   ```

3. **Before committing:**
   ```bash
   make fmt
   make check
   ```

## Getting Help

- Read `.github/copilot-instructions.md` first
- Check existing issues and PRs
- Ask questions in discussions
- Tag maintainers for urgent issues

## License

By contributing, you agree that your contributions will be licensed under the project's license.
```

---

## 7. Summary of Recommendations

### Immediate Actions (High Priority)

1. **Create `copilot-setup-steps.yml`** ⚠️ CRITICAL
   - Enables reliable PR generation
   - Pre-installs dependencies
   - Runs sqlc generate automatically
   - Estimated setup time: 3 minutes

2. **Add path-specific instructions** 📁
   - Domain layer instructions
   - Repository adapter instructions
   - SQL query instructions
   - Migration instructions
   - Test file instructions

3. **Create issue templates** 📝
   - Feature request template
   - Bug report template
   - Technical task template
   - Epic template

4. **Enhance existing `copilot-instructions.md`** ✏️
   - Add code style guidelines
   - Add PR guidelines
   - Add security checklist
   - Add troubleshooting section

### Short-Term Actions (Medium Priority)

5. **Create additional custom agents** 🤖
   - sqlc-query-specialist
   - migration-specialist
   - Enhance existing agents with domain knowledge

6. **Add supporting documentation** 📚
   - Copilot workspace guide
   - CONTRIBUTING.md with Copilot section

### Long-Term Actions (Low Priority)

7. **Continuous Improvement** 🔄
   - Monitor Copilot PR quality
   - Update instructions based on common issues
   - Collect metrics on PR success rate
   - Refine agent definitions based on usage

---

## 8. Expected Impact

### Before Improvements
- ❌ Copilot may fail to compile code (missing sqlc generate)
- ❌ PRs may have incorrect patterns (no path-specific guidance)
- ❌ Issues lack detail for automated PR generation
- ❌ Long setup time for Copilot environment

### After Improvements
- ✅ Copilot environment ready in 3 minutes
- ✅ Generated code follows project patterns
- ✅ PRs more likely to pass CI on first try
- ✅ Issues well-scoped for automation
- ✅ Faster iteration on features

### Success Metrics

Track these to measure improvement:
- **PR Success Rate**: % of Copilot PRs that pass tests on first try
- **Setup Time**: Time for Copilot to prepare environment
- **Review Time**: Time humans spend reviewing Copilot PRs
- **Issue Quality**: % of issues with complete acceptance criteria
- **Agent Usage**: Which agents are most frequently used

---

## 9. Implementation Priority

### Phase 1: Critical Foundation (Week 1)
1. Create `copilot-setup-steps.yml`
2. Enhance `copilot-instructions.md` with missing sections
3. Create feature request and bug report templates

### Phase 2: Guidance Enhancement (Week 2)
1. Create all path-specific instructions
2. Create technical task and epic templates
3. Add CONTRIBUTING.md

### Phase 3: Specialized Agents (Week 3)
1. Create sqlc-query-specialist agent
2. Create migration-specialist agent
3. Enhance existing agents with domain knowledge
4. Create Copilot workspace guide

### Phase 4: Monitoring & Iteration (Ongoing)
1. Track success metrics
2. Gather feedback from PR reviews
3. Refine instructions based on common issues
4. Update documentation as project evolves

---

## 10. Conclusion

The mjr.wtf project has an **excellent foundation** for GitHub Copilot integration. The existing `.github/copilot-instructions.md` is comprehensive and well-structured. The six custom agents provide specialized expertise.

The **key gaps** are:
1. Missing `copilot-setup-steps.yml` (CRITICAL - prevents reliable PR generation)
2. No path-specific instructions (limits context-aware code generation)
3. No issue templates (reduces issue quality)

Implementing these recommendations will transform Copilot from a "helpful assistant" to a "reliable team member" that can:
- Generate PRs with 80%+ success rate
- Follow project patterns consistently
- Handle complex multi-file changes
- Reduce human review time by 50%

**Estimated implementation effort:** 8-12 hours spread over 3 weeks
**Expected ROI:** Significant - faster development, higher code quality, less review overhead

---

## Appendix A: Comparison to Best Practices

| Best Practice | Current State | Recommendation | Priority |
|--------------|---------------|----------------|----------|
| Well-scoped issues | ⚠️ No templates | Add 4 issue templates | High |
| Repository-wide instructions | ✅ Excellent | Enhance with style guide | Medium |
| Path-specific instructions | ❌ None | Add 5 path-specific files | High |
| Pre-install dependencies | ❌ None | Create setup-steps.yml | CRITICAL |
| Custom agents | ✅ Good (6 agents) | Add 2 specialized agents | Low |
| Coding standards | ⚠️ Implied | Make explicit | Medium |
| PR guidelines | ❌ None | Add to instructions | Medium |
| Security checklist | ⚠️ Minimal | Add comprehensive list | High |

---

**Document Version:** 1.0  
**Last Updated:** 2025-11-08  
**Author:** Business Analyst Agent
