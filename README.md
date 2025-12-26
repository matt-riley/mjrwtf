# mjr.wtf - url shortener

A simple URL shortener, written in Go.

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
- `GET /api/urls` - List user's URLs with pagination (requires auth)
- `DELETE /api/urls/{shortCode}` - Delete URL (requires auth)
- `GET /api/urls/{shortCode}/analytics` - Get analytics with optional time range (requires auth)
- `GET /{shortCode}` - Redirect to original URL (public)
- `GET /health` - Health check (public)
- `GET /metrics` - Prometheus metrics (optional auth)

See the [OpenAPI specification](openapi.yaml) for detailed request/response schemas, authentication, and error handling.

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

The application supports two authentication methods:

### 1. Bearer Token Authentication (API Endpoints)

The API uses token-based authentication to protect URL creation and deletion endpoints. Authentication is implemented using Bearer tokens.

#### Configuration

Set the `AUTH_TOKEN` environment variable to configure your authentication token:

```bash
export AUTH_TOKEN=your-secret-token-here
```

Or add it to your `.env` file:

```
AUTH_TOKEN=your-secret-token-here
```

**Security Note:** The AUTH_TOKEN is required for the application to start. Choose a strong, randomly generated token for production deployments.

#### Making Authenticated Requests

Include the token in the `Authorization` header with the `Bearer` scheme:

```bash
curl -X POST https://mjr.wtf/api/urls \
  -H "Authorization: Bearer your-secret-token-here" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "short_code": "abc123"}'
```

#### Authentication Responses

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

### 2. Session-Based Authentication (Dashboard)

The web dashboard uses server-side session management with httpOnly cookies for enhanced security.

#### How It Works

1. **Login**: Navigate to `/login` and enter your authentication token (the same `AUTH_TOKEN` configured above)
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
- `POST /api/urls` - Create a new short URL (Bearer token)
- `DELETE /api/urls/:code` - Delete a short URL (Bearer token or session)
- `GET /dashboard` - View URL dashboard (session required)

Public endpoints (no authentication required):
- `GET /` - Home page
- `GET /create` - URL creation form
- `GET /login` - Login page
- `GET /:code` - Redirect to original URL
- `GET /health` - Health check
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
| `DATABASE_URL` | Database connection string (SQLite: `./database.db`, PostgreSQL: `postgresql://user:pass@host:port/db`) | - | ✓ |
| `AUTH_TOKEN` | Secret token for API authentication | - | ✓ |
| `SERVER_PORT` | HTTP server port | `8080` | |
| `BASE_URL` | Base URL for shortened links | `http://localhost:8080` | |

### Database Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DB_TIMEOUT` | Timeout for database operations (e.g., `5s`, `100ms`, `1m`) | `5s` | |

The `DB_TIMEOUT` setting applies a bounded deadline to all database operations, preventing queries from hanging indefinitely under network or database issues. This helps ensure that the application remains responsive even when the database is slow or experiencing problems.

### Security Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ALLOWED_ORIGINS` | CORS allowed origins (comma-separated) | `*` | |
| `SECURE_COOKIES` | Enable secure cookies (requires HTTPS) | `false` | |
| `METRICS_AUTH_ENABLED` | Require authentication for `/metrics` endpoint | `false` | |

### Optional Features

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DISCORD_WEBHOOK_URL` | Discord webhook URL for error notifications | - | |
| `GEOIP_ENABLED` | Enable GeoIP location tracking | `false` | |
| `GEOIP_DATABASE` | Path to GeoIP database file | - | Required if `GEOIP_ENABLED=true` |

### Logging Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LOG_LEVEL` | Logging level (`debug`, `info`, `warn`, `error`) | `info` | |
| `LOG_FORMAT` | Log format (`json`, `pretty`) | `json` | |

### Example Configuration

```bash
# .env file
DATABASE_URL=postgresql://mjrwtf:password@localhost:5432/mjrwtf
AUTH_TOKEN=your-secret-token-here
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

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
