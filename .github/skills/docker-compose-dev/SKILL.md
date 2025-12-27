---
name: docker-compose-dev
description: Run mjr.wtf locally using Docker Compose (PostgreSQL + app), including migrations from the host, logs, and teardown. Use when reproducing prod-like behavior or developing against Postgres.
license: MIT
compatibility: Requires docker, docker compose, bash, git, and make.
metadata:
  repo: mjrwtf
  runner: github-copilot-cli
  version: 1.1
allowed-tools: Bash(git:*) Bash(make:*) Bash(docker:*) Bash(curl:*) Read
---

## Tooling assumptions

- Use a terminal runner with bash and git available.
- Prefer `make` targets when available; fall back to direct CLI commands when needed.

## Quick start (PostgreSQL via compose)

1) Create env file (optional but recommended):

```bash
cp .env.example .env
```

2) Start services:

```bash
make docker-compose-up
```

3) Run migrations from the host (required):

```bash
make build-migrate
export DATABASE_URL='postgresql://mjrwtf:INSECURE_CHANGE_ME@localhost:5432/mjrwtf'
./bin/migrate up
```

4) Verify health:

```bash
curl http://localhost:8080/health
```

## Useful ops

- Logs:

```bash
make docker-compose-logs
```

- Status:

```bash
make docker-compose-ps
```

- Stop + remove containers:

```bash
make docker-compose-down
```

## Common pitfalls

- Migrations are not run automatically; do them before hitting API endpoints.
- If you changed Postgres credentials in `.env`, update `DATABASE_URL` accordingly.
