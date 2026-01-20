# 🔒 Gotify Security Checklist

This document provides a security checklist for your Gotify deployment, including the security enhancements we've implemented.

## ✅ Implemented Security Enhancements

### 1. XSS Protection (Critical)
**Status: ✅ Implemented**
- All Markdown content is sanitized using `rehype-sanitize`
- Prevents script injection via message content
- Verified in: `ui/src/common/Markdown.tsx`

### 2. Default Password Policy (High)
**Status: ✅ Implemented**
- New installations require password change on first login
- `MustChangePassword` flag added to user model
- API returns `mustChangePassword` field for UI enforcement
- Verified in: `model/user.go`, `api/user.go`

### 3. Security Response Headers (High)
**Status: ✅ Implemented**
All responses include:
- `Content-Security-Policy`: Restricts resource loading
- `X-Content-Type-Options: nosniff`: Prevents MIME sniffing
- `X-Frame-Options: DENY`: Prevents clickjacking
- `X-XSS-Protection: 1; mode=block`: Legacy browser XSS filter
- `Referrer-Policy: strict-origin-when-cross-origin`: Controls referrer info
- `Strict-Transport-Security`: Enforces HTTPS (when SSL enabled)
- Verified in: `router/router.go`

### 4. Brute Force Protection (Medium)
**Status: ✅ Implemented**
- Rate limiting: 5 requests per second per IP
- Burst capacity: 10 concurrent requests
- Returns HTTP 429 when limit exceeded
- Verified in: `auth/ratelimit.go`, `router/router.go`

---

## 📋 Pre-Deployment Checklist

### Environment Hardening

- [ ] Change default admin password immediately after first login
- [ ] Set `GOTIFY_PASSSTRENGTH` to at least 12
- [ ] Enable HTTPS in production (set `GOTIFY_SERVER_SSL_ENABLED=true`)
- [ ] Configure CORS allowed origins for your domain
- [ ] Set `GOTIFY_REGISTRATION=false` if not needed
- [ ] Configure trusted proxies if behind a reverse proxy

### Docker Security

- [ ] Run container as non-root user (`user: 1000:1000`)
- [ ] Use read-only filesystem where possible
- [ ] Drop all Linux capabilities except needed ones
- [ ] Set resource limits (CPUs, memory)
- [ ] Configure log rotation
- [ ] Use secrets for sensitive environment variables

### Database Security

- [ ] For production, use PostgreSQL or MySQL instead of SQLite
- [ ] Use strong database passwords
- [ ] Enable SSL for database connections
- [ ] Regularly backup database

### Network Security

- [ ] Place Gotify behind a reverse proxy (Traefik, Nginx, Caddy)
- [ ] Configure firewall rules
- [ ] Use HTTPS only (redirect HTTP to HTTPS)
- [ ] Consider using a WAF for additional protection

---

## 🔧 Configuration Examples

### Rate Limiting

```yaml
# docker-compose.yml
environment:
  GOTIFY_SERVER_RATELIMIT_ENABLED: "true"
  GOTIFY_SERVER_RATELIMIT_REQUESTSPERSECOND: "5"
  GOTIFY_SERVER_RATELIMIT_BURST: "10"
```

### SSL/TLS with Let's Encrypt

```yaml
environment:
  GOTIFY_SERVER_SSL_ENABLED: "true"
  GOTIFY_SERVER_SSL_LETSENCRYPT_ENABLED: "true"
  GOTIFY_SERVER_SSL_LETSENCRYPT_ACCEPTTOS: "true"
  GOTIFY_SERVER_SSL_LETSENCRYPT_HOSTS: "gotify.example.com"
```

### Custom Security Headers

```yaml
environment:
  GOTIFY_SERVER_RESPONSEHEADERS_CSP: "default-src 'self'; script-src 'self' https://trusted.cdn.com"
```

---

## 🧪 Testing Security

### 1. Test XSS Protection
```bash
# Send a message with XSS payload
curl -X POST http://localhost:80/message \
  -H "X-Gotify-Key: YOUR_APP_TOKEN" \
  -d '{"message": "<script>alert(1)</script>", "title": "Test"}'

# Verify the script is escaped in the UI
```

### 2. Test Rate Limiting
```bash
# Make rapid requests
for i in {1..15}; do
  curl -I http://localhost:80/version
done

# Should see HTTP 429 after 10 requests
```

### 3. Verify Security Headers
```bash
curl -I http://localhost:80/version
```

Expected headers:
- `Content-Security-Policy`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`

### 4. Verify Password Policy
```bash
# Login and check response includes mustChangePassword
curl -u admin:admin http://localhost:80/current/user
```

Response should include:
```json
{
  "id": 1,
  "name": "admin",
  "admin": true,
  "mustChangePassword": true
}
```

---

## 📊 Monitoring

### Log Analysis

Monitor logs for security events:
```bash
# View logs
docker-compose logs -f gotify

# Search for rate limit events
docker-compose logs gotify | grep "Rate limit"

# Search for authentication failures
docker-compose logs gotify | grep "unauthorized"
```

### Metrics to Watch

- Rate limit violations (HTTP 429 responses)
- Authentication failures
- Invalid user attempts
- Large message volumes (potential spam)

---

## 🚨 Incident Response

If you suspect a security incident:

1. **Immediate Actions**
   - Block the source IP at firewall level
   - Rotate all API keys and tokens
   - Force password reset for all users

2. **Investigation**
   - Review access logs
   - Check for unauthorized messages
   - Audit user accounts

3. **Recovery**
   - Apply security patches
   - Review and tighten CORS settings
   - Enable additional logging

---

## 📚 References

- [Gotify Documentation](https://gotify.net/docs)
- [OWASP Security Headers](https://owasp.org/www-project-secure-headers/)
- [Rate Limiting Best Practices](https://cloud.google.com/armor/docs/rate-limiting-overview)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)

---

Last Updated: 2024-01-20
