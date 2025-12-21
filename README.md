# mjr.wtf - url shortener

A simple URL shortener, written in Go.

## Quick Start

### Docker Compose (Recommended)

The easiest way to run mjr.wtf with PostgreSQL is using Docker Compose:

```bash
# 1. Copy and configure environment variables (optional - has defaults)
cp .env.example .env
# Edit .env to set your configuration (especially AUTH_TOKEN and POSTGRES_PASSWORD)

# 2. Start all services (app + PostgreSQL)
make docker-compose-up

# 3. Run database migrations (required on first start)
# Build the migrate tool
make build-migrate

# Run migrations against the PostgreSQL database
export DATABASE_URL=postgresql://mjrwtf:INSECURE_CHANGE_ME@localhost:5432/mjrwtf
# Or if you changed the credentials in .env, replace the values manually:
# export DATABASE_URL=postgresql://your-user:your-password@localhost:5432/your-db
./bin/migrate up

# 4. Verify the application is running
curl http://localhost:8080/health

# View logs
make docker-compose-logs

# Stop all services
make docker-compose-down
```

**Note:** Database migrations must be run before the application can create or retrieve URLs. The migrate tool is run from your host machine and connects to PostgreSQL through the exposed port 5432.

The docker-compose configuration includes:
- Go application server
- PostgreSQL database with persistent storage
- Automatic health checks and restart policies
- Networked services with proper dependencies

### Docker (Single Container)

To run just the application container:

```bash
# Build the Docker image
make docker-build

# Copy and configure environment variables
cp .env.example .env
# Edit .env to set your configuration (especially AUTH_TOKEN and DATABASE_URL)

# Run the container
make docker-run
```

See [docs/docker.md](docs/docker.md) for comprehensive Docker documentation, including advanced configurations and production deployment guidelines.

### Local Development

```bash
# Install dependencies
go mod download

# Run database migrations
export DATABASE_URL=./database.db
make migrate-up

# Build and run the server
make build-server
./bin/server
```

## Authentication

The API uses token-based authentication to protect URL creation and deletion endpoints. Authentication is implemented using Bearer tokens.

### Configuration

Set the `AUTH_TOKEN` environment variable to configure your authentication token:

```bash
export AUTH_TOKEN=your-secret-token-here
```

Or add it to your `.env` file:

```
AUTH_TOKEN=your-secret-token-here
```

**Security Note:** The AUTH_TOKEN is required for the application to start. Choose a strong, randomly generated token for production deployments.

### Making Authenticated Requests

Include the token in the `Authorization` header with the `Bearer` scheme:

```bash
curl -X POST https://mjr.wtf/api/urls \
  -H "Authorization: Bearer your-secret-token-here" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "short_code": "abc123"}'
```

### Authentication Responses

- **200 OK** - Request succeeded with valid token
- **401 Unauthorized** - Missing, invalid, or malformed token

Example error responses:

```text
// Missing Authorization header
Unauthorized: missing authorization header

// Invalid format (not "Bearer <token>")
Unauthorized: invalid authorization format

// Token doesn't match configured AUTH_TOKEN
Unauthorized: invalid token
```

### Protected Endpoints

The following endpoints require authentication:
- `POST /api/urls` - Create a new short URL
- `DELETE /api/urls/:code` - Delete a short URL

Public endpoints (no authentication required):
- `GET /:code` - Redirect to original URL
- `GET /health` - Health check

## Database Migrations

This project uses [goose](https://github.com/pressly/goose) for database migrations, supporting both SQLite and PostgreSQL.

### Prerequisites

Set the `DATABASE_URL` environment variable to your database connection string:

```bash
# For SQLite
export DATABASE_URL=./database.db

# For PostgreSQL
export DATABASE_URL=postgresql://user:password@localhost:5432/mjrwtf
```

Alternatively, you can copy `.env.example` to `.env` and configure it there.

### Migration Commands

The following Makefile targets are available for managing migrations:

```bash
# Apply all pending migrations
make migrate-up

# Rollback the most recent migration
make migrate-down

# Show migration status
make migrate-status

# Create a new migration file
make migrate-create NAME=add_new_feature

# Reset all migrations (caution: destroys data)
make migrate-reset
```

### Manual Migration Management

You can also use the migrate tool directly:

```bash
# Build the migrate tool
go build -o bin/migrate ./cmd/migrate

# Run migrations with explicit driver and URL
./bin/migrate -driver sqlite3 -url ./database.db up

# Run migrations for PostgreSQL
./bin/migrate -driver postgres -url "postgresql://user:pass@localhost/dbname" up

# Show help
./bin/migrate
```

### Migration Files

Migration files are located in:
- `internal/migrations/sqlite/` - SQLite-specific migrations
- `internal/migrations/postgres/` - PostgreSQL-specific migrations

Each migration consists of:
- An `.sql` file with `-- +goose Up` and `-- +goose Down` sections
- The "Up" section applies the migration
- The "Down" section reverts the migration

### Creating New Migrations

To create a new migration:

```bash
# For SQLite (default)
make migrate-create NAME=add_users_table

# For PostgreSQL
DATABASE_URL=postgresql://... make migrate-create NAME=add_users_table
```

This will create a new timestamped migration file in the appropriate directory.

### Embedded Migrations

Migrations are embedded in the binary at build time, so the migrate tool is self-contained and doesn't require external migration files at runtime.

## Testing

The project includes comprehensive unit and integration tests.

### Running Tests

```bash
# Run all tests (unit + integration)
make test

# Run unit tests only (fast, excludes integration tests)
make test-unit

# Run integration tests only
make test-integration

# Run with coverage report
go test -cover ./...

# Run specific test suite
go test -v -run TestE2E ./internal/infrastructure/http/server/
```

### Integration Tests

The integration test suite provides end-to-end testing of the entire application:
- **Full workflow testing** - authenticate → create URL → redirect → verify analytics
- **API endpoint testing** - All REST endpoints with authentication
- **Error scenario testing** - Invalid inputs, auth failures, not found errors
- **Concurrent operation testing** - Thread safety and race conditions
- **Database integration** - Uses in-memory SQLite for fast, isolated tests

See [docs/INTEGRATION_TESTING.md](docs/INTEGRATION_TESTING.md) for comprehensive integration testing documentation.

### Test Coverage

Current test coverage:
- HTTP API Endpoints: 100%
- Authentication: 100%
- URL Creation & Redirects: 100%
- Analytics: 100%
- Error Handling: 100%

All tests run in CI/CD with no external dependencies required.
