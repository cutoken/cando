<p align="center">
  <img src="docs/images/cando_logo.png" alt="CanDo Logo" width="200">
</p>

# CanDo

The coding agent that actually gets shit done. True autonomous coding - it writes, tests, debugs, and ships real features while you focus on what matters. Supports Z.AI (GLM models) and OpenRouter (Claude, GPT-4, and 100+ models). Cross-platform: Linux, macOS, Windows.

## Install

### Automatic (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
```

- Detects your OS/arch, downloads the right binary, and drops it in `~/.local/bin` (or `%USERPROFILE%\.local\bin` on Windows Git Bash/WSL).
- No Apple/Windows developer account needed; the binary runs from the terminal just like other CLI tools.

### Manual downloads

Grab the latest release from [GitHub Releases](https://github.com/cutoken/cando/releases/latest).

**Linux / macOS**

```bash
# Example: Linux amd64
curl -L https://github.com/cutoken/cando/releases/latest/download/cando-linux-amd64-bin -o cando
chmod +x cando
sudo mv cando /usr/local/bin/

# Example: macOS arm64 (Apple Silicon)
curl -L https://github.com/cutoken/cando/releases/latest/download/cando-darwin-arm64-bin -o cando
chmod +x cando
sudo mv cando /usr/local/bin/
```

**Windows (PowerShell) - Recommended**

```powershell
irm https://raw.githubusercontent.com/cutoken/cando/main/install.ps1 | iex
```

This installs to `%LOCALAPPDATA%\Programs\cando`, adds to PATH, and creates a Start Menu shortcut.

**Windows (Manual)**

```powershell
# For AMD64 (most Windows PCs):
Invoke-WebRequest -Uri "https://github.com/cutoken/cando/releases/latest/download/cando-windows-amd64.exe" -OutFile "cando.exe"

# For ARM64 (Surface Pro X, Windows on ARM):
Invoke-WebRequest -Uri "https://github.com/cutoken/cando/releases/latest/download/cando-windows-arm64.exe" -OutFile "cando.exe"

Move-Item cando.exe C:\Users\$env:USERNAME\bin\  # choose any folder on PATH
```

> **Note:** If you launch the `.exe` from File Explorer, Windows SmartScreen will warn about an "unrecognized app." Choose "More info → Run anyway" once, or run it from PowerShell/CMD to skip the dialog. On macOS, running from Terminal bypasses Gatekeeper as well.

### Getting Started

1. Run `cando`
2. Open http://localhost:3737 (or the URL shown in terminal)
3. Configure your AI provider (Z.AI or OpenRouter) and API key
4. Select a workspace folder for your project
5. Start coding

Credentials are stored in `~/.cando/credentials.yaml`. Workspace data lives under `~/.cando/projects/`.

## What Can CanDo Build?

![Doom game built with CanDo](docs/images/doom_game.png)
*A fully functional Doom-style game built by CanDo in under 5 minutes*

### Community & Support

**Join our Discord server for help, discussions, and updates:**
- 🎮 **Discord**: https://discord.gg/fzWbCf9CA
- 💬 GitHub Discussions: https://github.com/cutoken/cando/discussions
- 🐛 Report Issues: https://github.com/cutoken/cando/issues

### Command-line Options

```bash
cando                              # Start web UI (default)
cando --sandbox /path/to/project   # Use specific workspace
cando --port 8080                  # Custom port
cando --setup                      # Manage credentials via CLI
cando -p "your prompt"             # One-shot mode (no UI)
cando --version                    # Show version
```

---

![CanDo Web UI](docs/images/cando-ui.png)

## Sample Projects

Projects built with CanDo:

| | |
|:---:|:---:|
| ![Pacman Game](docs/images/pacman.png) | ![Milkyway Animation](docs/images/milkyway-animation.png) |
| Pacman Game | Milkyway Animation |
| ![Moon Phases](docs/images/moon-phases-animation.png) | ![Sculpting Tool](docs/images/basic-sculpting-tool.png) |
| Moon Phases Animation | Sculpting Tool |

---

## Developer Setup

### Prerequisites

- Go 1.24 or later
- Git

### Build from Source

```bash
# Clone repository
git clone https://github.com/cutoken/cando.git
cd cando

# Build binary (cross-platform)
go build -o cando ./cmd/cando     # Linux/macOS
go build -o cando.exe ./cmd/cando  # Windows

# Run setup to configure credentials
./cando --setup

# Or run from source directly
go run ./cmd/cando --setup
```

### Development Workflow

**Live Reload (recommended):**
```bash
# Install air for live reload (cross-platform)
go install github.com/cosmtrek/air@latest

