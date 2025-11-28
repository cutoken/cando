#!/usr/bin/env bash
# Cando BETA installer - installs from beta branch
# Usage: curl -fsSL https://raw.githubusercontent.com/cutoken/cando/beta/install-beta.sh | bash

set -e

# Configuration for BETA channel
REPO_OWNER="${CANDO_REPO_OWNER:-cutoken}"
REPO_NAME="${CANDO_REPO_NAME:-cando}"
BRANCH="${CANDO_BRANCH:-beta}"  # Beta branch by default
INSTALL_DIR="${CANDO_INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="cando-beta"  # Different binary name to avoid conflicts
BASE_URL="${CANDO_BASE_URL:-}"  # Allow custom hosting

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

beta_notice() {
    echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                   BETA VERSION                       ║${NC}"
    echo -e "${BLUE}║  This is a beta release for testing purposes only.   ║${NC}"
    echo -e "${BLUE}║  Report issues to the development team.              ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
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

# Download and install binary
install_binary() {
    local platform download_url tmp_file

    platform=$(detect_platform)
    
    beta_notice
    
    info "Installing Cando BETA for ${platform}..."
    info "Branch: ${BRANCH}"

    # Determine download URL
    if [ -n "$BASE_URL" ]; then
        # Custom hosting (e.g., your own server)
        if [[ "$platform" == *"windows"* ]]; then
            download_url="${BASE_URL}/cando-${platform}.exe"
            BINARY_NAME="${BINARY_NAME}.exe"
        else
            download_url="${BASE_URL}/cando-${platform}"
        fi
        info "Using custom URL: ${download_url}"
    else
        # GitHub raw content from beta branch
        local raw_base="https://github.com/${REPO_OWNER}/${REPO_NAME}/raw/${BRANCH}/dist"
        if [[ "$platform" == *"windows"* ]]; then
            download_url="${raw_base}/cando-${platform}.exe"
            BINARY_NAME="${BINARY_NAME}.exe"
        else
            download_url="${raw_base}/cando-${platform}"
        fi
        info "Downloading from beta branch: ${download_url}"
    fi

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Download binary
    tmp_file=$(mktemp)
    info "Downloading beta binary..."
    if ! curl -fsSL "$download_url" -o "$tmp_file" 2>/dev/null; then
        warn "Direct download failed, trying to build from source..."
        build_from_source "$platform"
        return
    fi

    # Install binary
    mv "$tmp_file" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    info "Cando BETA installed to: ${INSTALL_DIR}/${BINARY_NAME}"
    
    # Create symlink for convenience
    if [ -L "${INSTALL_DIR}/cando" ] || [ ! -f "${INSTALL_DIR}/cando" ]; then
        ln -sf "${BINARY_NAME}" "${INSTALL_DIR}/cando"
        info "Created symlink: cando -> ${BINARY_NAME}"
    else
        warn "Regular cando binary exists, beta installed as ${BINARY_NAME}"
    fi
}

# Build from source if binary not available
build_from_source() {
    local platform="$1"
    
    info "Building from beta branch source..."
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        error "Go is required to build from source. Install from https://golang.org"
    fi
    
    # Clone and build
    local tmp_dir=$(mktemp -d)
    cd "$tmp_dir"
    
    info "Cloning beta branch..."
    git clone --depth 1 --branch "$BRANCH" "https://github.com/${REPO_OWNER}/${REPO_NAME}.git"
    cd "$REPO_NAME"
    
    info "Building beta version..."
    go build -ldflags="-X main.Version=${BRANCH}-$(git rev-parse --short HEAD)" \
             -o "${INSTALL_DIR}/${BINARY_NAME}" ./cmd/cando
    
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    cd /
    rm -rf "$tmp_dir"
    
    info "Built and installed from source"
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
    if [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        info "Installation successful!"
        echo ""
        
        # Try to get version
        if version=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>/dev/null); then
            info "Installed version: $version"
        fi
        
        echo ""
        info "Run '${BINARY_NAME} --help' to get started"
        info "Report beta issues at: https://github.com/${REPO_OWNER}/${REPO_NAME}/issues"
    else
        error "Installation verification failed"
    fi
}

# Main installation flow
main() {
    echo "Cando BETA Installer"
    echo "===================="
    echo ""

    install_binary
    check_path
    verify_installation

    echo ""
    info "To uninstall beta, run: rm ${INSTALL_DIR}/${BINARY_NAME}"
    info "To switch to stable, run the regular installer"
}

main