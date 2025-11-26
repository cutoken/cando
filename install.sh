#!/usr/bin/env bash
# Cando installer - works on Linux, macOS, and Windows (Git Bash/WSL)
# Usage: curl -fsSL https://raw.githubusercontent.com/USER/REPO/main/install.sh | bash

set -e

# Configuration
REPO_OWNER="${CANDO_REPO_OWNER:-cutoken}"
REPO_NAME="${CANDO_REPO_NAME:-cando}"
INSTALL_DIR="${CANDO_INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="cando"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)          error "Unsupported operating system: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)              error "Unsupported architecture: $(uname -m)" ;;
    esac

    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        warn "Could not fetch latest version, using 'latest'"
        echo "latest"
    else
        echo "$version"
    fi
}

# Download and install binary
install_binary() {
    local platform version download_url tmp_file

    platform=$(detect_platform)
    version=$(get_latest_version)

    info "Installing Cando for ${platform}..."
    info "Version: ${version}"

    # Construct download URL
    if [ "$version" = "latest" ]; then
        if [[ "$platform" == *"windows"* ]]; then
            download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/cando-${platform}.exe"
            BINARY_NAME="${BINARY_NAME}.exe"
        else
            download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/cando-${platform}-bin"
        fi
    else
        if [[ "$platform" == *"windows"* ]]; then
            download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/cando-${platform}.exe"
            BINARY_NAME="${BINARY_NAME}.exe"
        else
            download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/cando-${platform}-bin"
        fi
    fi

    info "Downloading from: ${download_url}"

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Download binary
    tmp_file=$(mktemp)
    if ! curl -fsSL "$download_url" -o "$tmp_file"; then
        error "Failed to download binary from ${download_url}"
    fi

    # Install binary
    mv "$tmp_file" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    info "Cando installed to: ${INSTALL_DIR}/${BINARY_NAME}"
}

# Check if directory is in PATH
check_path() {
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        warn "${INSTALL_DIR} is not in your PATH"
        echo ""
        echo "Add this line to your shell configuration file:"
        echo ""

        if [ -n "$BASH_VERSION" ]; then
            echo "    echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
            echo "    source ~/.bashrc"
        elif [ -n "$ZSH_VERSION" ]; then
            echo "    echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
            echo "    source ~/.zshrc"
        else
            echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        fi
        echo ""
    else
        info "Installation directory is in PATH"
    fi
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        info "Installation successful!"
        info "Run 'cando --help' to get started"
    elif [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        info "Installation successful!"
        warn "But '${BINARY_NAME}' is not in your PATH yet"
        info "Run '${INSTALL_DIR}/${BINARY_NAME} --help' to get started"
    else
        error "Installation verification failed"
    fi
}

# Main installation flow
main() {
    echo "Cando Installer"
    echo "===================="
    echo ""

    install_binary
    check_path
    verify_installation

    echo ""
    info "To uninstall, simply run: rm ${INSTALL_DIR}/${BINARY_NAME}"
}

main
