# Authentication Patterns

## Identity Architecture

Identity providers must:
- remain interface-driven
- avoid direct Zitadel coupling in handlers

---

## JWT Rules

JWT validation belongs in:
- middleware
- provider layer

Never:
- handlers

---

## Tenant Rules

organization_id propagation is:
- infrastructure concern
- middleware responsibility

---

## Middleware

Auth middleware:
- validates token
- extracts claims
- injects request context

Handlers should trust:
- authenticated context

---

## Future Zitadel Integration

Planned:
- OIDC discovery
- JWKS verification
- RS256 validation
- issuer validation
- audience validation