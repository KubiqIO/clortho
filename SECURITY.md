# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |

## Reporting a Vulnerability

We take the security of Clortho seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of these methods:

1. **GitHub Security Advisories** (Preferred): Use [GitHub's private vulnerability reporting](https://github.com/KubiqIO/clortho/security/advisories/new)

2. **Email**: Contact the maintainers directly at security@keyva.dev

### What to Include

Please include the following information to help us triage your report:

- Type of vulnerability (e.g., authentication bypass, SQL injection, XSS)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue and how it might be exploited

### Response Timeline

- **Initial Response**: Within 48 hours of your report
- **Status Update**: Within 7 days with our assessment
- **Resolution Timeline**: Depends on severity and complexity

### Disclosure Policy

- We will work with you to understand and resolve the issue quickly
- We will keep you informed of our progress
- We ask that you give us reasonable time to address the issue before public disclosure
- We will credit you in our security advisories (unless you prefer to remain anonymous)

## Security Best Practices for Operators

When deploying Clortho:

1. **Use strong secrets**: Generate a secure `admin_secret` (minimum 32 characters, random)
2. **Enable TLS**: Always run behind a TLS-terminating reverse proxy in production
3. **Database security**: Use strong PostgreSQL passwords and restrict network access
4. **Rate limiting**: Keep rate limiting enabled to prevent brute-force attacks
5. **Environment variables**: Use environment variables or secrets management for sensitive configuration
