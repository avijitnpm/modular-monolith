Modular Monolith Backend Architecture State
Overview

This project is a modular monolith backend built using:

Go 1.22+
Chi Router
PostgreSQL 17
Zitadel (planned real auth provider)
SvelteKit Static Frontend
OpenTelemetry + OpenObserve
Docker-based local infrastructure

The architecture follows:

interface-first design
modular boundaries
service/repository separation
middleware-driven request lifecycle
stateless JWT authentication
multi-tenant SaaS architecture
transaction-safe workflows

The backend is designed to become:

reusable SaaS infrastructure
enterprise-compatible
provider-swappable
deployment portable

The frontend is intentionally decoupled from backend runtime and will eventually be embedded into Go using embed.FS.

Current Backend Mental Model

The backend currently follows this request lifecycle:

HTTP Request
↓
Chi Router
↓
Middleware Chain
↓
JWT Authentication
↓
Tenant Context Injection
↓
Handler
↓
Validation
↓
Service Layer
↓
Transaction Wrapper
↓
Repository Layer
↓
PostgreSQL
↓
Commit / Rollback

This is the core architecture of the system.

High-Level Architectural Principles
1. Middleware-Driven Architecture

Authentication and tenancy are request enrichment systems.

Middleware is responsible for:

validating JWTs
extracting claims
injecting authenticated user
injecting organization context
propagating request-scoped metadata

Handlers should never manually parse auth tokens.

2. Interface-First Design

External systems are abstracted behind interfaces.

The system intentionally avoids hard coupling to:

Zitadel
Dodo Payments
JWT libraries
infrastructure providers

Current provider abstraction exists for:

Identity Provider

Future provider abstractions:

Payments
Email
Blob Storage
Queue Systems
AI Providers

The goal is:

swappable infrastructure
stable application layer
easier testing
3. Service Layer Owns Business Workflows

Repositories:

execute queries
remain relatively dumb

Services:

orchestrate workflows
own transaction boundaries
coordinate multi-step operations

This separation became important during:

transaction architecture
audit log integration
rollback handling
4. PostgreSQL is Treated as a Security Boundary

This project intentionally uses PostgreSQL Row Level Security (RLS) concepts.

The long-term architecture is:

JWT Org Claims
↓
Request Context
↓
Transaction Context
↓
Postgres Session Variable
↓
RLS Policies
↓
Tenant Isolation

This means:
tenant safety should eventually be enforced by the database itself, not by application-level WHERE clauses everywhere.

Current Implemented Systems
Router Layer

Using:

Chi Router

Current route structure:

public routes
protected routes
middleware groups

Current major routes:

/health
/api/v1/token
/api/v1/users

Protected routes use:

JWT middleware
tenant middleware
Logging System

Structured logging exists using:

slog

Current logs include:

method
path
request duration
panic recovery

The architecture is prepared for:

OpenTelemetry tracing
OpenObserve ingestion

Current logs are still relatively minimal.

Config System

Config is environment-driven.

Current stack:

.env
Koanf

The system intentionally avoids:

complex secret managers
early overengineering

Current important configs:

server port
database URL
environment
auth-related values
Authentication System
Current State

The system currently uses:

locally generated JWTs
middleware validation
provider abstraction

Auth flow currently works as:

Generate Token
↓
Bearer Token Request
↓
JWT Validation
↓
Claims Extraction
↓
Authenticated User Injection

Current claims:

user_id
organization_id
email
Identity Provider Architecture

A provider abstraction exists:

identity.Provider

This was created to avoid coupling auth logic directly to:

JWT libraries
Zitadel-specific implementations

Current provider:

ZitadelProvider

Currently still uses:

local JWT secret

Planned future evolution:

OIDC discovery
JWKS fetching
RS256 validation
issuer verification
audience verification
real Zitadel integration
Multi-Tenancy Architecture

The backend is evolving into:

organization-aware SaaS infrastructure

Current request flow includes:

authenticated user context
organization context propagation

The system is designed so every request eventually becomes:

org-aware
tenant-safe
Important Tenant Concept

Authentication answers:

who is the user

Tenancy answers:

which organization owns this request

These are separate concerns.

Database Architecture
PostgreSQL

Current DB:

PostgreSQL 17
Dockerized locally

Tables currently include:

users
organizations
audit_logs
Repository Pattern

Repositories currently own:

raw SQL execution
persistence logic

Current stack:

pgx
raw SQL
Tern migrations

The project intentionally avoids:

heavy ORM abstraction
Migration System

Using:

Tern

A major debugging issue occurred because migration files were accidentally created outside the migrations directory.

This caused:

schema drift
code/database mismatch
transaction failures

Key lesson:
migrations are part of application state, not optional SQL notes.

Transaction Architecture

The system now supports:

transaction wrappers
rollback safety
atomic workflows

Current architecture:

Service Layer
↓
WithTransaction()
↓
Repository Operations
↓
Commit or Rollback

This became important when:

audit log insertion failed
user creation rolled back correctly

This proved transaction consistency was working.

Audit Logging System

Current audit logging is still primitive.

Current stored fields:

id
action
created_at

The architecture is prepared for future expansion:

actor IDs
org IDs
metadata payloads
entity tracking
compliance events
Current Major Lessons Learned
1. Infrastructure State Matters More Than Code

Major debugging issues came from:

migration drift
schema mismatch
wrong DB assumptions
stale tables

Not from:

router syntax
Go language features

This shifted the understanding from:
“backend = code”
to:
“backend = stateful infrastructure system”

2. Middleware Is Request Enrichment

Middleware became easier to understand once viewed as:

request enrichment pipeline

rather than:

magic auth system
3. Transactions Protect Business Consistency

The biggest realization:
transactions are not DB convenience features.

They are:

business consistency guarantees
4. Interface Boundaries Matter

Import cycles and provider abstractions exposed how important dependency direction is.

The system evolved toward:

middleware
↓
provider interface
↓
provider implementation
↓
library

instead of tangled dependency graphs.

Current Weak Areas

The project is NOT finished.

Current weak/incomplete areas:

real Zitadel integration
JWKS validation
production-grade RLS
tracing
metrics
structured audit payloads
billing integration
email infrastructure
deployment stack
frontend integration
role/permission systems
Planned Future Architecture
Identity
Real Zitadel OIDC
JWKS verification
RS256 validation
role claims
org claims
Observability
OpenTelemetry
distributed traces
OpenObserve ingestion
request spans
DB spans
Billing
Dodo Payments abstraction
webhook processing
org subscription state
metadata sync
Frontend
SvelteKit static build
embed.FS serving
API + SPA unified binary
Infrastructure
Docker Compose
Traefik
HTTPS routing
SSH-secured DB access
VPS deployment
Current Project State

Current backend maturity is approximately:

70–75% of core backend architecture complete

The hardest conceptual layers are already implemented:

middleware
auth
tenancy
transactions
provider abstraction

Remaining work is mostly:

infrastructure refinement
integrations
observability
deployment
production hardening
Most Important Current Mental Model

The backend should now mentally feel like:

request enters
↓
middleware enriches context
↓
services orchestrate workflows
↓
repositories persist state
↓
transactions protect consistency
↓
database enforces safety

That is the core architecture currently being built.