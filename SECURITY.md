# Security Policy

## Supported Versions

We support the latest version of Gotify server. Security updates are released regularly.

## Current Security Features

### Rate Limiting (Tiered Protection)
Gotify implements a 4-tier rate limiting system:
- **Level 1 (Global)**: 20 requests/second (burst 50) - All requests
- **Level 2 (Auth API)**: 10 requests/second (burst 20) - Authenticated clients and admin APIs
- **Level 3 (Message Sending)**: 15 requests/second (burst 30) - Application token message creation
- **Level 4 (Admin API)**: 5 requests/second (burst 10) - Administrative operations

### Authentication Blacklist
- Automatic IP blocking after 5 failed authentication attempts within 5 minutes
- Blocked IPs are automatically unblocked after 1 hour
- Whitelist support for trusted IPs (supports CIDR notation)
- In-memory storage (cleared on restart)

### Additional Security Measures
- **XSS Protection**: Markdown content is sanitized via rehype-sanitize
- **Security Headers**: CSP, X-Frame-Options, X-Content-Type-Options, etc.
- **Mandatory Password Change**: Default admin users must change password on first login
- **Enforce Password Policy**: Configurable bcrypt strength (default: 10)

## Reporting a Vulnerability

Please report (suspected) security vulnerabilities to
**[gotify@protonmail.com](mailto:gotify@protonmail.com)**. You will receive a
response from us within a few days. If the issue is confirmed, we will release a
patch as soon as possible.
