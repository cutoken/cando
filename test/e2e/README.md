# Cando E2E Tests

End-to-end tests covering builds, CLI functionality, and LLM provider integration.

## Test Suite

### 1. Build Tests (`test_build.sh`)

Tests compilation and cross-platform builds:
- Current platform build
- Cross-compilation for Linux (amd64, arm64)
- Cross-compilation for macOS (amd64, arm64)
- Cross-compilation for Windows (amd64)
- `--version` flag functionality
- Code linting (`go vet`, `go fmt`)

**Run:**
```bash
./test_build.sh
```

**No prerequisites** - runs entirely offline.

### 2. Provider Tests (`test_providers.sh`)

Tests real API interactions with LLM providers:
- ZAI provider (if `ZAI_API_KEY` set)
- OpenRouter provider (if `OPENROUTER_API_KEY` set)
- Multi-provider configuration
- One-shot prompt mode (`-p` flag)

**Run:**
```bash
# Test ZAI only
ZAI_API_KEY="your-key" ./test_providers.sh

# Test OpenRouter only
OPENROUTER_API_KEY="your-key" ./test_providers.sh

# Test both
ZAI_API_KEY="your-zai-key" OPENROUTER_API_KEY="your-or-key" ./test_providers.sh
```

**Prerequisites:**
- At least one valid API key (ZAI or OpenRouter)
- Internet connection
- API quota/credits available

### 3. Run All Tests (`run_all.sh`)

Master script that runs all test suites:

```bash
# Run all tests (provider tests only if API keys available)
./run_all.sh

# With API keys for full coverage
ZAI_API_KEY="..." OPENROUTER_API_KEY="..." ./run_all.sh
```

## CI Integration

### GitHub Actions Example

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run build tests
        run: ./test/e2e/test_build.sh

      - name: Run mock CLI tests
        run: ./test/e2e/test_cli_mock.sh
```

### Local Development

Add to your shell profile for quick testing:

```bash
# ~/.bashrc or ~/.zshrc
export ZAI_API_KEY="your-development-key"
export OPENROUTER_API_KEY="your-development-key"

alias cando-test="cd ~/code/cando && ./test/e2e/run_all.sh"
```

## Test Output

Each test script provides:
- Colored output (green=pass, red=fail, yellow=skip)
- Test counters (run/passed/failed/skipped)
- Exit code 0 on success, 1 on failure
- Automatic cleanup of temporary files

### Example Output

```
========================================
Cando E2E Test Suite
========================================

[1/2] Running build tests...
[INFO] Building: Current platform (linux/amd64)
[INFO] ✓ Build passed: Current platform (size: 16M)
[INFO] Testing --version flag: Current platform
[INFO] ✓ Version test passed: Current platform
...
✓ Build tests passed

[2/2] Running provider tests...
[INFO] Setting up ZAI provider test...
[INFO] Running: ZAI: Simple math
[INFO] ✓ Test passed: ZAI: Simple math
...
✓ Provider tests passed

========================================
All E2E tests passed!
========================================
```

## Debugging Failed Tests

### Build test failures

Check compilation errors:
```bash
cd ../../  # Go to project root
go build -v ./cmd/cando
```

### Provider test failures

Run with verbose output:
```bash
set -x  # Enable bash debugging
ZAI_API_KEY="..." ./test_providers.sh
```

Check credentials manually:
```bash
# Create test credentials
export CANDO_CREDENTIALS_PATH=/tmp/test-creds.yaml
cat > /tmp/test-creds.yaml <<EOF
default_provider: zai
providers:
  zai:
    api_key: ${ZAI_API_KEY}
EOF

# Test directly
./cando -p "test prompt"
```

## Notes

- Tests use real API calls (no mocking) per project guidelines
- Provider tests are automatically skipped if API keys unavailable
- All tests use temporary directories and clean up after themselves
- Tests can be run in parallel (each uses isolated temp space)
