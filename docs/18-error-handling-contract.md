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

- UNAUTHORIZED
- INVALID_TOKEN
- TOKEN_EXPIRED

---

### URL

- INVALID_URL
- SHORT_CODE_NOT_FOUND
- PLAN_LIMIT_REACHED

---

### Billing

- PAYMENT_FAILED
- SUBSCRIPTION_EXPIRED

---

## HTTP Status Mapping

- 400 bad request
- 401 unauthorized
- 403 forbidden
- 404 not found
- 429 rate limit
- 500 internal error
