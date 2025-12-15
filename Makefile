.PHONY: help test lint fmt vet build build-server build-migrate clean migrate-up migrate-down migrate-status migrate-create migrate-reset templ-generate templ-watch docker-build docker-run

# Default target
help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  lint              - Run golangci-lint"
	@echo "  fmt               - Format code with gofmt"
	@echo "  vet               - Run go vet"
	@echo "  build             - Build all binaries"
	@echo "  build-server      - Build the HTTP server binary"
	@echo "  build-migrate     - Build the migration tool"
	@echo "  clean             - Clean build artifacts and coverage files"
	@echo "  templ-generate    - Generate Go code from Templ templates"
	@echo "  templ-watch       - Watch and auto-regenerate Templ templates"
	@echo ""
	@echo "Migration targets:"
	@echo "  migrate-up        - Apply all pending migrations"
	@echo "  migrate-down      - Rollback the most recent migration"
	@echo "  migrate-status    - Show migration status"
	@echo "  migrate-create NAME=<name> - Create a new migration"
	@echo "  migrate-reset     - Rollback all migrations"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build      - Build Docker image"
	@echo "  docker-run        - Run Docker container (requires .env file)"

# Run all tests
test:
	go test -v ./...

# Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v2.5.0"; \
		exit 1; \
	fi

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Build all binaries
build: templ-generate build-server build-migrate

# Build server binary
build-server: templ-generate
	go build -o bin/server ./cmd/server

# Build migrate tool
build-migrate:
	go build -o bin/migrate ./cmd/migrate

# Clean build artifacts and coverage files
clean:
	rm -rf bin/

# Migration commands
migrate-up: build-migrate
	./bin/migrate up

migrate-down: build-migrate
	./bin/migrate down

migrate-status: build-migrate
	./bin/migrate status

migrate-create: build-migrate
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=my_migration"; \
		exit 1; \
	fi
	./bin/migrate create $(NAME)

migrate-reset: build-migrate
	./bin/migrate reset

# Run all checks (templ-generate, fmt, vet, lint, test)
check: templ-generate fmt vet lint test
	@echo "All checks passed!"

# Generate Go code from Templ templates
templ-generate:
	@if command -v templ >/dev/null 2>&1; then \
		templ generate; \
	else \
		echo "templ not installed. Install with:"; \
		echo "  go install github.com/a-h/templ/cmd/templ@latest"; \
		exit 1; \
	fi

# Watch and auto-regenerate Templ templates
templ-watch:
	@if command -v templ >/dev/null 2>&1; then \
		templ generate --watch; \
	else \
		echo "templ not installed. Install with:"; \
		echo "  go install github.com/a-h/templ/cmd/templ@latest"; \
		exit 1; \
	fi

# Docker targets
docker-build:
	docker build -t mjrwtf:latest .

docker-run:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Copy .env.example to .env and configure it."; \
		exit 1; \
	fi
	docker run -d \
		--name mjrwtf \
		-p 8080:8080 \
		--env-file .env \
		mjrwtf:latest