# Start with auto-reload on code changes
air
```

**Build for Current Platform:**
```bash
# Build for current platform
go build -o cando ./cmd/cando

# Optimized build with version
go build -ldflags="-s -w -X main.Version=dev" -o cando ./cmd/cando
```

**Build for All Platforms:**
```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/cando-linux-amd64 ./cmd/cando

# Linux arm64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/cando-linux-arm64 ./cmd/cando

# macOS amd64
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/cando-darwin-amd64 ./cmd/cando

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/cando-darwin-arm64 ./cmd/cando

# Windows amd64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/cando-windows-amd64.exe ./cmd/cando
```

**Using Makefile (optional, Unix/Linux only):**

If you have `make` installed:
```bash
make build    # Build for current platform
make all      # Build for all platforms
make dev      # Start with live reload
make clean    # Clean build artifacts
```

### CI/CD and Releases

- Continuous integration lives in `.github/workflows/ci.yml`. Every push to `main` and every pull request runs unit tests (`go test ./...`), `go vet`, and the cross-platform smoke build script (`./test/e2e/test_build.sh`) inside GitHub Actions.
- Tagging a commit with `v*` automatically triggers `.github/workflows/release.yml`, which rebuilds all platform binaries, generates installer-friendly `*-bin` artifacts, computes checksums, and publishes a GitHub Release with everything the install script expects.
- To cut a release locally with guardrails, run `./scripts/release.sh vX.Y.Z`. The script verifies a clean working tree, runs tests, cross-builds via `make release`, adds the raw binaries/checksums, and—after an explicit yes/no prompt—creates & pushes the tag (which kicks off the release workflow). Set `GIT_REMOTE` if you need to push somewhere other than `origin`.

### Project Structure
```
cmd/cando/          - Main entry point
internal/
  agent/            - Core orchestration, web UI
  config/           - Configuration management
  contextprofile/   - Memory compression strategies
  llm/              - Provider-agnostic interface
  zai/              - Z.AI client
  openrouter/       - OpenRouter client
  state/            - Conversation persistence
  tooling/          - Sandboxed tool registry
```

### Configuration Files

Cando loads configuration from `~/.cando/config.yaml` (created automatically with defaults on first run).

**Example config.yaml:**
```yaml
# Model settings
model: glm-4.6
provider_models:
  zai: glm-4.6
  openrouter: qwen/qwen2-72b-instruct

# LLM settings
temperature: 0.7
thinking_enabled: true

# Context/memory settings
context_profile: memory          # default or memory
context_message_threshold: 5000
context_conversation_threshold: 200000
context_protect_recent: 5

# Compaction settings
compaction_summary_prompt: "Summarize the following text in 40 words or fewer. Return only the summary. Capture important aspects and key facts in it."
summary_model: glm-4.5-air

# Workspace (can be overridden with --sandbox flag)
workspace_root: .

# System prompt (customize for your needs)
system_prompt: >-
  You are Cando, a concise and capable coding assistant.
