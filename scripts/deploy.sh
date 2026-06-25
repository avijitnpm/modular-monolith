#!/bin/bash
set -euo pipefail

# Deployment Script
# Usage: ./scripts/deploy.sh
# Validates environment, builds, deploys, and verifies health.

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
ok()   { echo -e "${GREEN}[ OK ]${NC} $1"; }

echo "=== Modular Monolith Deployment ==="
echo ""

# Step 1: Prerequisites
echo "--- Checking prerequisites ---"
command -v docker >/dev/null 2>&1 || fail "docker not found"
ok "docker"
docker compose version >/dev/null 2>&1 || fail "docker compose not found"
ok "docker compose"
command -v go >/dev/null 2>&1 || fail "go not found"
ok "go"

# Step 2: Environment
echo ""
echo "--- Validating environment ---"
if [ ! -f .env ]; then
  fail ".env file not found. Copy .env.example and configure."
fi
ok ".env exists"

# Check required vars for production
if grep -q "APP_ENV=production" .env 2>/dev/null; then
  for var in DOMAIN ACME_EMAIL SESSION_SECRET POSTGRES_PASSWORD; do
    val=$(grep "^${var}=" .env | cut -d= -f2-)
    if [ -z "${val}" ] || [ "${val}" = "change-me-to-at-least-32-random-characters" ] || [ "${val}" = "postgres" ]; then
      fail "Production requires a proper ${var} value"
    fi
  done
  ok "Production environment validated"
else
  ok "Development environment (skipping strict validation)"
fi

# Step 3: Build
echo ""
echo "--- Building ---"
go build ./... || fail "Go build failed"
ok "go build"

docker compose build --quiet || fail "Docker build failed"
ok "docker compose build"

# Step 4: Deploy
echo ""
echo "--- Deploying ---"
docker compose up -d || fail "docker compose up failed"
ok "Containers started"

# Step 5: Wait for health
echo ""
echo "--- Waiting for health (60s timeout) ---"
TIMEOUT=60
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
  if docker compose exec -T app wget -qO- http://127.0.0.1:8080/health/ready 2>/dev/null | grep -q "ready"; then
    ok "Application healthy after ${ELAPSED}s"
    break
  fi
  sleep 2
  ELAPSED=$((ELAPSED + 2))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
  fail "Health check timed out after ${TIMEOUT}s. Check logs: docker compose logs app"
fi

# Step 6: Verify services
echo ""
echo "--- Verifying services ---"
for svc in app postgres openobserve traefik; do
  STATUS=$(docker compose ps --format '{{.Status}}' "${svc}" 2>/dev/null | head -1)
  if echo "${STATUS}" | grep -qi "up\|running"; then
    ok "${svc}: ${STATUS}"
  else
    fail "${svc}: ${STATUS:-not running}"
  fi
done

echo ""
echo "=== Deployment complete ==="
