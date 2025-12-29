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
- `SERVER_PORT` (default: 8080)
- `BASE_URL` (default: http://localhost:8080)
- `ALLOWED_ORIGINS` (default: *)

For the full list, see the repository README:

- https://github.com/matt-riley/mjrwtf/blob/main/README.md
