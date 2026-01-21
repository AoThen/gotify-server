#!/bin/bash
# Gotify Docker Quick Start Script
# 
# This script sets up and starts Gotify with tiered rate limiting (Option A):
# - Level 1: Global 20 req/s, burst 50
# - Level 2: Auth API 10 req/s, burst 20
# - Level 3: Message sending 15 req/s, burst 30
# - Level 4: Admin API 5 req/s, burst 10
#
# Usage: ./docker-start.sh [--build] [--reset]

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}🚀 Starting Gotify with Tiered Rate Limiting...${NC}"
echo ""

BUILD=false
RESET=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --build) BUILD=true; shift ;;
        --reset) RESET=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  No .env file found. Creating from template...${NC}"
    if [ -f "docker-compose.env.example" ]; then
        cp docker-compose.env.example .env
    fi
fi

if [ "$RESET" = true ]; then
    echo -e "${YELLOW}⚠️  Resetting Gotify...${NC}"
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

if [ "$BUILD" = true ]; then
    echo -e "${GREEN}🔨 Building Gotify...${NC}"
    docker-compose build
fi

echo -e "${GREEN}▶️  Starting Gotify...${NC}"
docker-compose up -d

echo -e "${GREEN}⏳ Waiting for Gotify to be healthy...${NC}"
sleep 5

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
    echo "   3. Default rate limiting tiers:"
    echo "      - Global: 20 req/s (burst 50)"
    echo "      - Auth API: 10 req/s (burst 20)"
    echo "      - Message: 15 req/s (burst 30)"
    echo "      - Admin: 5 req/s (burst 10)"
    echo "   4. Adjust tiers in docker-compose.yml environment variables"
    echo "   5. Consider using HTTPS in production"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
else
    echo -e "${RED}❌ Gotify failed to start. Check logs with:${NC}"
    echo "   docker-compose logs gotify"
fi
