# Hardening Guide

## Container Security

### Applied Hardening

| Control | App | Postgres | Traefik |
|---------|-----|----------|---------|
| Non-root user | ✅ UID 10001 | ✅ (postgres user) | ✅ (traefik user) |
| Read-only filesystem | ✅ | — (needs data dir) | — (needs acme.json) |
| cap_drop ALL | ✅ | ✅ + minimal cap_add | ✅ + NET_BIND_SERVICE |
| no-new-privileges | ✅ | ✅ | ✅ |
| Resource limits | ✅ 512M/1CPU | ✅ 2G/2CPU | ✅ 128M/0.25CPU |
| Ulimits (nofile) | ✅ 65536 | — | — |
| tmpfs for /tmp | ✅ 64M | — | — |
| Log rotation | ✅ 25M×5 | ✅ 25M×5 | ✅ 25M×5 |

### Intentionally Deferred

| Control | Reason |
|---------|--------|
| seccomp profiles | Requires custom profile testing per workload; default Docker seccomp is adequate |
| AppArmor | Distribution-specific; works with default Docker AppArmor profile |
| PID limits | Compose version conflict with deploy.resources; use host-level cgroup limits |
| Network policies | Docker bridge isolation is sufficient for single-host; use Kubernetes NetworkPolicy for multi-node |

## Production Validation

The application enforces at startup (APP_ENV=production):

| Check | Error Message |
|-------|--------------|
| SESSION_SECRET not placeholder | "must not use the default placeholder" |
| DEV_TOKEN_SECRET absent | "must not be set in production" |
| OIDC_REDIRECT_URL uses https | "must use https:// in production" |
| CORS_ORIGIN uses https | "must use https:// in production" |
| DATABASE_URL no sslmode=disable | "must not use sslmode=disable" |
| OTEL not insecure | "must be false in production" |
| CORS_ORIGIN set | "required in production" |
| METRICS_TOKEN set | "required in production" |
| DODO_BASE_URL set | "required in production" |

## Docker Secrets

Secrets are read via the `_FILE` convention:

```
SESSION_SECRET_FILE=/run/secrets/session_secret → reads file → sets SESSION_SECRET
```

Supported secrets:
- `SESSION_SECRET`
- `DATABASE_URL`
- `POSTGRES_PASSWORD`
- `DODO_API_KEY`
- `DODO_WEBHOOK_SECRET`
- `METRICS_TOKEN`
- `ZO_ROOT_USER_PASSWORD`
- `OTEL_EXPORTER_OTLP_HEADERS`

Backward-compatible: if `_FILE` is not set, the regular env var is used.

## Image Security

- Base image: `alpine:3.21.3` (minimal attack surface)
- No package manager in runtime (no apk add)
- Static binary (CGO_ENABLED=0)
- Stripped symbols (-ldflags="-s -w")
- No shell access needed (binary runs directly)

## Network Security

- Internal Docker bridge network (not exposed)
- Only Traefik ports (80, 443) are published
- PostgreSQL not exposed to host in production
- Service-to-service communication via Docker DNS

## TLS

- Automatic HTTPS via Let's Encrypt ACME
- HTTP → HTTPS 301 permanent redirect
- HSTS with 2-year max-age, includeSubdomains, preload
- TLS termination at Traefik (no TLS between Traefik ↔ App on internal network)

## Monitoring

- Health: `/health/live` (liveness), `/health/ready` (readiness with DB check)
- Metrics: `/metrics` (Prometheus, token-protected)
- Traces: OTLP to OpenObserve (or external collector)
- Logs: JSON stdout, captured by Docker json-file driver

## Host-Level Recommendations

These are outside Docker Compose scope but recommended:

1. **Firewall**: Only allow inbound 80, 443, and SSH
2. **Automatic updates**: Enable unattended-upgrades for kernel/OS patches
3. **SSH hardening**: Key-only auth, disable root login
4. **Disk encryption**: Encrypt the Docker data directory
5. **Audit logging**: Enable auditd for system-level audit trail
6. **Backup offsite**: Copy backup archives to S3/remote storage
