# mjr.wtf - url shortener

A simple URL shortener, written in Go.

## Contents

- [Quick Start](#quick-start)
- [API Documentation](#api-documentation)
- [Authentication](#authentication)
- [Configuration](#configuration)
- [TUI CLI](docs/TUI.md)
- [Database Migrations](#database-migrations)
- [Testing](#testing)
- [Local Development](#local-development)
- [Releases](#releases)
- [License](#license)

## Quick Start

### Docker Compose (SQLite)

The easiest way to run mjr.wtf locally is using Docker Compose with a persistent SQLite database (bind-mounted to `./data`):

```bash
# 1. Copy and configure environment variables
cp .env.example .env
# Edit .env to set AUTH_TOKEN (required)

# 2. Prepare a persistent data directory and run migrations (required on first run)
mkdir -p data
export DATABASE_URL=./data/database.db
make migrate-up

# 3. Start the server
make docker-compose-up

# 4. Verify the application is running
# Liveness:
curl http://localhost:8080/health

# Readiness (checks DB connectivity):
curl http://localhost:8080/ready

# View logs
make docker-compose-logs

# Stop the service
make docker-compose-down
```

**Note:** The container uses `/app/data/database.db`; the bind mount maps that to `./data/database.db` on the host.

### Docker (Single Container)

#### Using Pre-built Images from GitHub Container Registry

Production-ready multi-arch images (amd64/arm64) are available from GitHub Container Registry:

```bash
# Pull the latest version
docker pull ghcr.io/matt-riley/mjrwtf:latest

# Or pull a specific version
docker pull ghcr.io/matt-riley/mjrwtf:v1.0.0

# Copy and configure environment variables
cp .env.example .env
# Edit .env to set your configuration (especially AUTH_TOKEN and DATABASE_URL)

# Run the container
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  --env-file .env \
  ghcr.io/matt-riley/mjrwtf:latest
```

#### Building Locally

To build and run the image locally:

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

## API Documentation

The API is fully documented using OpenAPI 3.0. You can find the specification in [`openapi.yaml`](openapi.yaml) at the repository root.

### Viewing the API Documentation

**Interactive API Explorer:**
- [SwaggerUI](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/matt-riley/mjrwtf/main/openapi.yaml) - Interactive API documentation
- [ReDoc](https://redocly.github.io/redoc/?url=https://raw.githubusercontent.com/matt-riley/mjrwtf/main/openapi.yaml) - Clean, responsive API reference

**Local viewing:**
```bash
# Validate the OpenAPI spec
make validate-openapi

# Or validate manually with swagger-cli
npm install -g @apidevtools/swagger-cli
swagger-cli validate openapi.yaml
```

### API Endpoints

- `POST /api/urls` - Create shortened URL (requires auth)
- `GET /api/urls` - List URLs with pagination (requires auth)
- `DELETE /api/urls/{shortCode}` - Delete URL (requires auth)
- `GET /api/urls/{shortCode}/analytics` - Get analytics with optional time range (requires auth)
- `GET /{shortCode}` - Redirect to original URL (public)
- `GET /health` - Health check / liveness (public)
- `GET /ready` - Readiness check (public)
- `GET /metrics` - Prometheus metrics (optional auth)

See the [OpenAPI specification](openapi.yaml) for detailed request/response schemas, authentication, and error handling.

### Rate Limiting

Per-client rate limits protect the redirect endpoint and authenticated API routes. Configure limits via environment variables:

- `REDIRECT_RATE_LIMIT_PER_MINUTE` (default: 120) - Requests per minute per IP for `GET /{shortCode}`
- `API_RATE_LIMIT_PER_MINUTE` (default: 60) - Requests per minute per IP for `/api` routes

Requests exceeding the limit return HTTP 429 with a `Retry-After` header indicating when to retry.

**Note:** The rate limiter keys by client IP (using `X-Forwarded-For` / `X-Real-IP` when present). In production, ensure your reverse proxy strips/overwrites any client-provided forwarding headers; otherwise malicious clients can spoof these headers to bypass limits.

## Authentication

The application supports two authentication methods:

### 1. Bearer Token Authentication (API Endpoints)

The API uses token-based authentication to protect URL creation and deletion endpoints. Authentication is implemented using Bearer tokens.

#### Configuration

mjr.wtf accepts either a single token (`AUTH_TOKEN`) or a comma-separated list (`AUTH_TOKENS`) to support token rotation.

- **Preferred:** `AUTH_TOKENS` (comma-separated). If set, it **takes precedence** and `AUTH_TOKEN` is ignored.
- **Backward compatible:** `AUTH_TOKEN` (single token) is used only when `AUTH_TOKENS` is **unset**.

```bash
# Single token (legacy)
export AUTH_TOKEN=your-secret-token-here

# Multiple tokens (recommended for rotation)
export AUTH_TOKENS=token-current,token-next
```

Or add it to your `.env` file:

```
# AUTH_TOKENS takes precedence if set
AUTH_TOKENS=token-current,token-next
# AUTH_TOKEN=your-secret-token-here
```

**Security Note:** At least one auth token is required for the application to start. Choose strong, randomly generated tokens for production deployments.

#### Recommended token rotation procedure (no downtime)

1. Generate a new token.
2. Deploy with both tokens enabled: `AUTH_TOKENS=<current>,<new>`
3. Update clients to use `<new>` and verify requests succeed.
4. Deploy again with only the new token: `AUTH_TOKENS=<new>` (or switch back to `AUTH_TOKEN=<new>` if you don’t need multi-token support).

#### Making Authenticated Requests

Include the token in the `Authorization` header with the `Bearer` scheme:

```bash
curl -X POST https://mjr.wtf/api/urls \
  -H "Authorization: Bearer your-secret-token-here" \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com"}'
```

#### Authentication Responses

- **200 OK** - Request succeeded with valid token
- **401 Unauthorized** - Missing, invalid, or malformed token

Example error responses:

```json
{"error":"Unauthorized: missing authorization header"}
{"error":"Unauthorized: invalid authorization format"}
{"error":"Unauthorized: invalid token"}
```

### 2. Session-Based Authentication (Dashboard)

The web dashboard uses server-side session management with httpOnly cookies for enhanced security.

#### How It Works

1. **Login**: Navigate to `/login` and enter any active authentication token (from `AUTH_TOKENS` or `AUTH_TOKEN`)
2. **Session Creation**: Upon successful login, the server creates a session and sets an httpOnly cookie
3. **Session Duration**: Sessions last for 24 hours and are automatically refreshed on each request
4. **Logout**: Navigate to `/logout` or click the logout button to end your session

#### Security Features

- **HttpOnly Cookies**: Session cookies cannot be accessed by JavaScript, protecting against XSS attacks
- **SameSite Protection**: Cookies are set with `SameSite=Lax` to prevent CSRF attacks
- **Secure Cookies**: Configure `SECURE_COOKIES=true` in production to ensure cookies are only sent over HTTPS
- **Automatic Expiration**: Sessions are valid for 24 hours and are automatically extended on each request
- **In-Memory Storage**: Sessions are stored in-memory on the server (lost on restart)

#### Configuration

Set the `SECURE_COOKIES` environment variable to enable secure cookies:

```bash
# For production with HTTPS
export SECURE_COOKIES=true

# For local development (default)
export SECURE_COOKIES=false
```

Or add it to your `.env` file:

```
SECURE_COOKIES=true
```

**Important:** In production deployments with HTTPS, always set `SECURE_COOKIES=true` to prevent session hijacking over insecure connections.

#### Dashboard Features

- View all your shortened URLs
- See click statistics for each URL
- Delete URLs you no longer need
- Copy short URLs to clipboard
- Pagination for large URL collections

**Note:** For production deployments with multiple server instances, consider implementing a distributed session store (Redis, database, etc.) to maintain sessions across servers.

### Protected Endpoints

The following endpoints require authentication:
- `POST /api/urls` - Create a new short URL (Bearer token or session)
- `DELETE /api/urls/{shortCode}` - Delete a short URL (Bearer token or session)
- `GET /dashboard` - View URL dashboard (session required)

**Note:** Multi-user ownership is not implemented yet; any authenticated token/session can manage URLs.

Public endpoints (no authentication required):
- `GET /` - Home page
- `GET /create` - URL creation form
- `GET /login` - Login page
- `GET /{shortCode}` - Redirect to original URL
- `GET /health` - Health check (liveness)
- `GET /ready` - Readiness check (DB connectivity)
- `GET /metrics` - Prometheus metrics (can be optionally protected, see below)

### Protecting the Metrics Endpoint

The `/metrics` endpoint exposes operational metrics for Prometheus scraping. By default, it is publicly accessible to make local development easy. However, in production deployments, you may want to restrict access to this endpoint.

#### Option 1: Enable Authentication (Recommended for Most Cases)

Set the `METRICS_AUTH_ENABLED` environment variable to `true` to require Bearer token authentication for the `/metrics` endpoint:

```bash
export METRICS_AUTH_ENABLED=true
```

Or add it to your `.env` file:

```
METRICS_AUTH_ENABLED=true
```

When enabled, the `/metrics` endpoint will require the same Bearer token as other API endpoints:

```bash
curl -H "Authorization: Bearer your-secret-token-here" https://mjr.wtf/metrics
```

#### Option 2: Reverse Proxy Restrictions

If you prefer to keep the endpoint public within your infrastructure but restrict external access, configure your reverse proxy (nginx, Caddy, etc.) to:
- Only allow access to `/metrics` from specific IP addresses (e.g., your Prometheus server)
- Serve `/metrics` on a separate port that's not publicly exposed
- Use network policies to restrict access at the infrastructure level

Example nginx configuration:

```nginx
location /metrics {
    allow 10.0.0.0/8;  # Allow internal network
    deny all;          # Deny everyone else
    proxy_pass http://localhost:8080;
}
```

**Security Note:** The metrics endpoint may expose sensitive operational information including request rates, error rates, and resource usage. Always restrict access in production environments.

## Configuration

mjr.wtf is configured through environment variables. Below is a comprehensive list of all available configuration options.

### Core Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | SQLite database file path (SQLite-only; **file paths only**; values containing `://` are rejected) (e.g. `./database.db`) | - | ✓ |
| `AUTH_TOKENS` | Secret tokens for API authentication (comma-separated); takes precedence over `AUTH_TOKEN` | - | ✓* |
| `AUTH_TOKEN` | Secret token for API authentication (legacy single-token; used only if `AUTH_TOKENS` is unset) | - | ✓* |
| `SERVER_PORT` | HTTP server port | `8080` | |
| `BASE_URL` | Base URL for shortened links | `http://localhost:8080` | |

\* At least one of `AUTH_TOKENS` or `AUTH_TOKEN` must be set.

### Database Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DB_TIMEOUT` | Timeout for database operations (e.g., `5s`, `100ms`, `1m`) | `5s` | |

The `DB_TIMEOUT` setting applies a bounded deadline to all database operations, preventing queries from hanging indefinitely under network or database issues. This helps ensure that the application remains responsive even when the database is slow or experiencing problems.

### Rate limiting

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIRECT_RATE_LIMIT_PER_MINUTE` | Requests per minute per IP for `GET /{shortCode}` | `120` | |
| `API_RATE_LIMIT_PER_MINUTE` | Requests per minute per IP for `/api/*` routes | `60` | |

### Redirect click recording (async)

Redirects enqueue click events which are processed asynchronously by worker goroutines.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIRECT_CLICK_WORKERS` | Worker goroutines for async click recording | `100` | |
| `REDIRECT_CLICK_QUEUE_SIZE` | Queue size for async click recording | `REDIRECT_CLICK_WORKERS*2` | |

### Security Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ALLOWED_ORIGINS` | CORS allowed origins (comma-separated) | `*` | |
| `SECURE_COOKIES` | Enable secure cookies (requires HTTPS) | `false` | |
| `METRICS_AUTH_ENABLED` | Require authentication for `/metrics` endpoint | `false` | |
| `ENABLE_HSTS` | Enable `Strict-Transport-Security` header (**only** behind HTTPS) | `false` | |

### Optional Features

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DISCORD_WEBHOOK_URL` | Discord webhook URL for error notifications | - | |
| `GEOIP_ENABLED` | Enable GeoIP location tracking | `false` | |
| `GEOIP_DATABASE` | Path to GeoIP database file | - | Required if `GEOIP_ENABLED=true` |

### URL status checker (optional)

mjr.wtf can periodically check whether destination URLs are returning HTTP 404/410 and record the last seen status. Redirects **never** perform a live/on-request destination check; they only consult the stored status.

If a short URL is marked as gone, `GET /{shortCode}` returns an HTML interstitial with the stored status code (404 or 410) and, when available, a Wayback Machine link.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `URL_STATUS_CHECKER_ENABLED` | Enable the periodic destination status checker | `false` | |
| `URL_STATUS_CHECKER_POLL_INTERVAL` | How often the background job runs | `5m` | |
| `URL_STATUS_CHECKER_ALIVE_RECHECK_INTERVAL` | Minimum age before re-checking URLs not marked gone | `6h` | |
| `URL_STATUS_CHECKER_GONE_RECHECK_INTERVAL` | Minimum age before re-checking URLs marked gone | `24h` | |
| `URL_STATUS_CHECKER_BATCH_SIZE` | Max URLs processed per poll | `100` | |
| `URL_STATUS_CHECKER_CONCURRENCY` | Parallel outbound checks per poll | `5` | |
| `URL_STATUS_CHECKER_ARCHIVE_LOOKUP_ENABLED` | When a URL is gone, look up an archive.org Wayback snapshot | `true` | |
| `URL_STATUS_CHECKER_ARCHIVE_RECHECK_INTERVAL` | Minimum age before re-checking Wayback availability for gone URLs | `168h` | |

**Privacy note:** When enabled, the checker makes outbound HTTP requests to destination URLs and (if archive lookup is enabled) to `https://archive.org/wayback/available`.

### Logging Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LOG_LEVEL` | Logging level (`debug`, `info`, `warn`, `error`) | `info` | |
| `LOG_FORMAT` | Log format (`json`, `pretty`) | `json` | |
| `LOG_STACK_TRACES` | Include stack traces on panic recovery logs | `true` | |

### Example Configuration

```bash
# .env file
DATABASE_URL=./database.db

# Prefer AUTH_TOKENS (supports rotation); AUTH_TOKEN is legacy/back-compat.
AUTH_TOKENS=token-current,token-next
# AUTH_TOKEN=your-secret-token-here

SERVER_PORT=8080
BASE_URL=https://mjr.wtf
DB_TIMEOUT=10s
ALLOWED_ORIGINS=https://example.com,https://app.example.com
SECURE_COOKIES=true
LOG_LEVEL=info
LOG_FORMAT=json
METRICS_AUTH_ENABLED=true
```

## Database Migrations

This project uses [goose](https://github.com/pressly/goose) for SQLite database migrations.

### Prerequisites

Set the `DATABASE_URL` environment variable to your SQLite database file path:

```bash
export DATABASE_URL=./database.db
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

# Run migrations (uses embedded SQLite migrations by default)
./bin/migrate -url ./database.db up

# Show help
./bin/migrate
```

### Migration Files

Migration files are located in:
- `internal/migrations/sqlite/` - SQLite migrations

Each migration consists of:
- An `.sql` file with `-- +goose Up` and `-- +goose Down` sections
- The "Up" section applies the migration
- The "Down" section reverts the migration

### Creating New Migrations

To create a new migration:

```bash
make migrate-create NAME=add_users_table
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

To check coverage locally:

```bash
go test -cover ./...
```

All tests run in CI/CD with no external dependencies required.

## Local Development

### Prerequisites

Before you begin, install the required code generation tools:

```bash
# Install sqlc (for database code generation, requires v1.30.0+)
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

# Install templ (for template code generation)
go install github.com/a-h/templ/cmd/templ@latest
```

### Quick Start

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

**Note:** The `make generate` step is automatically run when you use `make build`, `make test`, or `make check`, so you typically don't need to run it manually. However, if you modify SQL queries or templates, you can run it explicitly.

**Alternative:** You can also use `go generate ./...` which will run the same code generation steps.

## Releases

Releases are automated via GitHub Actions using **Release Please** (versioning + changelog) and **GoReleaser** (binary artifacts).

- Commits merged into `main` drive a **Release PR** (semver bump + `CHANGELOG.md`).
- Merging the Release PR publishes a tagged GitHub Release (`vX.Y.Z`).
- Publishing the release triggers:
  - `.github/workflows/goreleaser.yml` (using `.goreleaser.yaml`) to attach `server` + `migrate` binaries and `checksums.txt`.
  - `.github/workflows/docker-publish.yml` to build and publish the GHCR Docker images.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
