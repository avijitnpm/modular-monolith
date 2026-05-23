# Backend Patterns

## Handlers

Handlers:
- decode requests
- validate input
- call services
- return responses

Handlers should NOT:
- contain business logic
- execute raw SQL

---

## Services

Services:
- orchestrate workflows
- own transactions
- coordinate repositories

---

## Repositories

Repositories:
- use pgx
- use raw SQL
- remain stateless

---

## Middleware

Middleware:
- enriches request context
- should not contain business workflows

---

## Migrations

Use Tern migrations only.

Never:
- mutate schema manually

All schema evolution must be:
- versioned
- reproducible

---

## Error Handling

- return structured JSON errors
- log internal failures
- avoid leaking DB internals

---

## API Style

Use:

/api/v1

JSON response format:

{
  "data": ...
}

Errors:

{
  "error": "message"
}