---
title: Docker Compose testing
description: Manual testing checklist for docker-compose.yml.
---


This document provides a testing plan for the SQLite-only `docker-compose.yml` configuration.

## Pre-Testing Validation

```bash
# Validate docker-compose.yml syntax
docker compose config

# List services
docker compose config --services
# Expected output: mjrwtf
```

## Testing Checklist

### 1. First run setup (data + auth)

- [ ] Create the persistent data directory: `mkdir -p data`

- [ ] Configure auth (at least one of):
  - `AUTH_TOKENS` (preferred; comma-separated)
  - `AUTH_TOKEN` (legacy; used only if `AUTH_TOKENS` is unset)

> Note: the container runs `./migrate up` automatically on startup (see `docker-entrypoint.sh`) and will create/apply the SQLite schema in `/app/data/database.db`.

### 2. Basic Startup Tests

- [ ] Start service: `make docker-compose-up`
- [ ] Verify service is running: `make docker-compose-ps`
- [ ] Verify no errors in logs: `make docker-compose-logs`

### 3. Service Health Checks

- [ ] Manual health check: `curl http://localhost:8080/health` (expect HTTP 200)
- [ ] Readiness check: `curl http://localhost:8080/ready` (expect HTTP 200)

### 4. Data Persistence

- [ ] Create a short URL (note the returned `short_code`):
  ```bash
  TOKEN=your-token-here
  curl -X POST http://localhost:8080/api/urls \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"original_url": "https://example.com"}'
  ```
- [ ] Stop service: `make docker-compose-down`
- [ ] Start service again: `make docker-compose-up`
- [ ] Verify redirect still works: `curl -L http://localhost:8080/<short_code_from_response>`
- [ ] Verify the DB file exists: `ls -l ./data/database.db`

### 5. Cleanup

- [ ] Stop service: `make docker-compose-down`
- [ ] Remove local data (DESTROYS DATA): `rm -rf ./data`
