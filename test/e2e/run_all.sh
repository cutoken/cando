#!/bin/bash
set -e

# Master E2E test runner - runs all tests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================"
echo "Cando E2E Test Suite"
echo -e "========================================${NC}"
echo ""

# Track overall status
OVERALL_PASSED=true

# Run build tests
echo -e "${BLUE}[1/3] Running build tests...${NC}"
if "$SCRIPT_DIR/test_build.sh"; then
    echo -e "${GREEN}✓ Build tests passed${NC}"
else
    echo -e "${RED}✗ Build tests failed${NC}"
    OVERALL_PASSED=false
fi

echo ""

# Run mock CLI test
echo -e "${BLUE}[2/3] Running mock CLI test...${NC}"
if "$SCRIPT_DIR/test_cli_mock.sh"; then
    echo -e "${GREEN}✓ Mock CLI test passed${NC}"
else
    echo -e "${RED}✗ Mock CLI test failed${NC}"
    OVERALL_PASSED=false
fi

echo ""

# Run provider tests (only if API keys are set)
if [ -n "$ZAI_API_KEY" ] || [ -n "$OPENROUTER_API_KEY" ]; then
    echo -e "${BLUE}[3/3] Running provider tests...${NC}"
    if "$SCRIPT_DIR/test_providers.sh"; then
        echo -e "${GREEN}✓ Provider tests passed${NC}"
    else
        echo -e "${RED}✗ Provider tests failed${NC}"
        OVERALL_PASSED=false
    fi
else
    echo -e "${BLUE}[3/3] Skipping provider tests (no API keys set)${NC}"
    echo "Set ZAI_API_KEY and/or OPENROUTER_API_KEY to run provider tests"
fi

echo ""
echo -e "${BLUE}========================================"
if [ "$OVERALL_PASSED" = true ]; then
    echo -e "${GREEN}All E2E tests passed!${NC}"
    echo -e "========================================${NC}"
    exit 0
else
    echo -e "${RED}Some E2E tests failed${NC}"
    echo -e "========================================${NC}"
    exit 1
fi
