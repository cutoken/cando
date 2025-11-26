#!/bin/bash
set -e

# E2E tests for Cando providers
# Tests both ZAI and OpenRouter with real API calls
# Requires ZAI_API_KEY and/or OPENROUTER_API_KEY environment variables

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BINARY="$PROJECT_ROOT/cando"
TEMP_DIR=$(mktemp -d)
TEMP_CREDS="$TEMP_DIR/credentials.yaml"
TEMP_CONFIG="$TEMP_DIR/config.yaml"
TEMP_WORKSPACE="$TEMP_DIR/workspace"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

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

log_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
}

run_test() {
    local test_name="$1"
    local provider="$2"
    local prompt="$3"

    TESTS_RUN=$((TESTS_RUN + 1))
    echo ""
    log_info "Running: $test_name"

    local output
    if output=$(CANDO_CREDENTIALS_PATH="$TEMP_CREDS" CANDO_CONFIG_PATH="$TEMP_CONFIG" "$BINARY" --sandbox "$TEMP_WORKSPACE" -p "$prompt" 2>&1); then
        # Check if we got a non-empty response
        if [ -n "$output" ] && echo "$output" | grep -q -v "^$"; then
            log_info "✓ Test passed: $test_name"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            echo "Response preview:"
            echo "$output" | head -5
            return 0
        else
            log_error "✗ Test failed: $test_name (empty response)"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        log_error "✗ Test failed: $test_name"
        echo "Error output:"
        echo "$output"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Check if binary exists, build if needed
if [ ! -f "$BINARY" ]; then
    log_info "Binary not found, building..."
    cd "$PROJECT_ROOT"
    go build -ldflags="-s -w -X main.Version=e2e-test" -o cando ./cmd/cando
    log_info "Build complete"
fi

# Verify binary
if [ ! -x "$BINARY" ]; then
    log_error "Binary not executable: $BINARY"
    exit 1
fi

log_info "Using binary: $BINARY"
log_info "Temp dir: $TEMP_DIR"

# Create workspace
mkdir -p "$TEMP_WORKSPACE"

# Test ZAI provider
if [ -n "$ZAI_API_KEY" ]; then
    log_info "Setting up ZAI provider test..."

    cat > "$TEMP_CREDS" <<EOF
default_provider: zai
providers:
  zai:
    api_key: ${ZAI_API_KEY}
EOF

    cat > "$TEMP_CONFIG" <<EOF
model: glm-4.6
provider_models:
  zai: glm-4.6
temperature: 0.7
thinking_enabled: true
context_profile: memory
context_message_threshold: 5000
context_conversation_threshold: 200000
context_protect_recent: 5
compaction_summary_prompt: "Summarize the following text in 40 words or fewer."
summary_model: glm-4.5-air
workspace_root: .
system_prompt: "You are Cando, a concise and helpful assistant."
EOF

    run_test "ZAI: Simple math" "zai" "What is 2+2? Answer with just the number."
    run_test "ZAI: Simple question" "zai" "What is the capital of France? Answer in one word."
    run_test "ZAI: Code generation" "zai" "Write a one-line Python function to reverse a string."
else
    log_skip "ZAI tests (ZAI_API_KEY not set)"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 3))
fi

# Test OpenRouter provider
if [ -n "$OPENROUTER_API_KEY" ]; then
    log_info "Setting up OpenRouter provider test..."

    cat > "$TEMP_CREDS" <<EOF
default_provider: openrouter
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
EOF

    cat > "$TEMP_CONFIG" <<EOF
model: qwen/qwen-2.5-72b-instruct
provider_models:
  openrouter: qwen/qwen-2.5-72b-instruct
temperature: 0.7
thinking_enabled: true
context_profile: memory
context_message_threshold: 5000
context_conversation_threshold: 200000
context_protect_recent: 5
compaction_summary_prompt: "Summarize the following text in 40 words or fewer."
summary_model: qwen/qwen-2.5-7b-instruct
workspace_root: .
system_prompt: "You are Cando, a concise and helpful assistant."
EOF

    run_test "OpenRouter: Simple math" "openrouter" "What is 3+3? Answer with just the number."
    run_test "OpenRouter: Simple question" "openrouter" "What is the capital of Italy? Answer in one word."
    run_test "OpenRouter: Code generation" "openrouter" "Write a one-line JavaScript function to check if a string is empty."
else
    log_skip "OpenRouter tests (OPENROUTER_API_KEY not set)"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 3))
fi

# Test both providers together (if both API keys available)
if [ -n "$ZAI_API_KEY" ] && [ -n "$OPENROUTER_API_KEY" ]; then
    log_info "Setting up multi-provider test..."

    cat > "$TEMP_CREDS" <<EOF
default_provider: zai
providers:
  zai:
    api_key: ${ZAI_API_KEY}
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
EOF

    cat > "$TEMP_CONFIG" <<EOF
model: glm-4.6
provider_models:
  zai: glm-4.6
  openrouter: qwen/qwen-2.5-72b-instruct
temperature: 0.7
thinking_enabled: true
context_profile: memory
context_message_threshold: 5000
context_conversation_threshold: 200000
context_protect_recent: 5
compaction_summary_prompt: "Summarize the following text in 40 words or fewer."
summary_model: glm-4.5-air
workspace_root: .
system_prompt: "You are Cando, a concise and helpful assistant."
EOF

    run_test "Multi-provider: Default (ZAI)" "zai" "Say 'hello' in one word."
else
    log_skip "Multi-provider test (requires both API keys)"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
fi

# Summary
echo ""
echo "========================================"
echo "E2E Test Summary"
echo "========================================"
echo "Total tests:  $TESTS_RUN"
echo -e "${GREEN}Passed:       $TESTS_PASSED${NC}"
echo -e "${RED}Failed:       $TESTS_FAILED${NC}"
echo -e "${YELLOW}Skipped:      $TESTS_SKIPPED${NC}"
echo "========================================"

if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi

if [ $TESTS_RUN -eq 0 ]; then
    log_error "No tests run! Set ZAI_API_KEY or OPENROUTER_API_KEY environment variables."
    exit 1
fi

log_info "All tests passed!"
exit 0
