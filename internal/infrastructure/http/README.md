# HTTP Infrastructure

This package provides the HTTP server infrastructure for the mjr.wtf URL shortener: routing, middleware, and handlers.

## Structure

```
internal/infrastructure/http/
├── handlers/    # HTTP handlers (HTML + API)
├── middleware/  # HTTP middleware components
└── server/      # Server wiring (router + routes)
```

Templ templates live in `internal/adapters/http/templates/` (see that README for details).

## Components

### Server (`server/`)

The server is built around `chi` and wires together:

- Global middleware stack (panic recovery, request IDs, logging, metrics, sessions, CORS, etc.)
- Routes for:
  - Liveness: `GET /health`
  - Readiness (DB ping with timeout): `GET /ready`
  - Metrics: `GET /metrics` (optionally authenticated)
  - HTML pages (home/create/login/dashboard)
  - Redirects: `GET /{shortCode}`
  - API: `/api/urls/*`

A complete working example of constructing and running the server is in `cmd/server/main.go`.

### Handlers (`handlers/`)

- `PageHandler` — HTML pages (and session auth flows)
- `URLHandler` — API URL create/list/delete
- `RedirectHandler` — short URL redirects (records clicks asynchronously)
- `AnalyticsHandler` — URL analytics endpoint(s)

### Middleware (`middleware/`)

The global middleware stack (order matters) is configured in `server.New(...)`:

1. Recovery (with optional Discord notifier)
2. Request ID
3. Security headers
4. Logger injection (request-scoped logger)
5. Request logger
6. Prometheus metrics
7. Session middleware
8. CORS

Route-specific auth is applied at the router level:

- `/dashboard` requires a session
- `/api/urls/*` uses `SessionOrBearerAuth(...)` (session for browser dashboard flows, bearer token for API clients)
- `/metrics` is public by default; enable auth with `METRICS_AUTH_ENABLED=true`

## Testing

```bash
# Unit + integration tests under this package
go test ./internal/infrastructure/http/...
```

## Configuration (common)

This package is configured via `internal/infrastructure/config` env vars; commonly relevant here:

- `SERVER_PORT` (default: `8080`)
- `BASE_URL` (default: `http://localhost:8080`)
- `ALLOWED_ORIGINS` (default: `*`, comma-separated)
- `DATABASE_URL` (SQLite file path)
- `AUTH_TOKENS` (preferred; comma-separated) or `AUTH_TOKEN` (legacy)
- `SECURE_COOKIES` (set true when behind HTTPS)
- `REDIRECT_RATE_LIMIT_PER_MINUTE` (default: `120`)
- `API_RATE_LIMIT_PER_MINUTE` (default: `60`)
- `METRICS_AUTH_ENABLED` (default: `false`)
- `ENABLE_HSTS` (default: `false`)
- `DB_TIMEOUT` (default: `5s`)
- `LOG_STACK_TRACES` (default: `true`)
- `DISCORD_WEBHOOK_URL` (optional)

See `.env.example` for the full set.
