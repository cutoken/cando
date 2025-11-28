# Beta Testing Setup Guide

## Quick Start

### 1. Initial Setup (One Time)

```bash
# Create beta branch
git checkout -b beta
git push origin beta

# Copy workflow to beta branch
git checkout beta
cp dev/beta.yml .github/workflows/
git add .github/workflows/beta.yml
git commit -m "Add beta workflow"
git push origin beta
```

### 2. Release Beta Version

**Option A: Manual (Full Control)**
```bash
git checkout beta
git merge main  # or cherry-pick specific commits
./dev/release-beta.sh v1.0.0-beta.1
```

**Option B: GitHub Actions (Automated)**
```bash
git checkout beta
git merge main
git push origin beta  # Triggers automatic build
```

### 3. Share with Testers

Send them this one-liner:
```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/beta/dev/install-beta.sh | bash
```

## Testing Scenarios

### Local Testing (No GitHub)
```bash
# Build and test locally
make all
./dev/test-install.sh
```

### Private Server Testing
```bash
# Build
make all

# Upload to your server
scp dist/cando-* user@server:/var/www/beta/

# Testers install from your server
CANDO_BASE_URL="https://your-server.com/beta" \
  curl -fsSL https://your-server.com/install-beta.sh | bash
```

### Private Repository Testing
```bash
# Fork to private repo
git remote add private git@github.com:private-org/cando.git
git push private beta

# Testers install from private repo
CANDO_REPO_OWNER="private-org" \
  curl -fsSL https://raw.githubusercontent.com/private-org/cando/beta/dev/install-beta.sh | bash
```

## Key Features

✅ **Isolation**: Beta installs as `cando-beta`, doesn't conflict with stable  
✅ **Flexibility**: Works with GitHub, private servers, or local testing  
✅ **Control**: You decide when/how to release betas  
✅ **Simple**: One-line install for testers  
✅ **Clean**: All beta stuff in `dev/` folder, main branch stays clean  

## Beta vs Stable

| Aspect | Stable | Beta |
|--------|--------|------|
| Branch | `main` | `beta` |
| Binary | `cando` | `cando-beta` |
| Installer | `install.sh` | `dev/install-beta.sh` |
| Version | `v1.0.0` | `v1.0.0-beta.1` |
| Users | Everyone | Testers only |

## Workflow Comparison

### Without GitHub Dependency
1. Build locally: `make all`
2. Host anywhere (S3, VPS, CDN)
3. Share installer with `CANDO_BASE_URL`

### With GitHub (Simple)
1. Push to beta branch
2. Share installer URL
3. Beta branch has binaries committed

### With GitHub Actions (Automated)
1. Push to beta branch
2. Actions builds automatically
3. Commits binaries back to beta branch

Choose based on your needs!