# Development & Beta Testing

This folder contains development tools and beta testing infrastructure.

## Structure

```
dev/
├── README.md           # This file
├── install-beta.sh     # Beta version installer
├── release-beta.sh     # Beta release script
├── beta.yml            # Beta release workflow (copy to .github/workflows/ for beta branch)
└── test-install.sh     # Local installation tester
```

## Beta Testing Workflow

### For Developers

1. **Create beta branch** (one time):
```bash
git checkout -b beta
git push origin beta
```

2. **Release a beta version**:
```bash
git checkout beta
git merge develop  # or cherry-pick specific commits
./dev/release-beta.sh v1.0.0-beta.1
```

3. **Share with testers**:
```bash
# Direct them to:
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/beta/dev/install-beta.sh | bash

# Or for custom hosting:
CANDO_BASE_URL="https://your-server.com/beta" bash <(curl -fsSL your-installer-url)
```

### For Beta Testers

Install beta version:
```bash
curl -fsSL https://raw.githubusercontent.com/cutoken/cando/beta/dev/install-beta.sh | bash
```

This installs `cando-beta` binary alongside regular `cando` (if installed).

### Testing Without GitHub

1. **Build locally**:
```bash
make all
```

2. **Host locally**:
```bash
cd dist && python3 -m http.server 8080
```

3. **Install from local server**:
```bash
CANDO_BASE_URL="http://localhost:8080" ./dev/install-beta.sh
```

### Private Beta Testing

For private beta testing with select users:

1. **Option A: Private server**
   - Upload binaries to your server
   - Share installer with `CANDO_BASE_URL` set

2. **Option B: Direct binary sharing**
   - Build binaries
   - Share via secure file transfer
   - Users run: `./cando-beta --version`

3. **Option C: Fork to private repo**
   - Fork to private GitHub/GitLab
   - Push beta branch there
   - Share installer with `CANDO_REPO_OWNER` set

## Version Management

Beta versions use semantic versioning with beta suffix:
- Stable: `v1.0.0`
- Beta: `v1.0.0-beta.1`, `v1.0.0-beta.2`, etc.
- RC: `v1.0.0-rc.1`

## Notes

- Beta binaries are named `cando-beta` to avoid conflicts
- Beta uses same config as stable (`~/.cando/`)
- Users can run both stable and beta side by side
- Beta telemetry (if enabled) is tagged with beta version