#!/usr/bin/env bash
# Code generation script for use with go generate
set -e

# Get the directory where this script lives (repository root)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd "$SCRIPT_DIR"

# Use a lock directory to prevent duplicate runs during the same go generate invocation.
# mkdir is atomic, so only one process will successfully create the directory.
LOCK_DIR="/tmp/mjrwtf-generate.lock"

# Attempt to acquire the lock. If this fails, another process is already running generation.
if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    exit 0
fi

# Clean up lock directory on exit
trap "rmdir \"$LOCK_DIR\" >/dev/null 2>&1 || true" EXIT

echo "Running code generation..."

# Run sqlc generate
if command -v sqlc >/dev/null 2>&1; then
    sqlc generate
else
    echo "Error: sqlc not installed. Install with:"
    echo "  go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0"
    exit 1
fi

# Run templ generate
if command -v templ >/dev/null 2>&1; then
    templ generate
else
    echo "Error: templ not installed. Install with:"
    echo "  go install github.com/a-h/templ/cmd/templ@latest"
    exit 1
fi

echo "Code generation complete!"
