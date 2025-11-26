#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEMP_DIR="$(mktemp -d)"
WORKSPACE="$TEMP_DIR/workspace"
BINARY="$TEMP_DIR/cando"

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

echo "[mock-cli] Building binary..."
cd "$PROJECT_ROOT"
go build -ldflags="-s -w -X main.Version=mock-test" -o "$BINARY" ./cmd/cando

mkdir -p "$WORKSPACE"

echo "[mock-cli] Running cando -p with mock client..."
OUTPUT=$(CANDO_MOCK_LLM=1 "$BINARY" --sandbox "$WORKSPACE" -p "ping" || true)

echo "$OUTPUT"

if [[ "$OUTPUT" != *"MOCK RESPONSE: ping"* ]]; then
    echo "[mock-cli] Expected mock response not found." >&2
    exit 1
fi

echo "[mock-cli] Mock CLI test passed."
