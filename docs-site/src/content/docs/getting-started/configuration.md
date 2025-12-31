---
title: Configuration
description: Environment variables and auth configuration.
---

mjr.wtf is configured via environment variables.

## Authentication tokens

At least one token must be set for the server to start:

- Preferred: `AUTH_TOKENS` (comma-separated, supports rotation)
- Legacy: `AUTH_TOKEN` (single token; used only if `AUTH_TOKENS` is unset)

Example:

```bash
export AUTH_TOKENS=token-current,token-next
# or:
export AUTH_TOKEN=token-current
```

## Common variables

- `DATABASE_URL` (required)
  - SQLite-only: set this to a **file path** (e.g. `./database.db`).
  - URL-form values (anything containing `://`) are rejected to avoid SQLite creating a local file literally named after the URL.
- `SERVER_PORT` (default: 8080)
- `BASE_URL` (default: http://localhost:8080)
- `ALLOWED_ORIGINS` (default: *)

## URL status checker (optional)

A background job can periodically check whether destination URLs are returning HTTP 404/410 and store the last seen status.

- Redirects **do not** perform a live/on-request destination check.
- If a short URL is marked as gone, `GET /{shortCode}` returns an HTML interstitial with the stored status code (404 or 410) and (when available) a Wayback Machine link.

Environment variables (Go duration format like `5m`, `6h`, `168h`):

- `URL_STATUS_CHECKER_ENABLED` (default: `false`)
- `URL_STATUS_CHECKER_POLL_INTERVAL` (default: `5m`)
- `URL_STATUS_CHECKER_ALIVE_RECHECK_INTERVAL` (default: `6h`)
- `URL_STATUS_CHECKER_GONE_RECHECK_INTERVAL` (default: `24h`)
- `URL_STATUS_CHECKER_BATCH_SIZE` (default: `100`)
- `URL_STATUS_CHECKER_CONCURRENCY` (default: `5`)
- `URL_STATUS_CHECKER_ARCHIVE_LOOKUP_ENABLED` (default: `true`)
- `URL_STATUS_CHECKER_ARCHIVE_RECHECK_INTERVAL` (default: `168h`)

**Privacy note:** When enabled, the checker makes outbound HTTP requests to destination URLs and (if archive lookup is enabled) to `https://archive.org/wayback/available`.

Example:

```bash
export DATABASE_URL=./database.db
```

For the full list, see the repository README:

- https://github.com/matt-riley/mjrwtf/blob/main/README.md
