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

Example `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mjrwtf:
    build: .
    image: mjrwtf:latest
    container_name: mjrwtf
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgresql://mjrwtf:CHANGE_ME_DB_PASSWORD@postgres:5432/mjrwtf
      - AUTH_TOKEN=${AUTH_TOKEN}
      - SERVER_PORT=8080
      - LOG_LEVEL=info
      - LOG_FORMAT=json
      - GEOIP_ENABLED=false
    depends_on:
      - postgres
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s

  postgres:
    image: postgres:16-alpine
    container_name: mjrwtf-postgres
    environment:
      - POSTGRES_DB=mjrwtf
      - POSTGRES_USER=mjrwtf
      - POSTGRES_PASSWORD=CHANGE_ME_DB_PASSWORD
    volumes:
      - postgres-data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  postgres-data:
```

Start with Docker Compose:

```bash
docker-compose up -d
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

**Option 1: Run migrations locally before deploying**

```bash
# Install the migrate tool locally
make build-migrate

# Run migrations
export DATABASE_URL=your-db-url
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
