# Gotify Server Dockerfile with Security Enhancements
# Optimized for amd64 only with simplified build process

# ============================================================
# Build Arguments
# ============================================================
ARG GO_VERSION=1.25
ARG ALPINE_VERSION=3.20

# ============================================================
# Stage 1: Build JavaScript/TypeScript UI
# ============================================================
FROM --platform=amd64 node:24-alpine AS js-builder

WORKDIR /src/gotify/ui

# Copy UI source
COPY ./ui/package*.json ./
COPY ./ui/src ./src

# Install dependencies and build
RUN npm install && \
    npm run build

# ============================================================
# Stage 2: Build Go Application
# ============================================================
FROM --platform=amd64 docker.io/library/golang:${GO_VERSION}-alpine AS builder

WORKDIR /src/gotify

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    musl-dev

# Copy Go modules
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built UI
COPY --from=js-builder /src/gotify/ui/build ./ui/build

# Build the application
# -ldflags: Strip debug info for smaller binary
# -tags: Include sqlite3, mysql, postgres drivers
# CGO_ENABLED=1: Alpine uses musl libc, needs CGO for SQLite/MySQL/Postgres
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /gotify-server \
    -tags 'sqlite3 mysql postgres' \
    .

# ============================================================
# Stage 3: Runtime Image (Minimal Alpine)
# ============================================================
FROM alpine:${ALPINE_VERSION}

WORKDIR /gotify

# Install runtime dependencies (Alpine uses musl libc)
# Note: All drivers must match what was compiled with CGO_ENABLED=1
RUN apk add --no-cache \
    ca-certificates \
    curl \
    tzdata \
    sqlite \
    mariadb-client \
    libpq

# Create non-root user
RUN addgroup -g 1000 gotify && \
    adduser -u 1000 -G gotify -s /bin/sh -D gotify && \
    mkdir -p /gotify/data /gotify/images /gotify/plugins && \
    chown -R gotify:gotify /gotify

USER gotify

# Health check (use wget instead of curl for smaller size)
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s \
    CMD wget --spider -q http://localhost:80/health || exit 1

# Expose port
EXPOSE 80

# Set environment
ENV GOTIFY_SERVER_PORT=80

# Entry point
ENTRYPOINT ["/gotify/gotify"]
