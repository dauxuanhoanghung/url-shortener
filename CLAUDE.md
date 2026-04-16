# CLAUDE.md

This file provides instructions for AI coding agents (Claude, ChatGPT, Cursor, Copilot) working on this repository.

The goal is to ensure generated code is **consistent, safe, maintainable, and aligned with project architecture**.

---

# 1. Project Overview

This repository contains a **SaaS URL shortener platform**.

Core features:

- authenticated users create short URLs
- public redirect endpoint
- subscription-based usage limits
- Stripe billing integration
- automatic cleanup of inactive URLs
- scalable redirect caching

Tech stack:

- Backend: Go + Gin
- Frontend: Vue 3 + Vite + Pinia
- Database: PostgreSQL
- Cache: Redis / Valkey
- Queue: PostgreSQL-based worker preferred
- Payment: Stripe

---

# 2. Architecture Rules

AI must follow **layered architecture**.

Mandatory flow:

Handler → Service → Repository

Never skip layers.

---

## Handler Layer

Location:

```text
internal/handler/
```

Responsibilities:

- parse request
- validate request payload
- call service
- return response

Handlers must remain thin.

Handlers must NOT contain business logic.

---

## Service Layer

Location:

```text
internal/service/
```

Responsibilities:

- business rules
- validation logic
- plan limits
- short code generation
- Stripe workflows
- caching logic coordination

All business logic must live here.

---

## Repository Layer

Location:

```text
internal/repository/
```

Responsibilities:

- SQL queries
- transactions
- persistence logic

Repositories must NOT contain HTTP logic.

Repositories must NOT contain business logic.

---

# 3. Backend Coding Rules (Go + Gin)

---

## Use Interfaces

Always create interfaces for services and repositories.

Example:

```go
type URLService interface {
    CreateShortURL(ctx context.Context, userID string, req dto.CreateURLRequest) (*dto.URLResponse, error)
}
```

This is required for testing and mocking.

---

## Dependency Injection

Always use constructor injection.

Example:

```go
func NewURLHandler(service service.URLService) *URLHandler
```

Never use global variables.

---

## Context Usage

Always pass context.Context through layers.

Required flow:

```text
Gin Context
→ context.Context
→ service
→ repository
```

---

## DTO Separation

Separate:

- request DTO
- response DTO
- database model

Never reuse DB model as API response.

---

## Error Handling

Always use centralized error responses.

Required format:

```json
{
  "success": false,
  "error": {
    "code": "PLAN_LIMIT_REACHED",
    "message": "Plan limit exceeded"
  }
}
```

Never return raw Go errors directly to client.

---

# 4. Database Rules

---

## Naming Convention

Use snake_case.

Example:

```sql
short_urls
last_accessed_at
click_count
```

---

## Migration Safety

Never modify existing migration files after applied.

Always create new migration.

Example:

```text
migrations/0002_add_subscription_table.sql
```

---

## Transactions

Use transactions for:

- billing updates
- subscription changes
- multi-table updates

---

# 5. Business Rules

These rules are mandatory.

---

## URL Creation

Only authenticated users can create URLs.

---

## Plan Limits

Before creating URL:

```text
count(user_urls) < max_plan_limit
```

Free plan:
100

Pro plan:
10000

---

## Redirect

Redirect endpoint is public.

Path:

```text
GET /r/:short_code
```

Required flow:

Redis → Postgres → cache update → redirect

---

## Inactive URL Cleanup

180 days unused:
soft delete

365 days unused:
hard delete

---

# 6. Short Code Rules

Default length:

```text
6 characters
```

Allowed chars:

```text
[a-zA-Z0-9]
```

Must check uniqueness before insert.

Collision handling required.

Retry up to 5 times.

---

# 7. Cache Rules

Redis keys:

```text
url:{short_code}
plan:{user_id}
rate_limit:{ip}
```

Redirect cache TTL:

```text
24 hours
```

Subscription cache TTL:

```text
1 hour
```

---

# 8. Frontend Rules (Vue)

Use:

- Composition API
- Pinia store
- service layer for API

Never call axios directly inside components.

Use:

```text
services/
stores/
views/
components/
```

---

# 9. Security Rules

Must validate URLs.

Reject:

```text
javascript:
data:
file:
```

Always sanitize user input.

JWT required for protected APIs.

Rate limit:

- login
- create URL
- redirect

---

# 10. Testing Rules

Every new feature should include:

- service unit test
- handler integration test

Minimum coverage target:

```text
80%
```

---

# 11. AI Agent Instructions

When making changes:

1. read related docs in `/docs`
2. preserve existing architecture
3. do not introduce new patterns without reason
4. explain major refactor decisions
5. prefer minimal safe changes
6. avoid unnecessary dependency additions

---

## Before Writing Code

AI must first identify:

- layer to modify
- affected business rules
- API impact
- migration impact
- cache impact

---

## Never Do

AI must NOT:

- put SQL inside handlers
- put HTTP logic in repositories
- bypass auth middleware
- expose raw internal errors
- break DTO contracts
- change existing API response format

---

# 12. Preferred Libraries

Backend preferred:

- gin
- pgx
- sqlc (optional)
- redis/go-redis
- stripe-go
- zap logger
- testify

---

# 13. Priority Order

When generating code, prioritize:

1. correctness
2. security
3. maintainability
4. performance
5. developer ergonomics

# Required Context Loading Order

Before generating or modifying code, AI must read these files in order:

1. `docs/00-overview.md`
2. `docs/01-product-requirements.md`
3. `docs/02-system-architecture.md`
4. `docs/03-database-design.md`
5. `docs/04-api-specification.md`
6. `docs/14-ai-agent-context.md`

For backend tasks also read:

- `docs/15-backend-folder-structure.md`
- `docs/18-error-handling-contract.md`

For frontend tasks also read:

- `docs/16-frontend-folder-structure.md`

AI must not write code before reading the relevant docs.
