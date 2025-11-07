.PHONY: help test test-short test-coverage test-coverage-html coverage-check lint fmt vet build clean

# Default target
help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  lint              - Run golangci-lint"
	@echo "  fmt               - Format code with gofmt"
	@echo "  vet               - Run go vet"
	@echo "  build             - Build the binary"
	@echo "  clean             - Clean build artifacts and coverage files"

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

# Build binary
build:
	go build -o bin/mjrwtf ./cmd/mjrwtf

# Clean build artifacts and coverage files
clean:
	rm -rf bin/

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"
