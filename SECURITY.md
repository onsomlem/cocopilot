# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Cocopilot, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email the maintainers directly or use GitHub's private vulnerability reporting feature.

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if you have one)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Assessment**: Within 1 week
- **Fix release**: As soon as practical, depending on severity

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest (main branch) | Yes |

Cocopilot does not currently use tagged releases. The latest commit on `main`
is the supported version.

## Security Best Practices

See [docs/security.md](docs/security.md) for deployment security guidance including:
- Enabling API key authentication
- Reverse proxy configuration with TLS
- Network binding restrictions
- Database file permissions
