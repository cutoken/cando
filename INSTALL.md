# Installation Guide

Cando is an AI coding agent that runs locally. Install on Linux, macOS, or Windows without admin privileges.

## Quick Install (Linux/macOS)

Run this one-liner:

```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
```

Or if you want to review the script first:

```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh -o install.sh
chmod +x install.sh
./install.sh
```

## Quick Install (Windows - Git Bash/WSL)

Same as Linux/macOS - the installer detects your platform automatically:

```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
```

## Manual Installation

### 1. Download the binary for your platform

Go to the [Releases page](https://github.com/cutoken/cando/releases) and download:

- **Linux (x86_64)**: `cando-linux-amd64-bin`
- **Linux (ARM64)**: `cando-linux-arm64-bin`
- **macOS (Intel)**: `cando-darwin-amd64-bin`
- **macOS (Apple Silicon)**: `cando-darwin-arm64-bin`
- **Windows**: `cando-windows-amd64.exe`

### 2. Install to your local bin directory

#### Linux/macOS:
```bash
# Create bin directory if it doesn't exist
mkdir -p ~/.local/bin

# Move the binary (rename to cando)
mv cando-*-bin ~/.local/bin/cando
chmod +x ~/.local/bin/cando

# Add to PATH (if not already)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### Windows:
```powershell
# Create bin directory
New-Item -ItemType Directory -Force -Path $HOME\.local\bin

# Move the binary
Move-Item cando-windows-amd64.exe $HOME\.local\bin\cando.exe

# Add to PATH (run as admin or update user PATH in System Properties)
$env:Path += ";$HOME\.local\bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path, [EnvironmentVariableTarget]::User)
```

## Building from Source

If you have Go 1.24+ installed:

```bash
# Clone the repository
git clone https://github.com/cutoken/cando.git
cd cando

# Build for current platform
go build -o cando ./cmd/cando

# Or build for all platforms (optional, uses Makefile)
make all

# Binaries will be in ./dist/
```

## Customizing Installation

### Custom install directory

```bash
export CANDO_INSTALL_DIR="$HOME/bin"
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash
```

## Verifying Installation

After installation, verify it works:

```bash
cando --help
```

## Uninstalling

Simply remove the binary:

```bash
rm ~/.local/bin/cando
```

## Getting Started

1. Run Cando:
```bash
cando
```

2. Your browser opens automatically to `http://127.0.0.1:3737`

3. On first run:
   - If no credentials exist, configure your AI provider (Z.AI or OpenRouter) and enter your API key
   - Select a workspace folder for your project

**CLI options:**
```bash
cando --sandbox /path/to/project   # Use specific workspace
cando --port 8080                  # Custom port
cando --setup                      # Manage credentials via CLI
```

## Configuration

Cando loads configuration from `~/.cando/config.yaml` (created automatically with defaults).

**Example config.yaml:**
```yaml
model: glm-4.6
provider_models:
  zai: glm-4.6
  openrouter: qwen/qwen2-72b-instruct
temperature: 0.7
workspace_root: .
thinking_enabled: true
context_profile: memory
```

Edit `~/.cando/config.yaml` to customize these settings.

## Troubleshooting

### Command not found

Make sure `~/.local/bin` is in your PATH:

```bash
echo $PATH | grep ".local/bin"
```

If not, add it:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Make it permanent by adding to your shell config (`~/.bashrc`, `~/.zshrc`, etc.)

### Permission denied

Make sure the binary is executable:

```bash
chmod +x ~/.local/bin/cando
```

### Port already in use

Cando auto-selects ports starting at 3737. If all ports are busy, specify one manually:

```bash
cando --port 3738
```
