# Secret Rotation Guide

## Overview

All secrets are stored as files in `deployments/docker/secrets/` and mounted via Docker Secrets at `/run/secrets/`. The application reads them at startup via the `_FILE` convention.

## Rotation Schedule

| Secret | Rotation Frequency | Impact of Rotation |
|--------|-------------------|-------------------|
| session_secret | Quarterly | Active sessions invalidated |
| postgres_password | Quarterly | Requires DB ALTER USER + restart |
| dodo_api_key | On compromise only | Payment processing interrupted briefly |
| dodo_webhook_secret | On compromise only | Webhooks fail until provider updated |
| metrics_token | Quarterly | Monitoring tools need new token |

## Procedures

### SESSION_SECRET

```bash
# 1. Generate new secret
openssl rand -hex 32 > deployments/docker/secrets/session_secret

# 2. Restart app (sessions will be invalidated)
docker compose restart app

# 3. Verify
docker compose exec -T app wget -qO- http://127.0.0.1:8080/health/ready
```

**Impact**: All active user sessions are invalidated. Users must re-authenticate.

### POSTGRES_PASSWORD

```bash
# 1. Generate new password
NEW_PASS=$(openssl rand -hex 24)

# 2. Update in PostgreSQL FIRST
docker compose exec postgres psql -U $POSTGRES_USER -c "ALTER USER $POSTGRES_USER PASSWORD '$NEW_PASS';"

# 3. Write new secret file
echo "$NEW_PASS" > deployments/docker/secrets/postgres_password

# 4. Update DATABASE_URL in .env with new password
# DATABASE_URL=postgres://user:NEW_PASS@postgres:5432/db?sslmode=require

# 5. Restart app
docker compose restart app

# 6. Verify
docker compose exec -T app wget -qO- http://127.0.0.1:8080/health/ready
```

**Impact**: Brief connection interruption during restart. No data loss.

### DODO_API_KEY / DODO_WEBHOOK_SECRET

```bash
# 1. Generate new key in Dodo Payments dashboard
# 2. Write new secret
echo "new-key-from-dashboard" > deployments/docker/secrets/dodo_api_key

# 3. Restart app
docker compose restart app
```

**Impact**: Payment operations fail between old key revocation and restart with new key. Coordinate with provider.

### METRICS_TOKEN

```bash
# 1. Generate new token
openssl rand -hex 16 > deployments/docker/secrets/metrics_token

# 2. Restart app
docker compose restart app

# 3. Update monitoring tools (Prometheus scrape config, etc.) with new token
```

### TRAEFIK_DASHBOARD_USERS

```bash
# 1. Generate new bcrypt hash
docker run --rm httpd:alpine htpasswd -nB admin

# 2. Update .env with new hash (escape $ as $$)
# TRAEFIK_DASHBOARD_USERS=admin:$$2y$$...

# 3. Restart traefik
docker compose restart traefik
```

## Emergency Rotation (Suspected Compromise)

```bash
# Rotate all secrets immediately
openssl rand -hex 32 > deployments/docker/secrets/session_secret
openssl rand -hex 24 > deployments/docker/secrets/postgres_password
openssl rand -hex 16 > deployments/docker/secrets/metrics_token

# Update DB password
docker compose exec postgres psql -U $POSTGRES_USER -c "ALTER USER $POSTGRES_USER PASSWORD '$(cat deployments/docker/secrets/postgres_password)';"

# Update DATABASE_URL in .env
# Restart everything
docker compose down && docker compose up -d

# Revoke old Dodo keys in provider dashboard, create new ones
# Update monitoring systems with new METRICS_TOKEN
```

## Verification After Rotation

```bash
# Health check
curl -s https://DOMAIN/health/ready

# API access works
curl -s https://DOMAIN/api/v1/health

# Metrics endpoint accepts new token
curl -sH "Authorization: Bearer NEW_TOKEN" https://DOMAIN/metrics | head -5

# Dashboard accessible
curl -su admin:newpassword https://traefik.DOMAIN/dashboard/
```
