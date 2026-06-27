# Phase 18C — Operations, Backup & CI/CD Hardening

## 1. Operations Audit Matrix

| Area | Previous Status | Current Status |
|------|----------------|----------------|
| PostgreSQL backups | None | **Fixed** — scripts/backup.sh (daily/weekly/monthly) |
| Restore capability | None | **Fixed** — scripts/restore.sh |
| Backup retention | None | **Fixed** — Configurable via env vars |
| CI/CD pipeline | None | **Fixed** — .github/workflows/ci.yml |
| Release workflow | None | **Fixed** — .github/workflows/release.yml |
| Deployment automation | None | **Fixed** — scripts/deploy.sh |
| Monitoring documentation | None | **Fixed** — Below |
| Runbooks | None | **Fixed** — Below |
| Disaster recovery | None | **Fixed** — Below |

---

## 2. Backup Architecture

```
┌─────────────────────────────────────────────────┐
│              Backup System                        │
│                                                   │
│  scripts/backup.sh [daily|weekly|monthly]        │
│       │                                           │
│       ▼                                           │
│  pg_dump → gzip → /backups/{type}/               │
│                                                   │
│  Retention (configurable via env):                │
│    BACKUP_RETAIN_DAILY=7                         │
│    BACKUP_RETAIN_WEEKLY=4                        │
│    BACKUP_RETAIN_MONTHLY=6                       │
│                                                   │
│  Schedule (add to host crontab):                  │
│    0 2 * * *   backup.sh daily                   │
│    0 3 * * 0   backup.sh weekly                  │
│    0 4 1 * *   backup.sh monthly                 │
│                                                   │
│  Restore:                                         │
│    scripts/restore.sh <file.sql.gz>              │
└─────────────────────────────────────────────────┘
```

### Crontab Setup

```bash
# Run from inside the postgres container or from host with docker exec:
# Host crontab example:
0 2 * * *   docker compose exec -T postgres /scripts/backup.sh daily
0 3 * * 0   docker compose exec -T postgres /scripts/backup.sh weekly
0 4 1 * *   docker compose exec -T postgres /scripts/backup.sh monthly
```

---

## 3. Disaster Recovery Flow

### Recovery Sequence: Empty Server → Healthy Deployment

```
1. Provision server (Ubuntu 22.04+ or similar)
     │
2. Install Docker + Docker Compose
     │
3. Clone repository
     │
4. Create .env from .env.example, fill production values
     │
5. Restore latest backup (if recovering data):
     docker compose up -d postgres
     # Wait for healthy
     docker compose exec -T postgres \
       sh -c "gunzip -c /backups/daily/latest.sql.gz | psql -U \$POSTGRES_USER -d \$POSTGRES_DB"
     │
6. Deploy:
     ./scripts/deploy.sh
     │
7. Verify:
     curl -s https://DOMAIN/health/ready   → {"status":"ready"}
     curl -s https://DOMAIN/               → Frontend loads
     │
8. Restore DNS / update load balancer
```

### RTO/RPO Targets

| Metric | Target | Method |
|--------|--------|--------|
| RPO (data loss) | ≤24h | Daily pg_dump backups |
| RTO (recovery time) | ≤30min | Scripted deployment + restore |

### Rollback Procedure

```bash
# 1. Identify the previous working image tag
docker compose logs app | head -5

# 2. Update .env or docker-compose to pin previous image
# APP_IMAGE=modular-monolith-app:v1.2.3

# 3. Redeploy
docker compose up -d app

# 4. Verify health
docker compose exec -T app wget -qO- http://127.0.0.1:8080/health/ready
```

---

## 4. CI/CD Pipeline Diagram

