# Docker Compose Testing Guide (SQLite)

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

### 1. Migrations (required on first run)

- [ ] Create the persistent data directory: `mkdir -p data`
- [ ] Run migrations on the host:
  ```bash
  export DATABASE_URL=./data/database.db
  make migrate-up
  ```

### 2. Basic Startup Tests

- [ ] Start service: `make docker-compose-up`
- [ ] Verify service is running: `make docker-compose-ps`
- [ ] Verify no errors in logs: `make docker-compose-logs`

### 3. Service Health Checks

- [ ] Manual health check: `curl http://localhost:8080/health` (expect HTTP 200)
- [ ] Readiness check: `curl http://localhost:8080/ready` (expect HTTP 200)

### 4. Data Persistence

- [ ] Create a short URL:
  ```bash
  curl -X POST http://localhost:8080/api/urls \
    -H "Authorization: Bearer INSECURE_CHANGE_ME" \
    -H "Content-Type: application/json" \
    -d '{"url": "https://example.com", "short_code": "test123"}'
  ```
- [ ] Stop service: `make docker-compose-down`
- [ ] Start service again: `make docker-compose-up`
- [ ] Verify redirect still works: `curl -L http://localhost:8080/test123`
- [ ] Verify the DB file exists: `ls -l ./data/database.db`

### 5. Cleanup

- [ ] Stop service: `make docker-compose-down`
- [ ] Remove local data (DESTROYS DATA): `rm -rf ./data`
