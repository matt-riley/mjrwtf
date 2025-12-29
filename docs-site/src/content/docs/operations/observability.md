---
title: Observability
description: Health/readiness endpoints and Prometheus metrics.
---

## Health

- `GET /health` (liveness)
- `GET /ready` (readiness; checks DB connectivity)

## Metrics

- `GET /metrics` (Prometheus metrics)

By default, `/metrics` is public. To require Bearer auth:

```bash
export METRICS_AUTH_ENABLED=true
```
