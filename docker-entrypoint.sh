#!/bin/sh
set -eu

# Ensure the SQLite schema exists before starting the server.
# Uses embedded goose migrations; requires DATABASE_URL to be set.
./migrate up

exec "$@"
