# Backend Folder Structure (Go + Gin)

## Recommended Structure

```text
backend/
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── config/
│   ├── database/
│   ├── router/
│   │
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── url_handler.go
│   │   └── billing_handler.go
│   │
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── url_service.go
│   │   └── billing_service.go
│   │
│   ├── repository/
│   │   ├── user_repository.go
│   │   ├── url_repository.go
│   │   └── subscription_repository.go
│   │
│   ├── model/
│   │   ├── user.go
│   │   ├── short_url.go
│   │   └── subscription.go
│   │
│   ├── dto/
│   │   ├── auth_dto.go
│   │   ├── url_dto.go
│   │   └── response_dto.go
│   │
│   ├── middleware/
│   │   ├── auth_middleware.go
│   │   └── rate_limit_middleware.go
│   │
│   ├── cache/
│   └── worker/
│
├── pkg/
│   ├── logger/
│   ├── validator/
│   └── utils/
│
├── migrations/
├── tests/
└── go.mod
```

---

## Layer Responsibilities

### handler

HTTP request/response only

Must NOT contain business logic

---

### service

Core business logic

Examples:

- plan validation
- short code generation
- Stripe workflow

---

### repository

Database access only

Examples:

- insert
- update
- select
- delete

---

### middleware

Cross-cutting concerns

Examples:

- JWT validation
- rate limiting
- logging
- CORS