```
┌─────────────────────────────────────────────────────────┐
│                  GitHub Actions CI                        │
│  Trigger: push/PR to main                                │
│                                                          │
│  ┌─────────────────┐    ┌─────────────────┐            │
│  │    Backend       │    │    Frontend      │            │
│  │                  │    │                  │            │
│  │  • gofmt check   │    │  • pnpm install  │            │
│  │  • go vet        │    │  • svelte-check  │            │
│  │  • go test -race │    │  • lint          │            │
│  │  • go build      │    │  • build         │            │
│  └────────┬─────────┘    └────────┬─────────┘            │
│           │                        │                      │
│           └──────────┬─────────────┘                      │
│                      ▼                                    │
│           ┌─────────────────┐                            │
│           │     Docker       │                            │
│           │                  │                            │
│           │  • compose config│                            │
│           │  • compose build │                            │
│           └─────────────────┘                            │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│               GitHub Actions Release                     │
│  Trigger: tag push (v*)                                  │
│                                                          │
│  • Login to GHCR                                         │
│  • Build image                                           │
│  • Tag: semver + sha                                     │
│  • Push to ghcr.io                                       │
└─────────────────────────────────────────────────────────┘
```

---

## 5. Files Created/Modified

| File | Action |
|------|--------|
| `scripts/backup.sh` | Created — pg_dump with retention |
| `scripts/restore.sh` | Created — gzip restore with safety |
| `scripts/deploy.sh` | Created — full deployment automation |
| `.github/workflows/ci.yml` | Created — CI pipeline |
| `.github/workflows/release.yml` | Created — Release workflow |
| `docs/PHASE18C_OPERATIONS.md` | Created — This document |

---

## 6. Monitoring & Operations

### OpenObserve Setup

OpenObserve is included in the stack (`openobserve/openobserve:v0.14.5`) at `http://localhost:5080`.

- **Tracing**: OTLP traces sent from the Go app via `otelhttp` middleware
- **Access**: `ZO_ROOT_USER_EMAIL` / `ZO_ROOT_USER_PASSWORD` from `.env`
- **Data**: Persistent volume `openobserve_data`

### Health Endpoints

| Endpoint | Purpose | Expected |
|----------|---------|----------|
| `/health/live` | Liveness probe | `{"status":"ok"}` — always 200 |
| `/health/ready` | Readiness probe | `{"status":"ready"}` — 200 when DB reachable |
| `/metrics` | Prometheus metrics | Protected by `METRICS_TOKEN` |

### Recommended Alerting

| Metric | Condition | Severity |
|--------|-----------|----------|
| `/health/ready` returns non-200 | > 30s | Critical |
| Container restart count | > 3 in 5min | Warning |
| Disk usage | > 85% | Warning |
| Backup age | > 26h (no daily backup) | Critical |
| Certificate expiry | < 14 days | Warning |
| Memory usage | > 90% of limit | Warning |

### Logging

All services use `json-file` driver with rotation:
- Dev: 10MB × 3 files per container
- Prod: 25MB × 5 files per container

Access logs: `docker compose logs app`

---

## 7. Runbooks

### RB-001: Service Unavailable

```
Symptoms: /health/ready returns 503, users see errors

1. Check container status:
   docker compose ps

2. If app is not running:
   docker compose logs app --tail=50
   docker compose restart app

3. If postgres is not healthy:
   docker compose logs postgres --tail=50
   docker compose restart postgres
   # Wait for healthcheck, app will reconnect

4. If all containers are up but app is unhealthy:
   docker compose exec app wget -qO- http://127.0.0.1:8080/health/live
   # If live but not ready → DB connection issue
   docker compose exec postgres pg_isready -U postgres

5. Nuclear option:
   docker compose down
   docker compose up -d
```

### RB-002: Failed Deployment

```
Symptoms: deploy.sh fails, new version won't start

1. Check build errors:
   docker compose build app 2>&1 | tail -30

2. If build passes but container won't start:
   docker compose logs app --tail=50
   # Look for: migration failures, config errors, DB connection

3. Rollback to previous image:
   docker compose pull  # or tag previous known-good image
   docker compose up -d app

4. If migration caused the failure:
   # Check schema_migrations table for partially applied
   docker compose exec postgres psql -U postgres -d app_db \
     -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;"
```

### RB-003: Database Restore

