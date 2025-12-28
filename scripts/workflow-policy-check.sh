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

if [[ ! -d "$WORKFLOWS_DIR" ]]; then
  echo "Error: workflows dir not found: $WORKFLOWS_DIR" >&2
  exit 1
fi

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

    ref="$(echo "$content" | sed -E 's/^\s*-?\s*uses:\s+//; s/\s+#.*$//; s/[[:space:]]+$//')"
    ref="${ref%\"}"
    ref="${ref#\"}"
    ref="${ref%\'}"
    ref="${ref#\'}"

    case "$ref" in
      ./*) continue ;; # local action
    esac

    if ! echo "$ref" | grep -Eq '@[0-9a-fA-F]{40}$'; then
      echo "Error: unpinned action reference: $content" >&2
      exit 1
    fi
  done < <(grep -RInE '^\s*(-\s*)?uses:\s+' "$WORKFLOWS_DIR" || true)
}

check_release_codegen() {
  if [[ ! -f "$GORELEASER_FILE" ]]; then
    echo "Error: .goreleaser.yaml not found: $GORELEASER_FILE" >&2
    exit 1
  fi

  if [[ ! -f "$GORELEASER_WORKFLOW" ]]; then
    echo "Error: workflow not found: $GORELEASER_WORKFLOW" >&2
    exit 1
  fi

  if ! grep -qE '^\s*-\s+sqlc generate\b' "$GORELEASER_FILE"; then
    echo "Error: .goreleaser.yaml must run 'sqlc generate' in release builds" >&2
    exit 1
  fi

  if ! grep -qE '^\s*-\s+templ generate\b' "$GORELEASER_FILE"; then
    echo "Error: .goreleaser.yaml must run 'templ generate' in release builds" >&2
    exit 1
  fi

  if ! grep -qE 'name:\s+Install sqlc\b' "$GORELEASER_WORKFLOW"; then
    echo "Error: .github/workflows/goreleaser.yml must contain a step named 'Install sqlc'" >&2
    exit 1
  fi

  if ! grep -qE 'name:\s+Install templ\b' "$GORELEASER_WORKFLOW"; then
    echo "Error: .github/workflows/goreleaser.yml must contain a step named 'Install templ'" >&2
    exit 1
  fi
}

check_sha_pinning
check_release_codegen
