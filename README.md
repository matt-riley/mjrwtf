# mjr.wtf

A simple URL shortener written in Go with SQLite storage, built for speed and ease of deployment.

## ✨ Key Features

- **Fast & Lightweight** - Minimal dependencies, SQLite database, sub-millisecond redirects
- **RESTful API** - OpenAPI 3.0 documented HTTP API with Bearer token authentication
- **Web Dashboard** - Modern UI for managing URLs with session-based authentication
- **Terminal UI** - Interactive TUI for command-line enthusiasts
- **Click Analytics** - Track clicks with timestamps, user agents, referrers, and optional GeoIP
- **URL Health Checks** - Optional periodic checks for dead links with Wayback Machine integration
- **Rate Limiting** - Per-IP rate limits for both redirects and API endpoints
- **Production Ready** - Prometheus metrics, structured logging, health checks, CORS, security headers
- **Easy Deployment** - Docker images (multi-arch), Docker Compose, or standalone binaries

## 🚀 Quick Start

```bash
# 1. Copy and configure environment variables
cp .env.example .env
# Edit .env to set AUTH_TOKEN (required)

# 2. Prepare a persistent data directory and run migrations
mkdir -p data
export DATABASE_URL=./data/database.db
make migrate-up

# 3. Start the server
make docker-compose-up

# 4. Verify it's running
curl http://localhost:8080/health
```

Visit `http://localhost:8080` to access the web dashboard.

## 📚 Documentation

For detailed documentation, visit **[docs.mjr.wtf](https://docs.mjr.wtf)**:

- **[Getting Started](https://docs.mjr.wtf/getting-started/overview/)** - Installation, deployment, and initial setup
- **[Configuration](https://docs.mjr.wtf/getting-started/configuration/)** - Environment variables and configuration options
- **[Authentication](https://docs.mjr.wtf/security/)** - Bearer tokens, sessions, and security settings
- **[API Reference](https://docs.mjr.wtf/api/)** - OpenAPI 3.0 specification ([SwaggerUI](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/matt-riley/mjrwtf/main/openapi.yaml) | [ReDoc](https://redocly.github.io/redoc/?url=https://raw.githubusercontent.com/matt-riley/mjrwtf/main/openapi.yaml))
- **[Database Migrations](https://docs.mjr.wtf/operations/migrations/)** - Managing schema changes
- **[Testing](https://docs.mjr.wtf/operations/integration-testing/)** - Running unit and integration tests
- **[Docker Guide](https://docs.mjr.wtf/operations/docker/)** - Container deployment and production setup
- **[Local Development](https://docs.mjr.wtf/getting-started/local-development/)** - Code generation, building, and contributing

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