```
Symptoms: Data corruption, accidental deletion, need to restore

1. List available backups:
   ls -la /backups/daily/ /backups/weekly/

2. Stop the application (prevent writes during restore):
   docker compose stop app

3. Restore:
   docker compose exec -T postgres \
     sh -c "gunzip -c /backups/daily/FILENAME.sql.gz | psql -U \$POSTGRES_USER -d \$POSTGRES_DB --single-transaction"

4. Restart application:
   docker compose start app

5. Verify:
   docker compose exec -T app wget -qO- http://127.0.0.1:8080/health/ready
```

### RB-004: Certificate Renewal Failure

```
Symptoms: HTTPS errors, cert expired, Traefik logs show ACME errors

1. Check Traefik logs:
   docker compose logs traefik --tail=50 | grep -i "acme\|cert\|tls"

2. Verify DNS points to this server:
   dig +short $DOMAIN

3. Verify port 80 is reachable (required for HTTP-01 challenge):
   curl -sI http://$DOMAIN/.well-known/acme-challenge/test

4. Force renewal — delete acme.json and restart:
   docker compose exec traefik rm /letsencrypt/acme.json
   docker compose restart traefik

5. Switch to staging for debugging:
   # Set CERT_RESOLVER=letsencrypt-staging in .env
   docker compose up -d traefik
```

### RB-005: Secret Rotation

```
Symptoms: Scheduled rotation or suspected compromise

1. Generate new values:
   SESSION_SECRET=$(openssl rand -hex 32)
   POSTGRES_PASSWORD=$(openssl rand -hex 24)

2. Update .env with new values

3. For DATABASE_URL, update both the URL and POSTGRES_PASSWORD

4. If rotating DB password:
   # Update in postgres first
   docker compose exec postgres psql -U postgres \
     -c "ALTER USER postgres PASSWORD 'new_password';"

5. Restart services:
   docker compose up -d

6. Verify:
   docker compose exec -T app wget -qO- http://127.0.0.1:8080/health/ready
```

### RB-006: Migration Failure

```
Symptoms: App won't start, logs show "migration failed" or "apply NNN_file.sql"

1. Check app logs:
   docker compose logs app --tail=20

2. Identify the failing migration file from the error

3. Check what's been applied:
   docker compose exec postgres psql -U postgres -d app_db \
     -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;"

4. If partially applied (rare with --single-transaction):
   # Fix the SQL manually, then mark as applied:
   docker compose exec postgres psql -U postgres -d app_db \
     -c "INSERT INTO schema_migrations (version) VALUES ('NNN_file.sql');"

5. Restart app:
   docker compose restart app
```

### RB-007: Rollback

```
Symptoms: New version has bugs, need to go back

1. Identify previous image tag:
   docker images modular-monolith-app --format '{{.Tag}} {{.CreatedAt}}' | head -5

2. Pin to previous version in .env or compose:
   # APP_IMAGE=modular-monolith-app:previous-tag

3. Bring up previous version:
   docker compose up -d app

4. If DB migrations need reverting (rare — only if new migration broke things):
   # Restore from backup taken before deployment
   # See RB-003

5. Verify:
   docker compose exec -T app wget -qO- http://127.0.0.1:8080/health/ready
```

---

## 8. Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ Pass |
| `go test ./... -short` | ✅ Pass |
| `docker compose config` | ✅ Valid |
| GitHub Actions YAML syntax | ✅ Valid (standard structure) |
| `scripts/backup.sh` shellcheck-safe | ✅ set -euo pipefail, quoted vars |
| `scripts/restore.sh` shellcheck-safe | ✅ set -euo pipefail, safety delay |
| `scripts/deploy.sh` shellcheck-safe | ✅ set -euo pipefail, early-fail |
| Backup retention configurable | ✅ Via BACKUP_RETAIN_{DAILY,WEEKLY,MONTHLY} |
| Restore procedure documented | ✅ Script + runbook |
| DR sequence documented | ✅ Empty server → healthy deployment |
| Runbooks cover common scenarios | ✅ 7 runbooks |