```

**Credentials** are stored separately in `~/.cando/credentials.yaml` (managed via `cando --setup`)

**Environment Variables:**
- `CANDO_CREDENTIALS_PATH`: Override default credentials path (useful for CI/testing)
- `CANDO_CONFIG_PATH`: Override default config path (useful for CI/testing)

---

## Features

### Web UI

- **Session Management:** Create, switch, and manage multiple conversation sessions
- **Provider Switching:** Toggle between Z.AI and OpenRouter on the fly
- **Thinking Mode:** View LLM reasoning process (collapsible blocks)
- **Plan View:** Track multi-step execution plans
- **Request Cancellation:** Stop in-flight requests with one click
- **Theme Support:** Dark, light, midnight, and high contrast themes

### Conversation Persistence

Conversations auto-save to `~/.cando/projects/<workspace>/conversations/YYYY-MM-DD/<session-key>.json`

Resume anytime with:
```bash
cando --resume <session-key>
```

### Context Profiles

**Default Profile:**
- Full conversation history sent to LLM

**Memory Profile:**
- Auto-summarizes messages >1KB to ≤20 words
- Stores originals in SQLite (`memory.db`)
- Exposes `recall_memory()` and `pin_memory()` tools
- Max 5 concurrent pins
- Smart cooldown prevents re-compaction

### Sandboxed Tools

All tools operate within `workspace_root` for safety:

| Tool | Description |
|------|-------------|
| `shell` | Execute commands (timeout-bounded, sandboxed) |
| `read_file` | Read files with byte cap |
| `write_file` | Write/overwrite files |
| `edit_file` | Apply inline edits (search & replace) |
| `glob` | Find files by pattern |
| `grep` | Search file contents with context |
| `list_directory` | Enumerate files/folders |
| `apply_patch` | Apply unified diff patches |
| `background_process` | Start/monitor/kill long-running processes |
| `web_fetch_json` | Fetch and parse JSON from URLs |
| `update_plan` | Persist execution plans |
| `recall_memory` | Expand summarized memories |
| `pin_memory` | Protect memories from compaction |
| `current_datetime` | Get local time |
| `current_working_directory` | Get workspace root |

**Example - Background Process:**
```json
{"action":"start","command":["npm","run","dev"]}
{"action":"list"}
{"action":"logs","job_id":"job-...","stream":"stderr","tail_lines":200}
{"action":"kill","job_id":"job-..."}
```

### Helper Binaries

Drop executables in `<workspace_root>/bin` to make them available to shell/patch tools:

```bash
mkdir -p bin
# Add ripgrep, patch, jq, etc.
cp $(which rg) bin/
chmod +x bin/*
```

---

## Providers

### Z.AI

Fast Chinese models (GLM-4 series). Get your API key at [z.ai](https://z.ai).

### OpenRouter

Access to Claude, GPT-4, Qwen, and 100+ models. Get your API key at [openrouter.ai/keys](https://openrouter.ai/keys).

You can switch providers anytime from the web UI settings or via `cando --setup`.

---

## Troubleshooting

**Accidental loops (repeated commands):**
- Press `Esc` or click "Cancel" to stop the request
- Provide more specific instructions

**Provider quota errors (Z.AI code 1113):**
- API key has no active resource pack
- Top up account or switch to `openrouter`

**Build errors:**
- Ensure Go 1.24+ installed: `go version`
- Run `go mod download` to fetch dependencies

**Web UI not loading:**
- Check firewall settings
- Try explicit port: `cando --port 3737`
- Check logs at `~/.cando/projects/<workspace>/cando.log`

---

## Advanced Usage

### One-shot Mode

Run a single prompt without the web UI:
```bash
cando -p "explain this error: <paste error>"
```

### Custom Workspace

Isolate file operations to a specific directory:
```bash
cando --sandbox /path/to/project
```

### CI/Testing

Override credentials and config paths for testing environments:
```bash
# Use custom credentials and config files
export CANDO_CREDENTIALS_PATH=/tmp/test-creds.yaml
export CANDO_CONFIG_PATH=/tmp/test-config.yaml

# Create test credentials programmatically
cat > /tmp/test-creds.yaml <<EOF
default_provider: zai
providers:
  zai:
    api_key: ${ZAI_API_KEY}
EOF

# Create test config with appropriate model
cat > /tmp/test-config.yaml <<EOF
model: glm-4.6
provider_models:
  zai: glm-4.6
temperature: 0.7
thinking_enabled: true
EOF

# Run cando in CI
cando --sandbox /tmp/test-workspace
```

---

## Testing

### E2E Tests

Cando includes comprehensive end-to-end tests covering builds and provider functionality.

**Quick test:**
```bash
# Run all tests
./test/e2e/run_all.sh

# Build tests only (no API keys needed)
./test/e2e/test_build.sh

# CLI non-interactive test with mock LLM client
./test/e2e/test_cli_mock.sh

# Provider tests (requires API keys)
ZAI_API_KEY="..." ./test/e2e/test_providers.sh
```

**What's tested:**
- Cross-platform builds (Linux, macOS, Windows)
- Non-interactive CLI flow against the embedded mock LLM (`CANDO_MOCK_LLM=1`)
- ZAI provider integration
- OpenRouter provider integration
- One-shot prompt mode (`-p` flag)
- Version and help flags
- Code linting

See [test/e2e/README.md](test/e2e/README.md) for full documentation.

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and test:
   ```bash
   go build -o cando ./cmd/cando && ./cando
   ```
4. Run tests:
   ```bash
   # Build and lint tests (required)
   ./test/e2e/test_build.sh

   # Provider tests (optional, requires API keys)
   ZAI_API_KEY="..." ./test/e2e/test_providers.sh
   ```
5. Run linters and formatters:
   ```bash
   go fmt ./...
   go vet ./...
   ```
6. Commit and push
7. Open a Pull Request

---

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE) - see the LICENSE file for details.

## Support

- Issues: https://github.com/cutoken/cando/issues
- Discussions: https://github.com/cutoken/cando/discussions
