#!/usr/bin/env bash
# Beta release script - builds and prepares beta releases
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
DEV_DIR="${ROOT_DIR}/dev"

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

# Check we're on beta branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "beta" ]] && [[ "${FORCE_BRANCH:-}" != "1" ]]; then
    warn "Not on beta branch (current: $CURRENT_BRANCH)"
    echo "Switch to beta branch or set FORCE_BRANCH=1"
    exit 1
fi

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
        SUGGESTED="${LAST_TAG}-beta.1"
    fi
    
    read -rp "Enter beta version (suggested: $SUGGESTED): " VERSION
    VERSION="${VERSION:-$SUGGESTED}"
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+$ ]]; then
    error "Version must be like v1.0.0-beta.1"
fi

beta "Building Beta Release: $VERSION"
echo "================================"

# Run tests
info "Running tests..."
if ! go test ./... > /dev/null 2>&1; then
    error "Tests failed"
fi

info "Running linter..."
if ! go vet ./... > /dev/null 2>&1; then
    warn "Linter warnings detected"
fi

# Build all platforms
info "Building binaries..."
rm -rf "$DIST_DIR"
VERSION="$VERSION" make all

# Create beta-specific artifacts
info "Creating beta artifacts..."
cd "$DIST_DIR"

# Add beta suffix to binaries for clarity
for file in cando-*; do
    if [[ -f "$file" ]]; then
        beta_name="${file/cando-/cando-beta-}"
        cp "$file" "$beta_name"
        beta "Created: $beta_name"
    fi
done

# Generate checksums
sha256sum cando-* > checksums-beta.txt

# Create beta-binaries.json for installer
cat > beta-binaries.json <<EOF
{
  "version": "$VERSION",
  "date": "$(date -Iseconds)",
  "binaries": {
    "linux-amd64": "cando-linux-amd64",
    "linux-arm64": "cando-linux-arm64", 
    "darwin-amd64": "cando-darwin-amd64",
    "darwin-arm64": "cando-darwin-arm64",
    "windows-amd64": "cando-windows-amd64.exe"
  }
}
EOF

cd "$ROOT_DIR"

# Option to push to beta branch
echo ""
beta "Beta artifacts ready in $DIST_DIR"
echo ""
echo "Next steps:"
echo "1. Test locally: CANDO_BASE_URL=file://$DIST_DIR ./dev/install-beta.sh"
echo "2. Commit and push: git add -f dist/ && git commit -m 'Beta release $VERSION' && git push origin beta"
echo "3. Tag (optional): git tag $VERSION && git push origin $VERSION"
echo "4. Share installer: https://raw.githubusercontent.com/YOUR_REPO/cando/beta/dev/install-beta.sh"
echo ""

read -rp "Commit beta artifacts to git? [y/N] " commit_confirm
if [[ "$commit_confirm" =~ ^[Yy]$ ]]; then
    info "Committing beta artifacts..."
    git add -f dist/
    git commit -m "Beta release $VERSION

- Built from: $(git rev-parse --short HEAD)
- Branch: $CURRENT_BRANCH
- Date: $(date)
"
    
    read -rp "Push to origin/beta? [y/N] " push_confirm
    if [[ "$push_confirm" =~ ^[Yy]$ ]]; then
        git push origin "$CURRENT_BRANCH"
        beta "Pushed to origin/$CURRENT_BRANCH"
        
        # Optionally tag
        read -rp "Create and push tag $VERSION? [y/N] " tag_confirm
        if [[ "$tag_confirm" =~ ^[Yy]$ ]]; then
            git tag "$VERSION"
            git push origin "$VERSION"
            beta "Tagged as $VERSION"
        fi
    fi
fi

beta "Beta release $VERSION complete!"