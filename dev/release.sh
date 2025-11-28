#!/usr/bin/env bash
# Release helper - creates and pushes release tags for stable versions
# GitHub Actions will automatically build and create releases
# Usage: ./dev/release.sh [version] [--dry-run]
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
success() { echo -e "${BLUE}[SUCCESS]${NC} $1"; }

# Parse arguments
DRY_RUN=false
VERSION=""
for arg in "$@"; do
    if [[ "$arg" == "--dry-run" ]]; then
        DRY_RUN=true
    elif [[ "$arg" =~ ^v[0-9] ]]; then
        VERSION="$arg"
    fi
done

if [[ "$DRY_RUN" == "true" ]]; then
    warn "DRY RUN MODE - No tag will be created"
fi

# Check we're in a git repo
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    error "Not in a git repository"
fi

# Check we're on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    error "Releases must be created from main branch (currently on: $CURRENT_BRANCH)"
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    error "You have uncommitted changes. Please commit or stash them first"
fi

# Check remote is up to date (skip in dry run)
if [[ "$DRY_RUN" == "false" ]]; then
    info "Fetching latest from origin..."
    git fetch origin main
    LOCAL=$(git rev-parse HEAD)
    REMOTE=$(git rev-parse origin/main)
    if [ "$LOCAL" != "$REMOTE" ]; then
        error "Your main branch is not up to date with origin/main. Please pull/push first"
    fi
fi

# Get version if not provided as argument
if [[ -z "$VERSION" ]]; then
    # Get the highest version tag (including betas)
    HIGHEST_TAG=$(git tag --sort=-version:refname | head -1 2>/dev/null || echo "v0.0.0")
    
    if [[ -z "$HIGHEST_TAG" ]]; then
        HIGHEST_TAG="v0.0.0"
    fi
    
    info "Highest version tag: $HIGHEST_TAG"
    
    # Check if highest tag is a beta - if so, we're releasing that version
    if [[ "$HIGHEST_TAG" =~ -beta\. ]]; then
        # Remove beta suffix - this IS the version we should release
        VERSION=$(echo "$HIGHEST_TAG" | sed 's/-beta.*//')
        if [[ "$DRY_RUN" == "true" ]]; then
            info "Dry run will test release for: $VERSION (from $HIGHEST_TAG)"
        else
            echo ""
            echo "You have beta $HIGHEST_TAG - releasing as $VERSION"
            echo ""
            read -rp "Release version $VERSION? (y/N) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                # Ask for different version
                read -rp "Enter release version (e.g., v1.0.0): " VERSION
                if [[ -z "$VERSION" ]]; then
                    error "Version is required"
                fi
            fi
        fi
    else
        # Not a beta, suggest next versions
        MAJOR=$(echo "$HIGHEST_TAG" | cut -d. -f1 | sed 's/v//')
        MINOR=$(echo "$HIGHEST_TAG" | cut -d. -f2)
        PATCH=$(echo "$HIGHEST_TAG" | cut -d. -f3 | cut -d- -f1)
        
        NEXT_PATCH="v${MAJOR}.${MINOR}.$((PATCH + 1))"
        NEXT_MINOR="v${MAJOR}.$((MINOR + 1)).0"
        NEXT_MAJOR="v$((MAJOR + 1)).0.0"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            # In dry run, use next minor as default
            VERSION="$NEXT_MINOR"
            info "Dry run will test release for: $VERSION"
        else
            echo ""
            echo "Current version: $HIGHEST_TAG"
            echo ""
            echo "Suggested versions:"
            echo "  1) $NEXT_PATCH - Patch release (bug fixes)"
            echo "  2) $NEXT_MINOR - Minor release (new features)"
            echo "  3) $NEXT_MAJOR - Major release (breaking changes)"
            echo ""
            
            read -rp "Enter release version (e.g., v1.0.0): " VERSION
            if [[ -z "$VERSION" ]]; then
                error "Version is required"
            fi
        fi
    fi
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    error "Version must be like v1.0.0 (no pre-release suffix for stable releases)"
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    error "Tag $VERSION already exists"
fi

success "Creating Release: $VERSION"
echo "================================"

# Check formatting
info "Checking code formatting..."
UNFORMATTED=$(gofmt -l .)
if [[ -n "$UNFORMATTED" ]]; then
    error "Code is not properly formatted. Run 'go fmt ./...' to fix:\n$UNFORMATTED"
fi

# Run linter
info "Running linter..."
if ! go vet ./... > /dev/null 2>&1; then
    error "Linter failed. Fix issues before releasing"
fi

# Run tests
info "Running tests..."
if ! go test ./... > /dev/null 2>&1; then
    error "Tests failed. Fix failing tests before releasing"
fi

# Build test
info "Testing build..."
if ! go build -o /tmp/cando-release-test cmd/cando/main.go 2>/dev/null; then
    error "Build failed"
fi
rm -f /tmp/cando-release-test

# Get release notes from user
if [[ "$DRY_RUN" == "false" ]]; then
    # Get the previous release tag for reference
    PREV_TAG=$(git tag --sort=-version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 2>/dev/null || "")
    
    echo ""
    echo "Enter release notes for $VERSION"
    if [[ -n "$PREV_TAG" ]]; then
        echo "(Previous release was $PREV_TAG)"
        echo ""
        echo "Recent commits:"
        git log ${PREV_TAG}..HEAD --oneline 2>/dev/null | head -10
    fi
    echo ""
    echo "Enter release notes (press Ctrl+D when done):"
    echo "----------------------------------------"
    RELEASE_NOTES=$(cat)
    
    if [[ -z "$RELEASE_NOTES" ]]; then
        error "Release notes cannot be empty"
    fi
    
    # Show release notes for confirmation
    echo ""
    echo "Release Notes:"
    echo "=============="
    echo "$RELEASE_NOTES"
    echo "=============="
    echo ""
else
    # In dry run, skip release notes
    RELEASE_NOTES="(Release notes will be entered during actual release)"
fi

if [[ "$DRY_RUN" == "true" ]]; then
    success "DRY RUN PASSED - All checks successful!"
    echo ""
    echo "To create the actual release, run without --dry-run:"
    echo -e "   ${GREEN}./dev/release.sh $VERSION${NC}"
else
    read -rp "Create release with these notes? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        error "Aborted"
    fi

    # Create annotated tag
    info "Creating tag: $VERSION"
    git tag -a "$VERSION" -m "$RELEASE_NOTES"

    success "Tag created successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Push the tag to trigger GitHub Actions:"
    echo -e "   ${GREEN}git push origin $VERSION${NC}"
    echo ""
    echo "2. GitHub Actions will automatically:"
    echo "   - Build binaries for all platforms"
    echo "   - Create a release on GitHub"
    echo "   - Publish to the releases page"
    echo ""
    echo "3. Users can install with:"
    echo -e "   ${BLUE}curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/install.sh | bash${NC}"
    echo ""
    echo "4. To delete the tag if needed:"
    echo "   git tag -d $VERSION"
    echo "   git push origin --delete $VERSION"
    echo ""
    echo "5. Don't forget to:"
    echo "   - Update documentation if needed"
    echo "   - Announce the release"
fi