1. Current Deployment Architecture

  ┌────────────────────────────────────────────────────────────────┐
  │                    Docker Compose Stack                         │
  │                                                                │
  │  ┌──────────┐    ┌──────────┐    ┌──────────────┐            │
  │  │ Traefik  │───▶│   App    │───▶│  PostgreSQL  │            │
  │  │  (v3.0)  │    │  (Go)    │    │    (v17)     │            │
  │  │  :80/:443│    │  :8080   │    │    :5432     │            │
  │  └──────────┘    └────┬─────┘    └──────────────┘            │
  │                       │                                       │
  │                       ▼                                       │
  │                  ┌──────────────┐                             │
  │                  │ OpenObserve  │                             │
  │                  │  (OTLP)     │                             │
  │                  │   :5080     │                             │
  │                  └──────────────┘                             │
  └────────────────────────────────────────────────────────────────┘

  - Single Go binary serving HTTP API + embedded SvelteKit frontend
  - PostgreSQL 17 with RLS-based multi-tenancy
  - Traefik v3 as reverse proxy with HTTP→HTTPS redirect configured
  - OpenObserve for OTLP traces
  - Tern for database migrations (manual execution via Makefile)
  - Named Docker volumes for PostgreSQL data and OpenObserve data

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  2. Findings

  DR-001 — No TLS Certificates Configured

  - Severity: Critical
  - Description: Traefik config defines a websecure entrypoint on :443 and HTTP→HTTPS redirect, but no TLS certificate resolver (Let's Encrypt/ACME) or static certificate is configured. The
  websecure entrypoint has no tls block. Traffic cannot complete HTTPS handshake.
  - Impact: HTTPS is unreachable in production. All clients get connection refused or TLS error on port 443. HTTP redirect sends users to a broken endpoint.
  - File(s): deployments/traefik/traefik.yml, docker-compose.yml
  - Recommended fix: Add ACME (Let's Encrypt) certResolver configuration to traefik.yml with a certificatesResolvers section, DNS or HTTP challenge, and reference it in docker-compose Traefik
  labels on the app service router.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-002 — Traefik Dashboard Exposed Without Auth

  - Severity: Critical
  - Description: api.insecure: true exposes the Traefik dashboard on port 8080 without authentication. In docker-compose, this is bound to 127.0.0.1:8081:8080, but the insecure: true flag
  means no auth is enforced if the port becomes externally reachable.
  - Impact: Full visibility into routing rules, backend health, and configuration. In production, if host firewall rules change, this becomes an information disclosure vector.
  - File(s): deployments/traefik/traefik.yml
  - Recommended fix: Set api.insecure: false, add basicAuth or forwardAuth middleware to the dashboard, or remove the dashboard entirely in production.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-003 — No Backup Strategy Exists

  - Severity: Critical
  - Description: There is no backup script, no pg_dump cron job, no WAL archiving, and no backup documentation. The scripts/ directory is empty.
  - Impact: Single point of data failure. If the postgres_data volume is corrupted or the host fails, all data is permanently lost.
  - File(s): scripts/ (empty), docker-compose.yml
  - Recommended fix: Implement automated pg_dump (logical) backups on a schedule, configure WAL archiving for PITR, store backups externally (S3/off-host), and add a tested restore procedure.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-004 — No Disaster Recovery Plan

  - Severity: Critical
  - Description: No documented or automated recovery procedure. No RTO/RPO targets defined. No runbook for restoring from a volume loss, container crash loop, or full host failure.
  - Impact: Extended downtime in any failure scenario. Recovery actions would be ad-hoc.
  - File(s): N/A (missing documentation)
  - Recommended fix: Create a disaster recovery runbook with documented RTO/RPO, tested restore procedures, and periodic DR drills.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-005 — Secrets in Plain Text / Committed .env

  - Severity: Critical
  - Description: Although .env is in .gitignore, the checked-in .env.example contains the actual OpenObserve Base64 credentials (YWRtaW5AZXhhbXBsZS5jb206U3Ryb25nUGFzc3dvcmQxMjM= decodes to
  admin@example.com:StrongPassword123). The tern.conf hardcodes postgres credentials. docker-compose defaults embed postgres:postgres credentials.
  - Impact: Default credentials in production deployments if operators don't override. Credential leakage to anyone with repo access.
  - File(s): .env.example, migrations/tern.conf, docker-compose.yml
  - Recommended fix: Use Docker secrets or a secrets manager (Vault, AWS Secrets Manager). Remove all credentials from committed files. Use environment variable references in tern.conf.
  Enforce non-default passwords in production config validation.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-006 — No Resource Limits on Containers

  - Severity: High
  - Description: None of the four services (app, postgres, openobserve, traefik) have deploy.resources.limits (memory/CPU) defined.
  - Impact: A memory leak or runaway query in any container can exhaust host resources, causing OOM kills across all services and cascading failures.
  - File(s): docker-compose.yml
  - Recommended fix: Add deploy.resources.limits and reservations for all services. E.g., app: 512MB/0.5CPU, postgres: 1GB/1CPU, openobserve: 512MB, traefik: 128MB.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-007 — No Database Connection Pool Configuration

  - Severity: High
  - Description: database.New() uses pgxpool.ParseConfig(databaseURL) without setting MaxConns, MinConns, MaxConnLifetime, or MaxConnIdleTime. The pool uses pgxpool defaults (4 conns per
  CPU).
  - Impact: In production under load, the pool may be too small (starving requests) or connections may be held indefinitely (leaking after PostgreSQL restarts).
  - File(s): internal/database/postgres.go
  - Recommended fix: Expose pool size configuration via environment variables. Set explicit MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime, and HealthCheckPeriod.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-008 — Database Migrations Not Automated in Deployment

  - Severity: High
  - Description: Migrations are run manually via make migrate (tern CLI). The docker-compose setup has no init container or entrypoint that runs migrations before the app starts. The app will
  fail on startup if the database schema is out of date.
  - Impact: Deployment requires manual intervention. Risk of forgetting migrations, causing runtime errors against a stale schema.
  - File(s): Makefile, docker-compose.yml, migrations/tern.conf
  - Recommended fix: Add a migration init container or a pre-start script in docker-compose that runs tern migrate before the app starts. Or embed migrations in the Go binary and run them on
  startup with a lock.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-009 — No Rollback Strategy

  - Severity: High
  - Description: There is no documented rollback procedure. Migrations use tern's ---- create above / drop below ---- convention for down migrations, but there is no tooling or procedure to
  invoke them. No blue-green or canary deployment mechanism exists.
  - Impact: A bad deployment requires manual intervention to revert. Schema rollbacks with data loss risk are undefined.
  - File(s): migrations/, Makefile
  - Recommended fix: Document rollback procedure. Tag Docker images with git SHA for easy revert. Create make rollback target. Consider separating destructive schema changes from additive
  ones.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-010 — Docker Socket Mounted to Traefik (Security Risk)

  - Severity: High
  - Description: /var/run/docker.sock:/var/run/docker.sock:ro gives Traefik read access to the Docker daemon. This is a well-known privilege escalation vector—a container escape via Traefik
  gives full Docker API access.
  - Impact: If Traefik is compromised, the attacker can inspect/control all containers on the host.
  - File(s): docker-compose.yml
  - Recommended fix: Use Traefik's Docker socket proxy (e.g., tecnativa/docker-socket-proxy) to limit exposed API surface, or switch to file-based provider for production.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-011 — OpenObserve Uses latest Tag

  - Severity: High
  - Description: image: openobserve/openobserve:latest means the image can change unpredictably between deployments.
  - Impact: Breaking changes, incompatible API versions, or regressions introduced silently during docker compose pull.
  - File(s): docker-compose.yml
  - Recommended fix: Pin to a specific version tag (e.g., openobserve/openobserve:v0.12.1).

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-012 — No Logging Retention or Rotation

  - Severity: High
  - Description: Application logs go to stdout (docker captures). No Docker logging driver configuration, no max-size/max-file limits defined. OpenObserve stores traces in a Docker volume
  with no retention policy configured.
  - Impact: Disk exhaustion over time as log files and trace data grow unbounded. Eventual container crash or host full.
  - File(s): docker-compose.yml, pkg/logger/logger.go
  - Recommended fix: Configure Docker logging driver with max-size and max-file options per service. Set OpenObserve data retention policies. Consider forwarding logs to OpenObserve's log
  ingestion endpoint.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-013 — PostgreSQL sslmode=disable

  - Severity: High
  - Description: The DATABASE_URL in docker-compose uses ?sslmode=disable. Database traffic between app and postgres containers is unencrypted.
  - Impact: Within the Docker network this is low risk, but if the architecture evolves to external PostgreSQL (RDS, separate host), credentials and data are transmitted in clear text.
  - File(s): docker-compose.yml, .env.example
  - Recommended fix: For production, use sslmode=require or sslmode=verify-full. Configure PostgreSQL with TLS certificates. Keep disable only for local development.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-014 — Traefik Router Uses web Entrypoint (HTTP)

  - Severity: Medium
  - Description: The app's Traefik labels route via entrypoints: web (port 80). The Traefik config redirects web to websecure, but there are no router labels for the websecure entrypoint.
  After redirect, no router matches on websecure.
  - Impact: Application is unreachable after HTTPS redirect. Requests loop or 404.
  - File(s): docker-compose.yml, deployments/traefik/traefik.yml
  - Recommended fix: Add labels for websecure entrypoint routers with TLS configuration, or remove the HTTP→HTTPS redirect for development and add it only for production.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-015 — App Missing Required Env Vars in Docker Compose

  - Severity: Medium
  - Description: The docker-compose app environment section is missing: OIDC_ISSUER, OIDC_AUDIENCE, OIDC_CLIENT_ID, OIDC_REDIRECT_URL, SESSION_SECRET, DEV_TOKEN_SECRET, CORS_ORIGIN,
  METRICS_TOKEN. The config validator requires these—app will panic on startup.
  - Impact: App container will crash-loop immediately unless all vars are provided via the shell environment or a .env file.
  - File(s): docker-compose.yml, internal/config/validate.go
  - Recommended fix: Add all required environment variables to the docker-compose app service, with appropriate defaults for development and documentation for production overrides.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-016 — No Container Healthcheck on Traefik

  - Severity: Medium
  - Description: The Traefik service has no healthcheck defined. Other services depend on it but have no way to detect if Traefik is healthy.
  - Impact: No automated detection or restart if Traefik becomes unresponsive.
  - File(s): docker-compose.yml
  - Recommended fix: Add a healthcheck that queries the Traefik health endpoint: wget -qO- http://127.0.0.1:8080/ping.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-017 — Migrations Run as Superuser

  - Severity: Medium
  - Description: tern.conf uses the postgres superuser for migrations. The application also connects as postgres. RLS policies are bypassed for table owners/superusers by default.
  - Impact: The application connection as the table owner bypasses RLS unless FORCE ROW LEVEL SECURITY is enabled on tables. Multi-tenant isolation may be ineffective.
  - File(s): migrations/tern.conf, docker-compose.yml
  - Recommended fix: Create a separate application user with limited privileges. Add ALTER TABLE ... FORCE ROW LEVEL SECURITY to ensure RLS applies to the table owner. Use the superuser only
  for migrations.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-018 — No Production Docker Compose Override

  - Severity: Medium
  - Description: deployments/docker/docker-compose.yml and deployments/docker/.env are both empty (0 bytes). There is no production-specific compose configuration.
  - Impact: No clear separation between development and production configurations. Operators must manually manage production overrides.
  - File(s): deployments/docker/docker-compose.yml, deployments/docker/.env
  - Recommended fix: Create a production override compose file with production-specific settings (resource limits, TLS, pinned versions, no debug endpoints).

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-019 — Startup Sequencing Gap: OpenObserve Not Healthy Before App Starts

  - Severity: Medium
  - Description: App depends on OpenObserve with condition: service_started (not service_healthy). If OpenObserve is slow to start, the app may fail OTLP export initially.
  - Impact: Trace data loss during startup window. Not critical due to graceful degradation in OTEL init (returns noopShutdown on error), but still results in missing early traces.
  - File(s): docker-compose.yml
  - Recommended fix: Change to condition: service_healthy since OpenObserve already has a healthcheck defined.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-020 — No Prometheus/Metrics Scraping Configured

  - Severity: Medium
  - Description: The app exposes a /metrics endpoint (Prometheus format) but no Prometheus or scraping service is defined in the compose stack. Metrics are exposed but never collected.
  - Impact: Production metrics (request rates, latencies, error rates) are not being stored or alerted on. No operational dashboards.
  - File(s): internal/router/router.go, docker-compose.yml
  - Recommended fix: Add Prometheus to the compose stack, or configure OpenObserve's Prometheus remote write receiver. Set up alerting rules.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-021 — Logging Middleware Does Not Capture Response Status

  - Severity: Medium
  - Description: The logging middleware records method, path, and duration but does not capture the HTTP response status code.
  - Impact: Cannot distinguish between 2xx, 4xx, and 5xx responses in logs. Makes debugging and monitoring significantly harder.
  - File(s): internal/middleware/logging.go
  - Recommended fix: Wrap the http.ResponseWriter to capture the status code and include it in the log output.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-022 — No Graceful Drain Period Before Shutdown

  - Severity: Low
  - Description: The shutdown handler immediately begins shutting down the HTTP server. There's no pre-shutdown delay to allow load balancers (Traefik) to deregister the backend.
  - Impact: In-flight requests during deployment may receive connection resets if Traefik still routes to the container after it stops accepting connections.
  - File(s): internal/app/shutdown.go
  - Recommended fix: Add a small sleep (2-5s) before calling HTTPServer.Shutdown() to allow Traefik to detect the health check failure and stop routing.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-023 — No Docker Image Tagging Strategy

  - Severity: Low
  - Description: The app image is tagged modular-monolith-app:local. No CI/CD pipeline or versioning strategy is visible.
  - Impact: No way to roll back to a previous image version. No traceability from running container to source code.
  - File(s): docker-compose.yml
  - Recommended fix: Tag images with git SHA and/or semantic version. Maintain an image registry with historical versions.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-024 — Dockerfile Uses Non-Existent Go Version

  - Severity: Low
  - Description: FROM golang:1.26-alpine — Go 1.26 does not exist as of 2026-06. This will fail docker build.
  - Impact: Build failure. Must be corrected before any Docker deployment.
  - File(s): Dockerfile
  - Recommended fix: Use the correct Go version matching go.mod (likely golang:1.24-alpine or whatever the project uses).

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-025 — No Non-Root User in Runtime Container

  - Severity: Low
  - Description: The final Alpine stage runs the binary as root. No USER directive or adduser in the Dockerfile.
  - Impact: If the container is compromised, the attacker has root privileges within the container, increasing escape risk.
  - File(s): Dockerfile
  - Recommended fix: Add RUN adduser -D -u 1001 appuser and USER appuser before the ENTRYPOINT.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  DR-026 — tern.conf Has Hardcoded Credentials

  - Severity: Low
  - Description: migrations/tern.conf hardcodes user = postgres and password = postgres. This file is committed to version control.
  - Impact: Works for local development but unsuitable for production. Production credentials would need a separate config mechanism.
  - File(s): migrations/tern.conf
  - Recommended fix: Use tern's environment variable substitution ($DATABASE_URL or individual $PGPASSWORD) for production migrations.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  3. Risk Summary

  ┌──────────┬───────┬────────────────────────────────────────────────────────────────┐
  │ Severity │ Count │ IDs                                                            │
  ├──────────┼───────┼────────────────────────────────────────────────────────────────┤
  │ Critical │ 5     │ DR-001, DR-002, DR-003, DR-004, DR-005                         │
  ├──────────┼───────┼────────────────────────────────────────────────────────────────┤
  │ High     │ 8     │ DR-006, DR-007, DR-008, DR-009, DR-010, DR-011, DR-012, DR-013 │
  ├──────────┼───────┼────────────────────────────────────────────────────────────────┤
  │ Medium   │ 8     │ DR-014, DR-015, DR-016, DR-017, DR-018, DR-019, DR-020, DR-021 │
  ├──────────┼───────┼────────────────────────────────────────────────────────────────┤
  │ Low      │ 5     │ DR-022, DR-023, DR-024, DR-025, DR-026                         │
  ├──────────┼───────┼────────────────────────────────────────────────────────────────┤
  │ Total    │ 26    │                                                                │
  └──────────┴───────┴────────────────────────────────────────────────────────────────┘

  Overall Assessment: Not production-ready. The stack has critical gaps in TLS termination, backup/DR, and secrets management that must be resolved before any production traffic is served.
  The Docker Compose setup is functional for development but requires significant hardening.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  4. Recommended Implementation Order

  Phase 1 — Blockers (must fix before production)

  1. DR-005 — Secret management (move secrets out of repo, use Docker secrets or vault)
  2. DR-001 — TLS certificates (configure ACME/Let's Encrypt in Traefik)
  3. DR-014 — Fix Traefik HTTPS router labels (unblocks TLS)
  4. DR-003 — Implement backup strategy (pg_dump + off-site storage)
  5. DR-004 — Document disaster recovery procedure

  Phase 2 — High-priority hardening

  6. DR-017 — Separate app DB user + FORCE RLS
  7. DR-015 — Complete docker-compose environment variables
  8. DR-008 — Automate migrations in deployment pipeline
  9. DR-006 — Add resource limits to all containers
  10. DR-013 — Enable SSL for PostgreSQL connections
  11. DR-010 — Replace Docker socket with socket proxy
  12. DR-011 — Pin OpenObserve version
  13. DR-012 — Configure log rotation and retention

  Phase 3 — Operational maturity

  14. DR-002 — Secure or remove Traefik dashboard
  15. DR-009 — Document rollback strategy
  16. DR-018 — Create production compose override
  17. DR-020 — Add metrics collection (Prometheus)
  18. DR-019 — Fix OpenObserve startup dependency
  19. DR-016 — Add Traefik healthcheck
  20. DR-007 — Configure connection pool parameters
  21. DR-021 — Add response status to logging

  Phase 4 — Polish

  22. DR-024 — Fix Dockerfile Go version
  23. DR-025 — Run as non-root user
  24. DR-022 — Add graceful drain period
  25. DR-023 — Implement image tagging strategy
  26. DR-026 — Externalize tern credentials
