#!/bin/bash
# Code generation script for use with go generate
set -e

# Get the directory where this script lives (repository root)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd "$SCRIPT_DIR"

# Create a marker file to prevent duplicate runs during the same go generate invocation
MARKER_FILE="/tmp/mjrwtf-generate-$$.marker"

# Check if we already ran in this invocation
if [ -f "$MARKER_FILE" ]; then
    exit 0
fi

# Create marker file
touch "$MARKER_FILE"

# Clean up marker file on exit
trap "rm -f $MARKER_FILE" EXIT

echo "Running code generation..."

# Run sqlc generate
if command -v sqlc >/dev/null 2>&1; then
    sqlc generate
else
    echo "Error: sqlc not installed. Install with:"
    echo "  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"
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
