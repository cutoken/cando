#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
REMOTE_DEFAULT="origin"

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "Missing required command: $1" >&2
		exit 1
	fi
}

for cmd in git go make tar zip sha256sum; do
	require_cmd "$cmd"
done

REMOTE="${GIT_REMOTE:-$REMOTE_DEFAULT}"
if ! git rev-parse --git-dir >/dev/null 2>&1; then
	echo "This script must be run from inside the git repository." >&2
	exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
	echo "Working tree has uncommitted changes. Commit or stash them before releasing." >&2
	exit 1
fi

REMOTE_URL="$(git remote get-url "$REMOTE" 2>/dev/null || true)"
if [[ -z "$REMOTE_URL" ]]; then
	echo "Remote '$REMOTE' is not configured. Set GIT_REMOTE to a valid remote name." >&2
	exit 1
fi
REMOTE_REPO=$(echo "$REMOTE_URL" | sed -E 's#.*github.com[:/](.+)(\.git)?#\1#')
ACTIONS_URL="https://github.com/${REMOTE_REPO}/actions"
if [[ "$REMOTE_REPO" == "$REMOTE_URL" ]]; then
	ACTIONS_URL="$REMOTE_URL"
fi

git fetch --tags "$REMOTE" >/dev/null 2>&1 || true

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
	read -rp "Enter the release tag (e.g., v0.5.0): " VERSION
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z]+)?$ ]]; then
	echo "Version must look like v0.0.0 or v0.0.0-rc1." >&2
	exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
	echo "Tag '$VERSION' already exists locally." >&2
	exit 1
fi

if git ls-remote --tags "$REMOTE" "refs/tags/$VERSION" | grep -q "$VERSION"; then
	echo "Tag '$VERSION' already exists on remote '$REMOTE'." >&2
	exit 1
fi

cat <<EOF
About to:
  1) Run unit tests (go test ./...) and go vet
  2) Run ./test/e2e/test_build.sh (cross-platform smoke build)
  3) Build release artifacts via 'make release'
  4) Copy raw binaries for installer consumption and generate checksums
  5) Create and push the git tag '$VERSION' to '$REMOTE' (triggers the GitHub release workflow)
EOF

read -rp "Continue? [y/N] " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
	echo "Aborted."
	exit 0
fi

echo "==> Running unit tests"
go test ./...
echo "==> Running go vet"
go vet ./...
echo "==> Running E2E build smoke test"
./test/e2e/test_build.sh

echo "==> Building release artifacts (VERSION=$VERSION)"
VERSION="$VERSION" make release

echo "==> Preparing installer-friendly binaries"
mkdir -p "$DIST_DIR"
cp "$DIST_DIR/cando-linux-amd64" "$DIST_DIR/cando-linux-amd64-bin"
cp "$DIST_DIR/cando-linux-arm64" "$DIST_DIR/cando-linux-arm64-bin"
cp "$DIST_DIR/cando-darwin-amd64" "$DIST_DIR/cando-darwin-amd64-bin"
cp "$DIST_DIR/cando-darwin-arm64" "$DIST_DIR/cando-darwin-arm64-bin"

echo "==> Generating checksums"
(
	cd "$DIST_DIR"
	sha256sum *.tar.gz *.zip *-bin *.exe > checksums.txt
)

echo "Artifacts ready in $DIST_DIR:"
ls -lh "$DIST_DIR"

read -rp "Create and push git tag '$VERSION'? [y/N] " tag_confirm
if [[ ! "$tag_confirm" =~ ^[Yy]$ ]]; then
	echo "Tag push skipped. Artifacts remain in $DIST_DIR"
	exit 0
fi

cleanup_tag_on_failure() {
	if [[ "${TAG_CREATED:-0}" -eq 1 ]]; then
		echo "Removing local tag '$VERSION' due to failure."
		git tag -d "$VERSION" >/dev/null 2>&1 || true
	fi
}

trap cleanup_tag_on_failure ERR

git tag "$VERSION"
TAG_CREATED=1
git push "$REMOTE" "$VERSION"
TAG_CREATED=0
trap - ERR

cat <<EOF
Tag '$VERSION' pushed to '$REMOTE'.
GitHub Actions will build and publish the release automatically.
Monitor ${ACTIONS_URL} for status.
EOF
