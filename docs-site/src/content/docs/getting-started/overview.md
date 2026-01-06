---
title: Overview
description: What mjr.wtf is and how to get started.
---

mjr.wtf is a small URL shortener written in Go, using SQLite for persistence.

## Quick Look

![mjr.wtf TUI - Interactive terminal interface for URL management](/images/tui/demo.gif)

*Manage your short URLs right from the terminal with an intuitive TUI.*

## Quick Start (Docker Compose)

```bash
git clone https://github.com/matt-riley/mjrwtf.git
cd mjrwtf

cp .env.example .env
# Edit .env and set AUTH_TOKENS (preferred) or AUTH_TOKEN (legacy)
mkdir -p data
make docker-compose-up
curl http://localhost:8080/health
```

Docker runs migrations automatically on startup.

## How to use these docs

- Start with **Local development** to run the server directly.
- Use **Docker** / **Docker Compose** for container-based runs.
- Use **API** for the OpenAPI spec and endpoint details.

## Source repository

- https://github.com/matt-riley/mjrwtf
