# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Security Considerations

Cando is a local AI coding agent that executes tools within a sandboxed workspace. While we implement security boundaries, users should be aware of the following:

### Sandbox Boundaries

- All file operations are restricted to the configured `workspace_root`
- Shell commands execute within the workspace context
- The web UI binds to `127.0.0.1` (localhost only) by default

### API Keys

- API keys are stored in `~/.cando/credentials.yaml`
- Keys are never logged or transmitted except to the configured LLM provider
- Use environment variables (`CANDO_CREDENTIALS_PATH`) for CI/testing environments

### Tool Execution

- Shell commands have configurable timeouts
- File operations have byte limits
- Background processes are tracked and can be terminated

## Reporting a Vulnerability

If you discover a security vulnerability in Cando, please report it responsibly:

1. **Do NOT** open a public GitHub issue for security vulnerabilities
2. Instead, report vulnerabilities via GitHub's private vulnerability reporting:
   - Go to the repository's Security tab
   - Click "Report a vulnerability"
   - Provide detailed information about the issue

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- We will acknowledge receipt within 48 hours
- We aim to provide an initial assessment within 7 days
- Critical vulnerabilities will be prioritized for immediate patches

## Security Best Practices for Users

1. **Keep Cando updated** to the latest version
2. **Review workspace contents** before running Cando on untrusted directories
3. **Protect your API keys** - don't commit credentials to version control
4. **Use dedicated workspaces** for sensitive projects
5. **Monitor background processes** started by Cando

## Scope

This security policy covers the Cando application itself. Third-party LLM providers (Z.AI, OpenRouter) have their own security policies and terms of service.
