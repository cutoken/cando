#!/bin/bash

# Debug environment for running cando with air
# This script is never committed (see .gitignore)

set -e

# Environment variables for debug mode
export DEV_MODE=true
export CANDO_PORT=4000
export CANDO_CONFIG_DIR="/tmp/cando-test"

# Ensure the test config directory exists
mkdir -p "$CANDO_CONFIG_DIR"

echo "🔧 Starting cando in debug mode:"
echo "   DEV_MODE: $DEV_MODE"
echo "   CANDO_PORT: $CANDO_PORT"
echo "   CANDO_CONFIG_DIR: $CANDO_CONFIG_DIR"
echo ""

# Check if air is installed
if ! command -v air &> /dev/null; then
    echo "❌ 'air' is not installed. Install it with:"
    echo "   go install github.com/air-verse/air@latest"
    exit 1
fi

# Run air with the debug environment
air
