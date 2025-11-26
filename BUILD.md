# Build & Release Guide

This document explains how to build, release, and distribute Cando.

## Development

For active development with auto-reload:

```bash
# Development mode with hot reload (recommended)
make dev
# or
DEV_MODE=true air

# This watches .go, .tmpl, .css, .js files
# Auto-rebuilds and restarts on changes
# Templates loaded from disk (not embedded)
```

For one-off builds during development:

```bash
# Build current platform only
make build

# Run directly
./cando web
```

**Note:** There is only ONE binary: `./cando`
- `make build` creates it
- `make dev` (air) creates `./tmp/cando`
- `make clean` removes both

## Quick Build Commands

```bash
# Build all platforms
make all

# Build specific platforms
make build-linux
make build-darwin
make build-windows

# Install locally to ~/.local/bin
make install

# Clean build artifacts
make clean

# Run tests
make test
```

## Build Output

All binaries are created in `dist/`:

```
dist/
├── cando-linux-amd64        # Linux x86_64
├── cando-linux-arm64        # Linux ARM64
├── cando-darwin-amd64       # macOS Intel
├── cando-darwin-arm64       # macOS Apple Silicon
└── cando-windows-amd64.exe  # Windows 64-bit
```

## Creating a Release

### Automated Release (GitHub Actions)

1. **Tag a version:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions automatically:**
   - Builds binaries for all platforms
   - Creates archives (.tar.gz for Unix, .zip for Windows)
   - Generates SHA256 checksums
   - Creates a GitHub Release
   - Uploads all artifacts

3. **Users can then install with:**
   ```bash
   curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
   ```

### Manual Release

If you prefer to release manually:

```bash
# 1. Build everything
make all

# 2. Create archives
cd dist

# Linux/macOS - create tar.gz archives
tar czf cando-linux-amd64.tar.gz cando-linux-amd64
tar czf cando-linux-arm64.tar.gz cando-linux-arm64
tar czf cando-darwin-amd64.tar.gz cando-darwin-amd64
tar czf cando-darwin-arm64.tar.gz cando-darwin-arm64

# Windows - create zip archive
zip cando-windows-amd64.zip cando-windows-amd64.exe

# 3. Generate checksums
sha256sum *.tar.gz *.zip > checksums.txt

# 4. Create GitHub release and upload files
gh release create v1.0.0 \
  cando-linux-amd64.tar.gz \
  cando-linux-arm64.tar.gz \
  cando-darwin-amd64.tar.gz \
  cando-darwin-arm64.tar.gz \
  cando-windows-amd64.zip \
  cando-linux-amd64 \
  cando-linux-arm64 \
  cando-darwin-amd64 \
  cando-darwin-arm64 \
  cando-windows-amd64.exe \
  checksums.txt \
  --title "Release v1.0.0" \
  --notes "Release notes here"
```

## Version Management

The version is embedded in the binary at build time:

```bash
# Version from git tag/commit
make all
./dist/cando-linux-amd64 --version
# Output: Cando version v1.0.0 (or commit hash)

# Custom version
VERSION=v2.0.0-beta make all
```

The Makefile automatically:
- Uses git tags if available
- Falls back to commit hash
- Appends `-dirty` if working directory has uncommitted changes

## Build Flags

The Makefile uses these Go build flags:

```
-ldflags "-X main.Version=$(VERSION) -s -w"
```

- `-X main.Version=$(VERSION)` - Sets the Version variable
- `-s` - Omit symbol table (reduces binary size)
- `-w` - Omit DWARF symbol table (reduces binary size)

This typically reduces binary size by ~30-40%.

## Installation Script

The `install.sh` script:

1. Detects OS and architecture automatically
2. Downloads the correct binary from GitHub releases
3. Installs to `~/.local/bin` (no sudo needed)
4. Makes it executable
5. Guides user on adding to PATH if needed

### Customizing the installer

Users can customize via environment variables:

```bash
# Install to custom directory
CANDO_INSTALL_DIR="$HOME/bin" curl -fsSL .../install.sh | bash
```

## Testing Builds

Before releasing, test each platform build:

```bash
# Linux (if on Linux)
./dist/cando-linux-amd64 --version
./dist/cando-linux-amd64 --help

# macOS (if on macOS)
./dist/cando-darwin-amd64 --version

# Windows (use WSL or Wine)
wine ./dist/cando-windows-amd64.exe --version
```

## GitHub Actions Workflow

The `.github/workflows/release.yml` workflow:

- **Triggers:** On git tag push (v*)
- **Builds:** All platform binaries
- **Creates:** Release with auto-generated notes
- **Uploads:** All binaries + archives + checksums

To trigger:
```bash
git tag v1.0.0
git push origin v1.0.0
```

## Troubleshooting

### Build fails with "command not found"

Make sure Go is installed:
```bash
go version  # Should show 1.24+
```

### Cross-compilation issues

The build uses pure Go, so cross-compilation should work. If you encounter issues:

```bash
# Install specific toolchain
GOOS=windows GOARCH=amd64 go install std

# Verify CGO is disabled (should be by default)
CGO_ENABLED=0 make all
```
