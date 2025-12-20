# Docker Guide for mjr.wtf

This guide explains how to build and run the mjr.wtf URL shortener using Docker.

## Overview

The project includes a multi-stage Dockerfile that:
- Uses `golang:1.24-alpine` for building (with CGO support for SQLite)
- Uses `alpine:latest` for the runtime image
- Creates a minimal, secure container (~20-30MB without GeoIP database)
- Runs as a non-root user for security
- Includes health checks

## Building the Docker Image

```bash
# Build the image
docker build -t mjrwtf:latest .

# Build with a specific tag
docker build -t mjrwtf:v1.0.0 .
```

The build process:
1. Installs build dependencies (gcc, musl-dev for CGO)
2. Downloads Go dependencies
3. Generates Templ templates
4. Compiles the server binary with CGO enabled
5. Creates a minimal runtime image with only the binary

## Running the Container

### Basic Usage

```bash
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  -e DATABASE_URL=./database.db \
  -e AUTH_TOKEN=your-secret-token \
  mjrwtf:latest
```

### With Environment File

Create a `.env` file (copy from `.env.example`):

```bash
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  --env-file .env \
  mjrwtf:latest
```

### With PostgreSQL

```bash
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  -e DATABASE_URL=postgresql://user:CHANGE_ME_DB_PASSWORD@postgres:5432/mjrwtf \
  -e AUTH_TOKEN=your-secret-token \
  --network mynetwork \
  mjrwtf:latest
```

### With SQLite and Persistent Storage

```bash
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  -v ./data:/app/data \
  -e DATABASE_URL=/app/data/database.db \
  -e AUTH_TOKEN=your-secret-token \
  mjrwtf:latest
```

### With GeoIP Database

```bash
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  -v ./GeoLite2-Country.mmdb:/app/geoip.mmdb:ro \
  -e DATABASE_URL=./database.db \
  -e AUTH_TOKEN=your-secret-token \
  -e GEOIP_ENABLED=true \
  -e GEOIP_DATABASE=/app/geoip.mmdb \
  mjrwtf:latest
```

## Docker Compose

The project includes a `docker-compose.yml` file that sets up the complete application stack with PostgreSQL.

### Quick Start

```bash
# 1. Copy environment variables (optional - has sensible defaults)
cp .env.example .env
# Edit .env to customize AUTH_TOKEN and database credentials

# 2. Start all services
make docker-compose-up
# or: docker compose up -d

# 3. Run database migrations (REQUIRED on first start)
# The migrate binary is not included in the Docker image, so run it from the host
make build-migrate

# Run migrations using the default credentials
export DATABASE_URL=postgresql://mjrwtf:INSECURE_CHANGE_ME@localhost:5432/mjrwtf
# Or if you changed the credentials in .env, replace the values manually:
# export DATABASE_URL=postgresql://your-user:your-password@localhost:5432/your-db
./bin/migrate up

# 4. Verify the application is ready
curl http://localhost:8080/health
# Expected output: {"status":"ok"}

# View logs
make docker-compose-logs
# or: docker compose logs -f

# Check service status
make docker-compose-ps
# or: docker compose ps

# Stop all services
make docker-compose-down
# or: docker compose down
```

**Important:** The database migrations step (step 3) is required before the application can function. Without migrations, the database tables won't exist and the application will fail when trying to create or retrieve URLs.

### Services Included

The Docker Compose stack includes:

1. **mjrwtf** - The Go application server
   - Exposed on port 8080
   - Automatically builds from the Dockerfile
   - Waits for PostgreSQL to be healthy before starting
   - Includes health checks and automatic restart

2. **postgres** - PostgreSQL 16 database
   - Data persisted in named volume `postgres-data`
   - Exposed on port 5432 for running migrations from host
   - Includes health checks
   - Automatically restarts on failure

### Configuration

The docker-compose.yml uses environment variables with sensible defaults:

