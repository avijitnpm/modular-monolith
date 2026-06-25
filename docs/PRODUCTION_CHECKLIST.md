# Production Deployment Checklist

## Prerequisites

- [ ] Server with Docker and Docker Compose installed
- [ ] Domain name pointing to server IP
- [ ] Port 80 and 443 open (for ACME HTTP challenge)
- [ ] Repository cloned to server

## First Deployment

### 1. Create secrets directory

```bash
cd deployments/docker
mkdir -p secrets
chmod 700 secrets
```

### 2. Generate secrets

```bash
openssl rand -hex 32 > secrets/session_secret
openssl rand -hex 24 > secrets/postgres_password
# Copy from your payment provider dashboard:
echo "your-dodo-api-key" > secrets/dodo_api_key
echo "your-dodo-webhook-secret" > secrets/dodo_webhook_secret
openssl rand -hex 16 > secrets/metrics_token
chmod 600 secrets/*
```

### 3. Configure environment

```bash
cp .env.example .env
```

Fill in:
- [ ] `DOMAIN` — your production domain (e.g., `app.example.com`)
- [ ] `ACME_EMAIL` — email for Let's Encrypt notifications
- [ ] `POSTGRES_DB` — database name
- [ ] `POSTGRES_USER` — database user
- [ ] `DATABASE_URL` — full connection string with sslmode=require
- [ ] `OIDC_ISSUER` — your identity provider URL
- [ ] `OIDC_AUDIENCE` — OIDC audience
- [ ] `OIDC_CLIENT_ID` — OIDC client ID
- [ ] `OIDC_REDIRECT_URL` — `https://DOMAIN/api/v1/auth/callback`
- [ ] `CORS_ORIGIN` — `https://DOMAIN`
- [ ] `DODO_BASE_URL` — `https://api.dodopayments.com`
- [ ] `OTEL_EXPORTER_OTLP_ENDPOINT` — your OTLP endpoint
- [ ] `TRAEFIK_DASHBOARD_USERS` — generate with `docker run --rm httpd:alpine htpasswd -nB admin`

### 4. Deploy

```bash
docker compose up -d
```

### 5. Verify

- [ ] `curl -s https://DOMAIN/health/ready` returns `{"status":"ready"}`
- [ ] `curl -s https://DOMAIN/` returns the frontend
- [ ] `curl -sI https://DOMAIN/` shows security headers (HSTS, X-Frame-Options)
- [ ] Certificate is valid: `echo | openssl s_client -connect DOMAIN:443 2>/dev/null | openssl x509 -noout -dates`

## Periodic Maintenance

### Weekly

- [ ] Verify backups exist: `ls -la backups/daily/`
- [ ] Check container health: `docker compose ps`
- [ ] Review logs for errors: `docker compose logs --since=7d | grep -i error`

### Monthly

- [ ] Review resource usage: `docker stats --no-stream`
- [ ] Check certificate expiry (should auto-renew)
- [ ] Update images if security patches available
- [ ] Test restore procedure with latest backup

### Quarterly

- [ ] Rotate secrets (see SECRET_ROTATION.md)
- [ ] Review access logs for anomalies
- [ ] Update Traefik/Postgres/OpenObserve to latest patch versions
- [ ] Run `docker compose build` with updated base images
