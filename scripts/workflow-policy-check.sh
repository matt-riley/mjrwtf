#!/usr/bin/env bash
set -euo pipefail

ROOT="."

while [[ $# -gt 0 ]]; do
  case "$1" in
    --root)
      ROOT="$2"
      shift 2
      ;;
    *)
      echo "Unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

WORKFLOWS_DIR="$ROOT/.github/workflows"
GORELEASER_FILE="$ROOT/.goreleaser.yaml"
GORELEASER_WORKFLOW="$WORKFLOWS_DIR/goreleaser.yml"

check_sha_pinning() {
  if grep -RInE '^\s*(-\s*)?uses:\s+[^#[:space:]]+@(main|master|HEAD)\b' "$WORKFLOWS_DIR"; then
    echo "Error: workflows must not pin actions to main/master/HEAD" >&2
    exit 1
  fi

  if grep -RInE '^\s*(-\s*)?uses:\s+[^#[:space:]]+@v[0-9]+(\.[0-9]+)*\b' "$WORKFLOWS_DIR"; then
    echo "Error: workflows must pin actions by commit SHA (not @vX tags)" >&2
    exit 1
  fi

  while IFS= read -r line; do
    content="$(echo "$line" | cut -d: -f3-)"
    ref="$(echo "$content" | sed -E 's/^\s*uses:\s+([^#[:space:]]+).*/\1/')"
    case "$ref" in
      ./*) continue ;; # local action
    esac
    if ! echo "$ref" | grep -Eq '@[0-9a-f]{40}$'; then
      echo "Error: unpinned action reference: $content" >&2
      exit 1
    fi
  done < <(grep -RInE '^\s*(-\s*)?uses:\s+' "$WORKFLOWS_DIR")
}

check_release_codegen() {
  if ! grep -nE '^\s*-\s+sqlc generate\b' "$GORELEASER_FILE"; then
    echo "Error: .goreleaser.yaml must run 'sqlc generate' in release builds" >&2
    exit 1
  fi

  if ! grep -nE '^\s*-\s+templ generate\b' "$GORELEASER_FILE"; then
    echo "Error: .goreleaser.yaml must run 'templ generate' in release builds" >&2
    exit 1
  fi

  if ! grep -nE 'name:\s+Install sqlc\b' "$GORELEASER_WORKFLOW"; then
    echo "Error: .github/workflows/goreleaser.yml must contain a step named 'Install sqlc'" >&2
    exit 1
  fi

  if ! grep -nE 'name:\s+Install templ\b' "$GORELEASER_WORKFLOW"; then
    echo "Error: .github/workflows/goreleaser.yml must contain a step named 'Install templ'" >&2
    exit 1
  fi
}

check_sha_pinning
check_release_codegen
