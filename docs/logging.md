# Logging Rules

Never log:
- passwords
- JWTs
- API keys
- cookies
- auth headers

Always log:
- request IDs
- tenant IDs
- errors
- external provider failures

Use structured fields instead of string formatting.