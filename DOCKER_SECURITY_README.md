# 🚀 Gotify Docker Deployment with Security Enhancements

This directory contains Docker configuration with comprehensive security enhancements for Gotify server.

## 📦 What's Included

| File | Description |
|------|-------------|
| `docker-compose.yml` | Production-ready Docker Compose configuration |
| `docker/Dockerfile` | Multi-stage build (amd64 only, no external build image) |
| `docker-start.sh` | Quick start script with safety checks |
| `docker-compose.env.example` | Environment configuration template |
| `SECURITY_CHECKLIST.md` | Post-deployment security checklist |

## 🔧 Dockerfile Features

The optimized Dockerfile provides:

- **amd64 Only**: Simplified build, no cross-platform complexity
- **Multi-stage Build**: Minimal final image (~100MB)
- **No External Build Image**: Uses official Go/Alpine images
- **Native SQLite/MySQL/Postgres**: All database drivers included
- **Non-root User**: Runs as `gotify` user (UID 1000)
- **Security Headers**: Built-in via Go code
- **Rate Limiting**: Built-in via Go code

## 🔒 Security Features

All security enhancements are **enabled by default**:

### 1. Rate Limiting
- **5 requests per second** per IP address
- **10 burst capacity** for concurrent requests
- Returns HTTP 429 when exceeded
- Prevents brute force attacks

### 2. Mandatory Password Change
- New installations require password change on first login
- Default credentials: `admin` / `admin`
- API returns `mustChangePassword` flag

### 3. Security Headers
- `Content-Security-Policy` - Restricts resource loading
- `X-Content-Type-Options: nosniff` - Prevents MIME sniffing
- `X-Frame-Options: DENY` - Prevents clickjacking
- `X-XSS-Protection: 1; mode=block` - Legacy browser protection
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Strict-Transport-Security` - Enforces HTTPS

### 4. XSS Protection
- All Markdown content is sanitized via `rehype-sanitize`
- Prevents script injection through messages

## 🚀 Quick Start

### 1. Clone and Setup
```bash
git clone https://github.com/gotify/server.git
cd server
```

### 2. Build the Docker Image
```bash
# Build the image (amd64 only, ~2-3 minutes first time)
docker-compose build

# Or use the quick start script (builds automatically)
./docker-start.sh --build
```

### 3. Configure Environment
```bash
# Copy environment template
cp docker-compose.env.example .env

# Edit with your settings
nano .env
```

**Important**: Change `GOTIFY_DEFAULTUSER_PASS` to a strong password!

### 4. Start Gotify
```bash
# Using the quick start script (recommended)
./docker-start.sh

# Or directly with docker-compose
docker-compose up -d
```

### 4. Access Gotify
- Open: http://localhost:80
- Login: `admin` / `admin` (change immediately!)
- First login will require password change

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOTIFY_DEFAULTUSER_NAME` | admin | Default admin username |
| `GOTIFY_DEFAULTUSER_PASS` | admin | ⚠️ Change this! |
| `GOTIFY_PASSSTRENGTH` | 12 | Bcrypt cost factor (10-15) |
| `GOTIFY_SERVER_RATELIMIT_REQUESTSPERSECOND` | 5 | Rate limit RPS |
| `GOTIFY_SERVER_RATELIMIT_BURST` | 10 | Rate limit burst |
| `GOTIFY_REGISTRATION` | false | Enable user registration |
| `GOTIFY_DATABASE_DIALECT` | sqlite3 | Database type |

### Data Persistence

By default, data is stored in `./data`:
```
./data/
├── gotify.db      # SQLite database
└── images/        # Uploaded images
```

To change the data directory:
```bash
export GOTIFY_DATA_PATH=/path/to/data
./docker-start.sh
```

## 🔐 Production Deployment

### Using External Database

Edit `docker-compose.yml` and uncomment the PostgreSQL section:

```yaml
services:
  gotify:
    # ... existing config ...
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    # ... rest of postgres config ...
```

### SSL/TLS

Option 1: Let's Encrypt (Automatic)
```yaml
environment:
  GOTIFY_SERVER_SSL_ENABLED: "true"
  GOTIFY_SERVER_SSL_LETSENCRYPT_ENABLED: "true"
  GOTIFY_SERVER_SSL_LETSENCRYPT_ACCEPTTOS: "true"
  GOTIFY_SERVER_SSL_LETSENCRYPT_HOSTS: "gotify.example.com"
```

Option 2: Custom Certificates
```yaml
volumes:
  - ./ssl/cert.pem:/gotify/ssl/cert.pem:ro
  - ./ssl/key.pem:/gotify/ssl/key.pem:ro

environment:
  GOTIFY_SERVER_SSL_ENABLED: "true"
  GOTIFY_SERVER_SSL_CERTFILE: "/gotify/ssl/cert.pem"
  GOTIFY_SERVER_SSL_CERTKEY: "/gotify/ssl/key.pem"
```

### Reverse Proxy (Recommended)

Place Gotify behind Traefik, Nginx, or Caddy:

```yaml
# Example Nginx config snippet
location / {
    proxy_pass http://gotify:80;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_cache_bypass $http_upgrade;
}
```

## 📊 Monitoring

### Health Check
```bash
curl http://localhost:80/health
```

### View Logs
```bash
docker-compose logs -f gotify
```

### Check Security Headers
```bash
curl -I http://localhost:80/version
```

## 🛡️ Security Best Practices

1. **Change default password immediately** - Most critical!
2. **Use HTTPS in production** - Enable SSL
3. **Use external database** - SQLite for development only
4. **Configure CORS** - Restrict allowed origins
5. **Set strong password strength** - Use 12 or higher
6. **Disable registration** - If not needed
7. **Use secrets** - Don't commit passwords to git
8. **Regular backups** - Backup `./data` directory
9. **Keep updated** - Watch for security releases
10. **Monitor logs** - Watch for suspicious activity

## 🆘 Troubleshooting

### Container won't start
```bash
# Check logs
docker-compose logs gotify

# Verify permissions
chmod 755 ./data
```

### Rate limited too aggressively
Edit `.env`:
```bash
GOTIFY_SERVER_RATELIMIT_REQUESTSPERSECOND=10
GOTIFY_SERVER_RATELIMIT_BURST=20
```

### Can't login
Reset the admin password:
```bash
docker-compose exec gotify gotify-app user password admin NEW_PASSWORD
```

## 📚 Resources

- [Gotify Documentation](https://gotify.net/docs)
- [Docker Hub Image](https://hub.docker.com/r/gotify/server)
- [GitHub Repository](https://github.com/gotify/server)
- [Security Issues](https://github.com/gotify/server/security)

---

**Remember**: Security is an ongoing process. Regularly review and update your security configuration!
