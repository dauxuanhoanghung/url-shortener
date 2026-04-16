# Testing Strategy

## Backend Tests

### Unit Tests

Test:

- services
- repositories
- utils

Example:

- URL validation
- short code generator
- plan limit checker

---

### Integration Tests

Test:

- API endpoints
- database transactions
- Redis cache

---

### E2E Tests

Full workflow:

login
→ create URL
→ redirect
→ delete

---

## Frontend Tests

- component tests
- store tests
- route guard tests

---

## Coverage Target

Minimum:

80%
