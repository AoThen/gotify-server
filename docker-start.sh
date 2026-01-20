#!/bin/bash
# Gotify Docker Quick Start Script
# 
# This script sets up and starts Gotify with security enhancements:
# - Rate limiting enabled (5 req/s, burst 10)
# - Mandatory password change for default user
# - Security headers (CSP, X-Frame-Options, etc.)
# - XSS protection via markdown sanitization
#
# Usage: ./docker-start.sh [--build] [--reset]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Gotify with Security Enhancements...${NC}"
echo ""

# Parse arguments
BUILD=false
RESET=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD=true
            shift
            ;;
        --reset)
            RESET=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check for .env file
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  No .env file found. Creating from template...${NC}"
    if [ -f "docker-compose.env.example" ]; then
        cp docker-compose.env.example .env
        echo -e "${YELLOW}   Please edit .env and set your password!${NC}"
        echo ""
    fi
fi

# Check password
if [ -f ".env" ]; then
    if grep -q "GOTIFY_DEFAULTUSER_PASS=your_secure_password" .env 2>/dev/null; then
        echo -e "${RED}⚠️  WARNING: Default password not changed!${NC}"
        echo -e "${YELLOW}   Edit .env and set a strong password!${NC}"
        echo ""
    fi
fi

# Create data directory
if [ ! -d "./data" ]; then
    echo -e "${GREEN}📁 Creating data directory...${NC}"
    mkdir -p ./data
fi

# Reset option
if [ "$RESET" = true ]; then
    echo -e "${YELLOW}⚠️  Resetting Gotify (this will delete all data!)...${NC}"
    read -p "Are you sure? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker-compose down -v
        rm -rf ./data/*
        echo -e "${GREEN}✅ Reset complete${NC}"
    else
        echo "Aborted."
        exit 0
    fi
fi

# Build option
if [ "$BUILD" = true ]; then
    echo -e "${GREEN}🔨 Building Gotify...${NC}"
    docker-compose build --no-cache
fi

# Start Gotify
echo -e "${GREEN}▶️  Starting Gotify...${NC}"
docker-compose up -d

# Wait for health check
echo -e "${GREEN}⏳ Waiting for Gotify to be healthy...${NC}"
sleep 5

# Check health
if curl -sf http://localhost:80/health > /dev/null 2>&1; then
    echo ""
    echo -e "${GREEN}✅ Gotify is running!${NC}"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${GREEN}🎉 Gotify is now available at:${NC}"
    echo "   http://localhost:80"
    echo ""
    echo -e "${YELLOW}⚠️  IMPORTANT SECURITY NOTES:${NC}"
    echo "   1. Log in with: admin / admin"
    echo "   2. Change the default password immediately!"
    echo "   3. Review CORS settings for your needs"
    echo "   4. Consider using HTTPS in production"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
else
    echo -e "${RED}❌ Gotify failed to start. Check logs with:${NC}"
    echo "   docker-compose logs gotify"
fi