```bash
# Database credentials (set in .env or use defaults)
POSTGRES_DB=mjrwtf
POSTGRES_USER=mjrwtf
POSTGRES_PASSWORD=INSECURE_CHANGE_ME  # CHANGE THIS!

# Application configuration
AUTH_TOKEN=INSECURE_CHANGE_ME  # CHANGE THIS!
LOG_LEVEL=info
LOG_FORMAT=json
ALLOWED_ORIGINS=*
```

**Important:** The default passwords are insecure. Always change `AUTH_TOKEN` and `POSTGRES_PASSWORD` in production.

### Data Persistence

Database data is stored in a named Docker volume (`postgres-data`) that persists across container restarts:

```bash
# List volumes
docker volume ls

# Inspect the postgres data volume
docker volume inspect mjrwtf_postgres-data

# Backup the database
docker compose exec postgres pg_dump -U mjrwtf mjrwtf > backup.sql

# Restore from backup
docker compose exec -T postgres psql -U mjrwtf mjrwtf < backup.sql
```

### Networking

Services communicate over an automatic Docker network:
- The app connects to PostgreSQL using the hostname `postgres`
- PostgreSQL port 5432 is exposed to the host for running migrations
- Port 8080 (app) is exposed to the host for accessing the application
- For production deployments, you can remove the PostgreSQL port exposure after initial setup

### Testing the Stack

```bash
# Start services
docker compose up -d

# Wait for services to be healthy
docker compose ps

# Run database migrations (required on first start)
make build-migrate
export DATABASE_URL=postgresql://mjrwtf:INSECURE_CHANGE_ME@localhost:5432/mjrwtf
./bin/migrate up

# Test the health endpoint
curl http://localhost:8080/health

# Create a short URL (requires AUTH_TOKEN)
curl -X POST http://localhost:8080/api/urls \
  -H "Authorization: Bearer INSECURE_CHANGE_ME" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "short_code": "test123"}'

# Test redirect
curl -L http://localhost:8080/test123

# View logs
docker compose logs -f mjrwtf

# Stop services
docker compose down
```

### Troubleshooting

**Services won't start:**
```bash
# Check service status
docker compose ps

# View logs
docker compose logs

# Rebuild containers
docker compose up -d --build
```

**Database connection errors:**
```bash
# Verify postgres is healthy
docker compose ps postgres

# Check postgres logs
docker compose logs postgres

# Connect to postgres directly
docker compose exec postgres psql -U mjrwtf -d mjrwtf
```

**Port already in use:**
```bash
# Check what's using port 8080
lsof -i :8080

# Use a different port (edit docker-compose.yml)
# Change "8080:8080" to "9090:8080"
```

### Production Deployment

For production use:

1. **Change default credentials:**
   ```bash
   # Set in .env file
   AUTH_TOKEN=your-strong-random-token-here
   POSTGRES_PASSWORD=your-strong-random-password-here
   ```

2. **Configure CORS:**
   ```bash
   ALLOWED_ORIGINS=https://yourdomain.com
   ```

3. **Use specific image tags:**
   ```yaml
   services:
     mjrwtf:
       image: mjrwtf:x.y.z  # Use actual version tag, not 'latest'
   ```

4. **Run behind a reverse proxy:**
   - Use nginx, Traefik, or Caddy for TLS termination
   - Don't expose port 8080 directly to the internet

5. **Secure PostgreSQL access (optional):**
   - After running initial migrations, you can remove the PostgreSQL port exposure
   - Comment out the `ports` section in the postgres service in docker-compose.yml
   - This prevents external access to the database
   - Migrations can still be run from within the Docker network if needed

6. **Regular backups:**
   ```bash
   # Automated backup script
   docker compose exec postgres pg_dump -U mjrwtf mjrwtf | gzip > backup-$(date +%Y%m%d).sql.gz
   ```

7. **Monitor logs:**
   ```bash
   docker compose logs -f --tail=100
   ```

