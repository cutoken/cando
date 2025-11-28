# Beta Testing Setup Guide

This guide explains how to set up and manage beta releases for Cando.

## Overview

Beta releases use GitHub's prerelease feature to distribute test builds without affecting stable releases:
- **Beta tags** (e.g., `v1.0.0-beta.1`) trigger GitHub Actions to build prereleases
- **Main installer** ignores prereleases, only installs stable versions  
- **Beta installer** fetches from prereleases only
- **No binaries committed** to the repository

## Quick Start

### For Release Managers

1. **Create a beta release:**
   ```bash
   ./dev/release-beta.sh v1.0.0-beta.1
   # or let it suggest a version:
   ./dev/release-beta.sh
   ```

2. **Push the tag to trigger builds:**
   ```bash
   git push origin v1.0.0-beta.1
   ```

3. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Create a GitHub prerelease
   - Make it available for testers

### For Beta Testers

Install the latest beta:
```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/dev/install-beta.sh | bash
```

Or install a specific beta version:
```bash
CANDO_BETA_VERSION=v1.0.0-beta.2 curl -fsSL https://raw.githubusercontent.com/cutoken/cando/main/dev/install-beta.sh | bash
```

The beta installs as `cando-beta` to avoid conflicts with stable `cando`.

## How It Works

### Release Workflow

The `.github/workflows/release.yml` workflow:
1. Triggers on any `v*` tag
2. Detects beta/alpha/rc tags and marks them as prereleases
3. Builds and uploads binaries to GitHub Releases

### Beta Installer

The `dev/install-beta.sh` script:
1. Queries GitHub API for latest prerelease
2. Downloads the appropriate binary 
3. Installs as `cando-beta`
4. Never interferes with stable `cando` installation

### Version Naming

Beta versions follow semantic versioning with beta suffix:
- `v1.0.0-beta.1` - First beta for v1.0.0
- `v1.0.0-beta.2` - Second beta for v1.0.0
- `v1.0.0-rc.1` - Release candidate (also a prerelease)

## Advanced Usage

### Custom Hosting

For private beta testing without GitHub:
1. Build binaries locally: `make all`
2. Host them on your server
3. Install with: `CANDO_BASE_URL=https://your-server.com/path ./dev/install-beta.sh`

### Check Available Betas

View all prereleases:
```bash
curl -s https://api.github.com/repos/cutoken/cando/releases | \
  jq '.[] | select(.prerelease) | {tag: .tag_name, date: .created_at}'
```

### Uninstall Beta

```bash
rm ~/.local/bin/cando-beta
```

## Troubleshooting

### No beta releases found
- Check https://github.com/cutoken/cando/releases for prereleases
- Ensure the tag was pushed: `git push origin TAG_NAME`
- Wait for GitHub Actions to complete the build

### Installation fails
- Check network connectivity
- Verify the version exists in releases
- Try specifying a version explicitly
- Check GitHub API rate limits

## Development Notes

### Key Files
- `.github/workflows/release.yml` - Main release workflow (handles all tags)
- `dev/release-beta.sh` - Helper script to create beta tags
- `dev/install-beta.sh` - Beta installer script
- `install.sh` - Main installer (ignores prereleases)

### Testing Locally

Before pushing a beta tag:
```bash
# Test the build
make all

# Test the installer with local files
CANDO_BASE_URL=file://$(pwd)/dist ./dev/install-beta.sh
```

### Beta vs Stable

| Aspect | Stable Release | Beta Release |
|--------|---------------|--------------|
| Tag format | `v1.0.0` | `v1.0.0-beta.1` |
| GitHub Release | Regular release | Prerelease |
| Installer | `install.sh` | `dev/install-beta.sh` |
| Binary name | `cando` | `cando-beta` |
| API endpoint | `/releases/latest` | `/releases` (filtered) |

## Security Considerations

- Beta releases are public if the repository is public
- Use custom hosting for private betas
- Beta testers should be aware these are unstable builds
- Always test beta releases before wide distribution