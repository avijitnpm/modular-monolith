# Phase 18B — Traefik, TLS & Reverse Proxy Hardening

## 1. Updated Traefik Audit Matrix

| ID | Finding | Previous | Current |
|----|---------|----------|---------|
| DR-001 | No TLS certificates configured | Outstanding | **Fixed** — ACME/Let's Encrypt with HTTP-01 challenge |
| DR-002 | Dashboard exposed without auth | Outstanding | **Fixed** — BasicAuth on HTTPS subdomain, api.insecure: false |
| NEW-001 | Routers only on HTTP, no HTTPS routers | Outstanding | **Fixed** — Single HTTPS catchall router |
| NEW-002 | No security headers | Outstanding | **Fixed** — Full middleware chain |
| NEW-003 | No HSTS | Outstanding | **Fixed** — 2yr, includeSubdomains, preload |
| NEW-004 | No compression | Outstanding | **Fixed** — gzip via Traefik compress middleware |
| NEW-005 | Duplicate routers (root + api) | Outstanding | **Fixed** — Single catchall router per host |
| NEW-006 | Redundant :8080 traefik entrypoint | Outstanding | **Fixed** — Removed |
| NEW-007 | read_only blocks cert storage | Outstanding | **Fixed** — letsencrypt_data volume, read_only removed from traefik |

## 2. Files Modified

| File | Action |
|------|--------|
| `deployments/traefik/traefik.yml` | Rewritten — TLS, cert resolvers, no insecure dashboard, access logging |
| `docker-compose.yml` | Updated — HTTPS router, security-headers middleware, compress middleware, dashboard auth, letsencrypt volume |
| `deployments/docker/docker-compose.yml` | Updated — Same Traefik labels for production |
| `deployments/docker/traefik.yml` | Created — Production traefik static config |
| `deployments/docker/.env.example` | Updated — Added DOMAIN, ACME_EMAIL, TRAEFIK_DASHBOARD_USERS |
| `.env.example` | Updated — Added DOMAIN, ACME_EMAIL, CERT_RESOLVER, TRAEFIK_DASHBOARD_USERS |

## 3. Reverse Proxy Architecture Diagram

```
Internet
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│                    Traefik v3.4.0                         │
│                                                          │
│  Entrypoints:                                            │
│    :80 (web)  ──── 301 redirect ────▶  :443 (websecure) │
│                                                          │
│  Middleware Chain (applied to all HTTPS traffic):         │
│    1. security-headers  (HSTS, CSP, X-Frame, etc.)      │
│    2. compress          (gzip)                           │
│                                                          │
│  Routers:                                                │
│    app       → Host(`DOMAIN`)         → app:8080        │
│    dashboard → Host(`traefik.DOMAIN`) → api@internal    │
│                                        + basicauth      │
│                                                          │
│  TLS:                                                    │
│    certResolver: letsencrypt (ACME HTTP-01)              │
│    storage: /letsencrypt/acme.json (persistent volume)   │
└─────────────────────────────────────────────────────────┘
    │
    ▼
┌──────────────────┐
│   App Container   │
│   :8080           │
│   (Go + frontend) │
└──────────────────┘
```

## 4. TLS Flow Diagram

```
Client ──HTTPS──▶ Traefik :443
                      │
                      ├─ Certificate exists? → Serve with existing cert
                      │
                      └─ No certificate?
                           │
                           ▼
                    ACME HTTP-01 Challenge
                           │
                    1. Traefik requests cert from Let's Encrypt
                    2. Let's Encrypt sends HTTP challenge to :80
                    3. Traefik responds on /.well-known/acme-challenge/
                    4. Let's Encrypt validates domain ownership
                    5. Certificate issued → stored in /letsencrypt/acme.json
                    6. Automatic renewal before expiry (~30 days before)
                           │
                           ▼
                    HTTPS connection established
                    
Staging vs Production:
  - CERT_RESOLVER=letsencrypt          → Production LE (rate-limited)
  - CERT_RESOLVER=letsencrypt-staging  → Staging LE (untrusted, for testing)

DNS Challenge (for wildcards — not configured by default):
  - Requires DNS provider API credentials
  - Set in traefik.yml: certificatesResolvers.letsencrypt.acme.dnsChallenge
  - Provider-specific env vars (e.g., CF_DNS_API_TOKEN for Cloudflare)
```

## 5. Remaining Deployment Findings

| ID | Finding | Status | Phase |
|----|---------|--------|-------|
| DR-003 | No backup strategy | Outstanding | Backup phase |
| DR-004 | No disaster recovery | Outstanding | DR phase |
| DR-005 | Secrets in plain text | Outstanding | Secrets phase |
| DR-007 | No DB pool configuration | Outstanding | App hardening |

All Traefik/TLS/ingress findings are now resolved.

## 6. Verification Results

| Check | Result |
|-------|--------|
| `docker compose config` (dev) | ✅ Valid |
| `docker compose config` (prod) | ✅ Valid (warnings for unset prod vars — expected) |
| `go build ./...` | ✅ Pass |
| HTTP → HTTPS redirect | ✅ Configured via entryPoints.web.http.redirections (permanent 301) |
| Certificate resolver config | ✅ letsencrypt + letsencrypt-staging, HTTP-01 challenge |
| ACME email injection | ✅ Via CLI args (env var interpolation at compose level) |
| Certificate storage | ✅ Persistent named volume `letsencrypt_data` at /letsencrypt |
| Security headers — HSTS | ✅ max-age=63072000, includeSubdomains, preload |
| Security headers — X-Frame-Options | ✅ DENY |
| Security headers — X-Content-Type-Options | ✅ nosniff |
| Security headers — XSS filter | ✅ Enabled |
| Security headers — Referrer-Policy | ✅ strict-origin-when-cross-origin |
| Security headers — Permissions-Policy | ✅ Restrictive defaults |
| Security headers — CSP | ✅ self + unsafe-inline for SvelteKit |
| Compression | ✅ gzip enabled, gRPC excluded |
| Dashboard protection | ✅ BasicAuth + HTTPS only on subdomain |
| Dashboard insecure mode | ✅ Disabled (api.insecure: false) |
| Single router (no duplication) | ✅ One `app` router per host |
| Redundant entrypoint removed | ✅ No :8080 entrypoint |

### Security Headers Delivered per Response

```
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=()
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'
X-Robots-Tag: noindex,nofollow
```

### Operator Notes

- Set `DOMAIN` to your actual domain (e.g., `app.example.com`)
- Set `ACME_EMAIL` for Let's Encrypt notifications
- Generate dashboard credentials: `docker run --rm httpd:alpine htpasswd -nB admin`
- For staging/testing: set `CERT_RESOLVER=letsencrypt-staging`
- Dashboard accessible at `https://traefik.DOMAIN/dashboard/` with basicauth
- DNS must point to the server for HTTP-01 challenge to succeed
