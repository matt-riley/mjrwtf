---
name: configuration-env
description: Configure mjr.wtf safely via environment variables and .env files, including DATABASE_URL, AUTH_TOKEN, CORS, cookies, rate limits, and GeoIP. Use when the server won’t start or behavior differs between environments.
license: MIT
compatibility: Requires bash, git, and Go tooling.
metadata:
  repo: mjrwtf
  runner: github-copilot-cli
  version: 1.1
allowed-tools: Bash(git:*) Bash(make:*) Bash(go:*) Read
---

## Tooling assumptions

- Use a terminal runner with bash and git available.
- Prefer `make` targets when available; fall back to direct CLI commands when needed.

## Primary references

- `.env.example` (template)
- README "Configuration" + "Authentication"

## Core variables (most common)

- `DATABASE_URL` (required):
  - SQLite: `./database.db` or `/path/to/database.db`
  - Postgres: `postgresql://user:pass@host:5432/dbname`
- `AUTH_TOKEN` (required): bearer token for API and dashboard login.
- `BASE_URL` (recommended): base URL used when constructing short links.

## Security-sensitive settings

- `SECURE_COOKIES=true` in production behind HTTPS.
- `ALLOWED_ORIGINS`: set to explicit origins in production (avoid `*` when credentials/cookies involved).
- `METRICS_AUTH_ENABLED=true` if `/metrics` should not be public.

## Optional features

- GeoIP:
  - `GEOIP_ENABLED=true`
  - `GEOIP_DATABASE=/path/to/GeoLite2-Country.mmdb`

## Fast validation

After editing env:

```bash
make build-server
./bin/server
```

Then:

```bash
curl http://localhost:8080/health
```
