#!/bin/bash
set -e

# E2E test for build and installation
# Tests that the project builds correctly for all platforms

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEMP_DIR=$(mktemp -d)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    rm -rf "$TEMP_DIR"
}

trap cleanup EXIT

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_build_test() {
    local name="$1"
    local goos="$2"
    local goarch="$3"
    local output="$4"

    TESTS_RUN=$((TESTS_RUN + 1))
    log_info "Building: $name ($goos/$goarch)"

    cd "$PROJECT_ROOT"
    if GOOS="$goos" GOARCH="$goarch" go build -ldflags="-s -w -X main.Version=e2e-test" -o "$output" ./cmd/cando 2>&1 | tee "$TEMP_DIR/build.log"; then
        if [ -f "$output" ]; then
            local size=$(du -h "$output" | cut -f1)
            log_info "✓ Build passed: $name (size: $size)"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            log_error "✗ Build failed: $name (binary not created)"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        log_error "✗ Build failed: $name"
        cat "$TEMP_DIR/build.log"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

run_version_test() {
    local binary="$1"
    local name="$2"

    TESTS_RUN=$((TESTS_RUN + 1))
    log_info "Testing --version flag: $name"

    if output=$("$binary" --version 2>&1); then
        if echo "$output" | grep -q "Cando version"; then
            log_info "✓ Version test passed: $name"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            log_error "✗ Version test failed: $name (unexpected output)"
            echo "Output: $output"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        log_error "✗ Version test failed: $name (execution failed)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo "========================================"
echo "Cando Build & Installation E2E Tests"
echo "========================================"
echo ""

# Test current platform build
log_info "Testing current platform build..."
run_build_test "Current platform" "" "" "$TEMP_DIR/cando-current"

if [ -f "$TEMP_DIR/cando-current" ]; then
    chmod +x "$TEMP_DIR/cando-current"
    run_version_test "$TEMP_DIR/cando-current" "Current platform"
fi

# Test cross-compilation for major platforms
log_info "Testing cross-compilation..."

run_build_test "Linux amd64" "linux" "amd64" "$TEMP_DIR/cando-linux-amd64"
run_build_test "Linux arm64" "linux" "arm64" "$TEMP_DIR/cando-linux-arm64"
run_build_test "macOS amd64" "darwin" "amd64" "$TEMP_DIR/cando-darwin-amd64"
run_build_test "macOS arm64" "darwin" "arm64" "$TEMP_DIR/cando-darwin-arm64"
run_build_test "Windows amd64" "windows" "amd64" "$TEMP_DIR/cando-windows-amd64.exe"
run_build_test "Windows arm64" "windows" "arm64" "$TEMP_DIR/cando-windows-arm64.exe"

# Test linters
TESTS_RUN=$((TESTS_RUN + 1))
log_info "Running go vet..."
cd "$PROJECT_ROOT"
if go vet ./... 2>&1 | tee "$TEMP_DIR/vet.log"; then
    log_info "✓ go vet passed"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log_error "✗ go vet failed"
    cat "$TEMP_DIR/vet.log"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test formatting
TESTS_RUN=$((TESTS_RUN + 1))
log_info "Checking go fmt..."
cd "$PROJECT_ROOT"
if [ -z "$(go fmt ./... 2>&1)" ]; then
    log_info "✓ go fmt passed (no changes needed)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log_error "✗ go fmt found formatting issues"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Summary
echo ""
echo "========================================"
echo "Build Test Summary"
echo "========================================"
echo "Total tests:  $TESTS_RUN"
echo -e "${GREEN}Passed:       $TESTS_PASSED${NC}"
echo -e "${RED}Failed:       $TESTS_FAILED${NC}"
echo "========================================"

if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi

log_info "All build tests passed!"
exit 0
