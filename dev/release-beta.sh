#!/usr/bin/env bash
# Beta release helper - creates and pushes beta tags
# GitHub Actions will automatically build and create prereleases
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }
beta() { echo -e "${BLUE}[BETA]${NC} $1"; }

# Check we're in a git repo
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    error "Not in a git repository"
fi

# Get version argument
VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
    # Suggest next beta version
    LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    if [[ "$LAST_TAG" =~ beta ]]; then
        # Increment beta number
        BASE=$(echo "$LAST_TAG" | sed 's/-beta.*//')
        NUM=$(echo "$LAST_TAG" | sed 's/.*beta.//')
        NEXT_NUM=$((NUM + 1))
        SUGGESTED="${BASE}-beta.${NEXT_NUM}"
    else
        # First beta for this version
        MAJOR=$(echo "$LAST_TAG" | cut -d. -f1 | sed 's/v//')
        MINOR=$(echo "$LAST_TAG" | cut -d. -f2)
        PATCH=$(echo "$LAST_TAG" | cut -d. -f3 | cut -d- -f1)
        # Increment minor for beta
        NEXT_MINOR=$((MINOR + 1))
        SUGGESTED="v${MAJOR}.${NEXT_MINOR}.0-beta.1"
    fi
    
    read -rp "Enter beta version (suggested: $SUGGESTED): " VERSION
    VERSION="${VERSION:-$SUGGESTED}"
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+$ ]]; then
    error "Version must be like v1.0.0-beta.1"
fi

beta "Creating Beta Release: $VERSION"
echo "================================"

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    warn "You have uncommitted changes:"
    git status --short
    read -rp "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        error "Aborted"
    fi
fi

# Run tests
if [[ "${SKIP_TESTS:-}" != "1" ]]; then
    info "Running tests..."
    if ! go test ./... > /dev/null 2>&1; then
        warn "Tests failed. Continue anyway? (y/N) "
        read -rp "" -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            error "Aborted"
        fi
    fi
else
    warn "Skipping tests (SKIP_TESTS=1)"
fi

info "Running linter..."
if ! go vet ./... > /dev/null 2>&1; then
    warn "Linter warnings detected"
fi

# Create annotated tag
info "Creating tag: $VERSION"
TAG_MESSAGE="Beta Release $VERSION

This is a prerelease version for testing.

Changes since last release:
$(git log $(git describe --tags --abbrev=0 2>/dev/null)..HEAD --oneline 2>/dev/null | head -10 || echo "Initial beta release")"

git tag -a "$VERSION" -m "$TAG_MESSAGE"

beta "Tag created successfully!"
echo ""
echo "Next steps:"
echo "1. Push the tag to trigger GitHub Actions:"
echo "   ${GREEN}git push origin $VERSION${NC}"
echo ""
echo "2. GitHub Actions will automatically:"
echo "   - Build binaries for all platforms"
echo "   - Create a prerelease on GitHub"
echo "   - Make it available for beta testers"
echo ""
echo "3. Testers can install with:"
echo "   ${BLUE}curl -fsSL https://raw.githubusercontent.com/YOUR_REPO/cando/main/dev/install-beta.sh | bash${NC}"
echo ""
echo "4. To delete the tag if needed:"
echo "   git tag -d $VERSION"
echo "   git push origin --delete $VERSION"