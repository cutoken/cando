#!/usr/bin/env bash
# Cando BETA installer - installs from GitHub prereleases
# Usage: curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/dev/install-beta.sh | bash
# Or specify a version: BETA_VERSION=v1.0.0-beta.1 curl -fsSL ... | bash

set -e

# Configuration for BETA channel
REPO_OWNER="${CANDO_REPO_OWNER:-cutoken}"
REPO_NAME="${CANDO_REPO_NAME:-cando}"
INSTALL_DIR="${CANDO_INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="cando-beta"  # Different binary name to avoid conflicts
BETA_VERSION="${CANDO_BETA_VERSION:-}"  # Specific beta version to install
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

# Get latest beta version from GitHub prereleases
get_latest_beta_version() {
    local version
    
    info "Fetching latest beta version..."
    
    # Fetch all releases and find the latest prerelease
    version=$(curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases" 2>/dev/null | \
              python3 -c "
import sys, json
releases = json.load(sys.stdin)
for release in releases:
    if release.get('prerelease', False):
        print(release['tag_name'])
        break
" 2>/dev/null)
    
    if [ -z "$version" ]; then
        # Fallback to simpler parsing if python3 isn't available
        version=$(curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases" 2>/dev/null | \
                  grep -B 5 '"prerelease": true' | \
                  grep '"tag_name"' | \
                  head -1 | \
                  sed -E 's/.*"([^"]+)".*/\1/')
    fi
    
    if [ -z "$version" ]; then
        error "No beta releases found. Please check https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
    fi
    
    echo "$version"
}

# Download and install binary
install_binary() {
    local platform download_url tmp_file version
    
    platform=$(detect_platform)
    
    beta_notice
    
    info "Installing Cando BETA for ${platform}..."
    
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
        version="custom"
    else
        # Get beta version
        if [ -z "$BETA_VERSION" ]; then
            version=$(get_latest_beta_version)
        else
            version="$BETA_VERSION"
        fi
        
        info "Installing beta version: ${version}"
        
        # GitHub releases URL for prereleases
        if [[ "$platform" == *"windows"* ]]; then
            download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/cando-${platform}.exe"
            BINARY_NAME="${BINARY_NAME}.exe"
        else
            download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/cando-${platform}-bin"
        fi
        info "Downloading from GitHub releases: ${download_url}"
    fi
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Download binary
    tmp_file=$(mktemp)
    info "Downloading beta binary..."
    if ! curl -fsSL "$download_url" -o "$tmp_file" 2>/dev/null; then
        echo -e "${RED}[ERROR]${NC} Failed to download from: $download_url"
        echo ""
        echo "Possible reasons:"
        echo "1. Beta version $version doesn't exist"
        echo "2. Network issues"
        echo "3. GitHub API rate limit"
        echo ""
        echo "Try:"
        echo "- Check available releases at https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
        echo "- Specify a version: CANDO_BETA_VERSION=v1.0.0-beta.1"
        exit 1
    fi
    
    # Install binary
    mv "$tmp_file" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    
    info "Cando BETA installed to: ${INSTALL_DIR}/${BINARY_NAME}"
    
    # Check for regular cando
    if [ -f "${INSTALL_DIR}/cando" ] && [ ! -L "${INSTALL_DIR}/cando" ]; then
        warn "Regular cando binary exists, beta installed as ${BINARY_NAME}"
    fi
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
        local version=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>/dev/null || echo "unknown")
        info "Installation successful!"
        info "Installed version: ${version}"
        echo ""
        info "Run '${BINARY_NAME} --help' to get started"
        info "Report beta issues at: https://github.com/${REPO_OWNER}/${REPO_NAME}/issues"
        echo ""
        info "To uninstall beta, run: rm ${INSTALL_DIR}/${BINARY_NAME}"
        info "To switch to stable, run the regular installer"
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
}

main "$@"