## Environment Variables

See `.env.example` for all available environment variables:

- `DATABASE_URL` - Database connection string (required)
- `AUTH_TOKEN` - API authentication token (required)
- `SERVER_PORT` - HTTP server port (default: 8080)
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: json, pretty (default: json)
- `ALLOWED_ORIGINS` - CORS allowed origins (default: *)
- `GEOIP_ENABLED` - Enable GeoIP tracking (default: false)
- `GEOIP_DATABASE` - Path to GeoIP database (required if GEOIP_ENABLED=true)

## Health Checks

The container includes a health check that queries the `/health` endpoint:

```bash
# Check container health
docker ps

# View health check logs
docker inspect --format='{{json .State.Health}}' mjrwtf | jq
```

The health check configuration:
- **Interval:** 30 seconds
- **Timeout:** 3 seconds
- **Start Period:** 5 seconds
- **Retries:** 3

## Image Size

Expected image sizes:
- **Without GeoIP database:** ~20-30MB
- **With GeoIP database:** ~30-80MB (depending on database size)

Check image size:

```bash
docker images mjrwtf
```

## Security

The Docker image follows security best practices:

1. **Non-root User:** Runs as user `appuser` (UID 1000)
2. **Minimal Base:** Uses Alpine Linux for small attack surface
3. **No Secrets:** Secrets are passed via environment variables, not baked in
4. **Read-only GeoIP:** GeoIP database mounted read-only when used
5. **Health Checks:** Monitors application health

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs mjrwtf

# Common issues:
# - Missing AUTH_TOKEN environment variable
# - Invalid DATABASE_URL
# - Database connection failed
```

### Health Check Failing

```bash
# Check if the service is responding
docker exec mjrwtf curl -f http://localhost:8080/health

# Check application logs
docker logs mjrwtf
```

### Port Already in Use

```bash
# Use a different port
docker run -p 9090:8080 mjrwtf:latest
```

### Database Migrations

The container doesn't automatically run migrations. You need to run migrations separately before starting the application.

**For Docker Compose:** See the Quick Start section above for step-by-step migration instructions. PostgreSQL is exposed on port 5432 to allow running migrations from the host.

**For other deployments:**

**Option 1: Run migrations from host**

```bash
# Build the migrate tool
make build-migrate

# Run migrations (adjust DATABASE_URL for your setup)
export DATABASE_URL=postgresql://user:password@host:5432/dbname
./bin/migrate up
```

**Option 2: Use a separate migration job**

Create a separate Docker image or Kubernetes Job that includes the migrate binary, or run migrations as part of your deployment pipeline before starting the application containers.

The main application container only includes the server binary for security and minimal image size.

## Development

For development, you might want to mount the source code and rebuild:

```bash
docker run -d \
  --name mjrwtf-dev \
  -p 8080:8080 \
  -v $(pwd):/app \
  -e DATABASE_URL=./database.db \
  -e AUTH_TOKEN=dev-token \
  mjrwtf:latest
```

## Production Deployment

For production:

1. Use specific image tags (not `latest`)
2. Set strong `AUTH_TOKEN`
3. Use PostgreSQL instead of SQLite
4. Configure proper CORS (`ALLOWED_ORIGINS`)
5. Use `LOG_FORMAT=json` for structured logging
6. Mount volumes for persistent data
7. Use secrets management (Docker secrets, Kubernetes secrets, etc.)
8. Run behind a reverse proxy (nginx, Traefik, etc.)

Example production run:

```bash
docker run -d \
  --name mjrwtf \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -e DATABASE_URL=postgresql://user:pass@db:5432/mjrwtf \
  -e AUTH_TOKEN=$(cat /run/secrets/auth_token) \
  -e LOG_LEVEL=info \
  -e LOG_FORMAT=json \
  -e ALLOWED_ORIGINS=https://mjr.wtf \
  mjrwtf:v1.0.0
```
