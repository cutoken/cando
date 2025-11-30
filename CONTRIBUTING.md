# Contributing to Cando

Thanks for your interest in contributing to Cando, an AI coding agent. This document covers the contribution workflow.

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Set up the development environment:
   ```bash
   git clone https://github.com/<your-fork>/cando.git
   cd cando
   go mod download
   ```

## Development Workflow

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make all

# Development mode with hot reload
make dev
```

### Testing

```bash
# Run unit tests
go test ./...

# Run linting
go vet ./...

# Run E2E tests (no API keys needed)
./test/e2e/test_build.sh
./test/e2e/test_cli_mock.sh

# Run provider tests (requires API keys)
ZAI_API_KEY="..." ./test/e2e/test_providers.sh
```

### Code Style

- Follow standard Go conventions
- Run `go fmt ./...` before committing
- Run `go vet ./...` to catch common issues
- Keep functions focused and reasonably sized
- Add comments for non-obvious logic

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feature/add-new-provider`
- `fix/memory-leak-in-compaction`
- `docs/update-install-guide`

### Commit Messages

Write clear, concise commit messages:
- Use present tense ("Add feature" not "Added feature")
- First line should be under 72 characters
- Reference issues when applicable ("Fix #123")

### Pull Requests

1. Ensure all tests pass locally
2. Update documentation if needed
3. Fill out the PR template completely
4. Keep PRs focused on a single change
5. Be responsive to review feedback

## What to Contribute

### Good First Issues

Look for issues labeled `good first issue` for beginner-friendly tasks.

### Feature Requests

Before implementing a major feature:
1. Check existing issues and discussions
2. Open an issue to discuss the approach
3. Wait for feedback before starting work

### Bug Reports

When reporting bugs, include:
- Steps to reproduce
- Expected vs actual behavior
- OS and Go version
- Relevant logs or error messages

## License

By contributing to Cando, you agree that your contributions will be licensed under the GNU Affero General Public License v3.0.

## Project Structure

```
cmd/cando/          - Entry point
internal/
  agent/            - Core orchestration, web UI
  config/           - Configuration
  contextprofile/   - Memory compression
  llm/              - Provider interface
  zai/              - Z.AI client
  openrouter/       - OpenRouter client
  state/            - Conversation persistence
  tooling/          - Sandboxed tools
```

## Configuration

Config: `~/.cando/config.yaml` (created on first run)

```yaml
model: glm-4.6
provider_models:
  zai: glm-4.6
  openrouter: qwen/qwen2-72b-instruct
temperature: 0.7
thinking_enabled: true
context_profile: memory
```

Credentials: `~/.cando/credentials.yaml` (via `cando --setup`)

Environment variables:
- `CANDO_CREDENTIALS_PATH` - Override credentials path
- `CANDO_CONFIG_PATH` - Override config path

## CI/CD

- Push to `main` or PR triggers CI (tests, vet, build)
- Tag with `v*` triggers release workflow
- Release script: `./scripts/release.sh vX.Y.Z`

## Questions?

- Open a [Discussion](https://github.com/cutoken/cando/discussions) for general questions
- Open an [Issue](https://github.com/cutoken/cando/issues) for bugs or feature requests
