---
title: Docker
description: Build and run mjr.wtf using Docker.
---


This guide explains how to build and run the mjr.wtf URL shortener using Docker.

## Quick Start with Pre-built Images

The easiest way to get started is using the pre-built multi-arch images from GitHub Container Registry:

```bash
# Pull the latest version
docker pull ghcr.io/matt-riley/mjrwtf:latest

# Run with environment file
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  --env-file .env \
  ghcr.io/matt-riley/mjrwtf:latest
```

Available image tags:
- `latest` - Latest published release
- `1.2.3` - Specific semantic version (from the release tag `mjrwtf-v1.2.3`)
- `1.2` - Latest patch version in the 1.2.x series
- `1` - Latest minor and patch version in the 1.x series

All images support both `linux/amd64` and `linux/arm64` architectures.

## Overview

The project includes a multi-stage Dockerfile that:
- Uses `golang:1.25-alpine` for building (with CGO support for SQLite)
- Uses `alpine:latest` for the runtime image
- Creates a minimal, secure container (~20-30MB without GeoIP database)
- Runs as a non-root user for security
- Includes health checks

## Building the Docker Image

### Using Pre-built Images (Recommended)

Production-ready images are automatically built and published to GitHub Container Registry on each release:

```bash
# Pull a specific version
docker pull ghcr.io/matt-riley/mjrwtf:1.0.0

# Pull the latest version
docker pull ghcr.io/matt-riley/mjrwtf:latest

# Tag for local use
docker tag ghcr.io/matt-riley/mjrwtf:latest mjrwtf:latest
```

### Building Locally

```bash
# Build the image
docker build -t mjrwtf:latest .

# Build with a specific tag
docker build -t mjrwtf:1.0.0 .
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
  -e AUTH_TOKENS=your-secret-token \
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

### With SQLite and Persistent Storage

```bash
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  -v ./data:/app/data \
  -e DATABASE_URL=/app/data/database.db \
  -e AUTH_TOKENS=your-secret-token \
  mjrwtf:latest
```

### With GeoIP Database

```bash
docker run -d \
  --name mjrwtf \
  -p 8080:8080 \
  -v ./GeoLite2-Country.mmdb:/app/geoip.mmdb:ro \
  -e DATABASE_URL=./database.db \
  -e AUTH_TOKENS=your-secret-token \
  -e GEOIP_ENABLED=true \
  -e GEOIP_DATABASE=/app/geoip.mmdb \
  mjrwtf:latest
```

## Docker Compose

The project includes a `docker-compose.yml` file that runs the server with a persistent SQLite database in a bind-mounted `./data` directory.

### Quick Start

```bash
# 1. Copy environment variables
cp .env.example .env
# Edit .env to set AUTH_TOKENS (recommended) or AUTH_TOKEN

# 2. Prepare a persistent data directory and run migrations (required on first run)
mkdir -p data
export DATABASE_URL=./data/database.db
make migrate-up

# 3. Start the server
make docker-compose-up

# 4. Verify the application is ready
curl http://localhost:8080/health
curl http://localhost:8080/ready

# View logs
make docker-compose-logs

# Stop the service
make docker-compose-down
```

**Note:** The container uses `/app/data/database.db`; the bind mount maps that to `./data/database.db` on the host.

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

**Port already in use:**
```bash
# Check what's using port 8080
lsof -i :8080

# Use a different port (edit docker-compose.yml)
# Change "8080:8080" to "9090:8080"
```

## Environment Variables

See `.env.example` for all available environment variables:

- `DATABASE_URL` - SQLite database file path (required)
- `AUTH_TOKENS` - API authentication tokens (recommended; comma-separated; takes precedence)
- `AUTH_TOKEN` - API authentication token (legacy; used only if AUTH_TOKENS is unset)
- `SERVER_PORT` - HTTP server port (default: 8080)
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: json, pretty (default: json)
- `ALLOWED_ORIGINS` - CORS allowed origins (default: *)
- `GEOIP_ENABLED` - Enable GeoIP tracking (default: false)
- `GEOIP_DATABASE` - Path to GeoIP database (required if GEOIP_ENABLED=true)

## Health Checks

The container includes a health check that queries the `/health` endpoint (liveness). For orchestrator-style readiness probes, use `/ready`.

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
# - Missing AUTH_TOKENS/AUTH_TOKEN environment variable
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

The container doesn't automatically run migrations. Run them separately before starting the application.

**Option 1: Run migrations from the host (recommended)**

```bash
# Build the migrate tool
make build-migrate

# Run migrations against your SQLite file
export DATABASE_URL=./data/database.db
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
  -e AUTH_TOKENS=dev-token \
  mjrwtf:latest
```

## Production Deployment

For production:

1. Use specific image tags (not `latest`)
2. Set strong `AUTH_TOKENS` (recommended) or `AUTH_TOKEN`
3. Configure proper CORS (`ALLOWED_ORIGINS`)
4. Use `LOG_FORMAT=json` for structured logging
5. Mount a volume for persistent SQLite data
6. Use secrets management (Docker secrets, Kubernetes secrets, etc.)
7. Run behind a reverse proxy (nginx, Traefik, etc.)

Example production run:

```bash
docker run -d \
  --name mjrwtf \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v /srv/mjrwtf:/app/data \
  -e DATABASE_URL=/app/data/database.db \
  -e AUTH_TOKENS=$(cat /run/secrets/auth_token) \
  -e LOG_LEVEL=info \
  -e LOG_FORMAT=json \
  -e ALLOWED_ORIGINS=https://mjr.wtf \
  mjrwtf:v1.0.0
```
