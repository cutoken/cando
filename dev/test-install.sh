#!/usr/bin/env bash
# Test the beta installer locally without pushing anywhere
set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
TEMP_DIR=$(mktemp -d)
TEST_INSTALL_DIR="$TEMP_DIR/test-install"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}!${NC} $1"; }
test() { echo -e "${BLUE}▶${NC} $1"; }

cleanup() {
    if [[ -n "${SERVER_PID:-}" ]]; then
        kill $SERVER_PID 2>/dev/null || true
    fi
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

echo "Beta Installer Local Test"
echo "========================="
echo ""

# Build if needed
if [[ ! -d "$DIST_DIR" ]] || [[ -z "$(ls -A $DIST_DIR 2>/dev/null)" ]]; then
    test "Building binaries..."
    cd "$ROOT_DIR"
    make all
fi

# Start local server
test "Starting local file server..."
cd "$DIST_DIR"
python3 -m http.server 8765 --bind 127.0.0.1 >/dev/null 2>&1 &
SERVER_PID=$!
sleep 2

# Test installer with local server
test "Testing installer with local server..."
cd "$TEMP_DIR"
CANDO_BASE_URL="http://127.0.0.1:8765" \
CANDO_INSTALL_DIR="$TEST_INSTALL_DIR" \
    bash "$ROOT_DIR/dev/install-beta.sh"

# Verify installation
test "Verifying installation..."
if [[ -x "$TEST_INSTALL_DIR/cando-beta" ]]; then
    info "Binary installed successfully"
    
    # Test version flag
    if version=$("$TEST_INSTALL_DIR/cando-beta" --version 2>/dev/null); then
        info "Version check passed: $version"
    else
        warn "Version check failed"
    fi
else
    warn "Binary not found at $TEST_INSTALL_DIR/cando-beta"
    exit 1
fi

# Test direct file:// URL method
test "Testing file:// URL method..."
kill $SERVER_PID 2>/dev/null || true
unset SERVER_PID

TEST_INSTALL_DIR2="$TEMP_DIR/test-install-2"
CANDO_BASE_URL="file://$DIST_DIR" \
CANDO_INSTALL_DIR="$TEST_INSTALL_DIR2" \
    bash "$ROOT_DIR/dev/install-beta.sh"

if [[ -x "$TEST_INSTALL_DIR2/cando-beta" ]]; then
    info "file:// URL installation works"
else
    warn "file:// URL installation failed"
fi

echo ""
echo "================================="
info "Local installation test complete"
echo ""
echo "Tested installations:"
echo "  • HTTP server: $TEST_INSTALL_DIR/cando-beta"
echo "  • File URL:    $TEST_INSTALL_DIR2/cando-beta"
echo ""
echo "To test with real beta testers:"
echo "  1. Push to beta branch"
echo "  2. Share: curl -fsSL https://raw.githubusercontent.com/cutoken/cando/beta/dev/install-beta.sh | bash"