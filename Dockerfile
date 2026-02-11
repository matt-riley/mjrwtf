# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies (gcc, musl-dev for CGO required by go-sqlite3)
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate templ templates
# Pin to specific version to ensure reproducible builds and reduce supply chain risk
RUN go install github.com/a-h/templ/cmd/templ@v0.3.960 && \
    templ generate

# Build the server binary
# CGO_ENABLED=1 is required for go-sqlite3
# TARGETOS and TARGETARCH are automatically set by buildx for multi-arch builds
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -installsuffix cgo -o server ./cmd/server && \
    CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -installsuffix cgo -o migrate ./cmd/migrate

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create a non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy the binaries from builder
COPY --from=builder /build/server .
COPY --from=builder /build/migrate .
COPY docker-entrypoint.sh .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app && chmod +x /app/docker-entrypoint.sh

# Switch to non-root user
USER appuser

# Expose the application port
EXPOSE 8080

# Health check
# Checks the /health endpoint every 30 seconds
# Starts checking after 5 seconds, times out after 3 seconds
# Retries 3 times before marking as unhealthy
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run migrations, then start the server
ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["./server"]
