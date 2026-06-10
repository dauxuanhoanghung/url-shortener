# API Error Handling Contract

All APIs must return a consistent error format.

---

## Response Format

```json
{
  "success": false,
  "error": {
    "code": "PLAN_LIMIT_REACHED",
    "message": "You have reached the maximum number of URLs for your plan"
  }
}
```

---

## Common Error Codes

### Auth

- `UNAUTHORIZED` — missing or unparseable token
- `INVALID_TOKEN` — token signature invalid or expired
- `TOKEN_EXPIRED` — (alias, same 401 response)
- `EMAIL_VERIFICATION_REQUIRED` — past grace period, email not yet verified (403)
- `ADMIN_REQUIRED` — caller is authenticated but is not an admin (403)

---

### Admin

- `USER_NOT_FOUND` — admin user lookup missed (404)
- `PLAN_NOT_FOUND` — admin plan lookup missed (404)
- `CANNOT_DISABLE_SELF` — admin tried to disable their own account (400)

---

### URL

- `INVALID_URL` — scheme not http(s), or blocked scheme (javascript:, data:, file:)
- `SHORT_CODE_NOT_FOUND` — redirect target does not exist
- `URL_NOT_FOUND` — URL lookup by ID returned no result (404)
- `URL_FORBIDDEN` — authenticated user does not own the URL (403)
- `PLAN_LIMIT_REACHED` — user has hit max_urls for their plan
- `TAG_LIMIT_EXCEEDED` — more than 20 tags submitted for a URL (400)

---

### Rate Limiting

- `RATE_LIMIT_EXCEEDED` — too many requests from this IP / user within the window (429)
  Applied to: `POST /auth/register`, `POST /auth/login`, `POST /auth/forgot-password` (10 req/min),
  `POST /urls` (30 req/min)

---

### Billing

- `PAYMENT_FAILED` — (planned)
- `SUBSCRIPTION_EXPIRED` — (planned)

---

### General

- `INTERNAL_ERROR` — unhandled server error or required infrastructure unavailable (e.g. Redis down for rate limiter)

---

## HTTP Status Mapping

| Code | Meaning           | Example codes                      |
| ---- | ----------------- | ---------------------------------- |
| 400  | bad request       | `INVALID_URL`, validation failures |
| 401  | unauthorized      | `UNAUTHORIZED`, `INVALID_TOKEN`    |
| 403  | forbidden         | `EMAIL_VERIFICATION_REQUIRED`      |
| 404  | not found         | `SHORT_CODE_NOT_FOUND`             |
| 429  | too many requests | `RATE_LIMIT_EXCEEDED`              |
| 500  | internal error    | `INTERNAL_ERROR`                   |
