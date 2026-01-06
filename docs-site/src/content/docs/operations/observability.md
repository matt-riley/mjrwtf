---
title: Observability
description: Health/readiness endpoints and Prometheus metrics.
---

## Health

- `GET /health` (liveness)
  - Always returns `200` with `{"status":"ok"}` if the server is running.
- `GET /ready` (readiness)
  - Returns `200` with `{"status":"ready"}` when the DB can be reached.
  - Returns `503` with `{"status":"unavailable"}` when the DB is unavailable.

Example:

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready
```

## Metrics

- `GET /metrics` (Prometheus exposition format)

By default, `/metrics` is public. To require Bearer auth:

```bash
export METRICS_AUTH_ENABLED=true
```

When auth is enabled, `/metrics` uses the same Bearer tokens as the API (`AUTH_TOKENS` / legacy `AUTH_TOKEN`):

```bash
TOKEN=token-current
curl -H "Authorization: Bearer ${TOKEN}" http://localhost:8080/metrics
```

Prometheus scraping notes:

- If you enable auth, prefer supplying the token via `bearer_token_file` (or similar) rather than embedding it directly in config.
- Alternatively, leave `METRICS_AUTH_ENABLED=false` and restrict `/metrics` at the network/reverse-proxy layer.

## Request IDs

mjr.wtf propagates `X-Request-ID`:

- If the request includes `X-Request-ID`, it is reused.
- Otherwise a UUID is generated.

The value is echoed back on responses via the `X-Request-ID` header and included in request logs as `request_id`.

## Logging

Logging is structured via `zerolog`.

- `LOG_LEVEL` (default: `info`)
- `LOG_FORMAT` (`json` or `pretty`, default: `json`)
- `LOG_STACK_TRACES` (default: `true`; include stack traces in panic recovery logs)

Each HTTP request is logged with fields like `request_id`, `method`, `path`, `status`, `size`, and `duration`.

