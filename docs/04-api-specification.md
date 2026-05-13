# API Specification

## Base URL

```text
/api/v1
```

All routes use the response contract from
[18-error-handling-contract.md](18-error-handling-contract.md).

---

## Auth

### Register

```text
POST /auth/register
```

Request:

```json
{ "email": "user@example.com", "password": "password123" }
```

Response: 201 + JWT. User is logged in immediately; email verification
is soft-gated (see [24-user-account-lifecycle.md](24-user-account-lifecycle.md)).

### Login

```text
POST /auth/login
```

### Refresh

```text
POST /auth/refresh
```

### Logout

```text
POST /auth/logout
```

### Verify Email

```text
POST /auth/verify-email
body: { "token": "<opaque>" }
```

### Resend Verification

```text
POST /auth/resend-verification     (requires auth, rate-limited)
```

### Forgot Password

```text
POST /auth/forgot-password
body: { "email": "user@example.com" }
```

Always returns 200, whether the email exists or not (enumeration
protection — see [11-security.md](11-security.md)).

### Reset Password

```text
POST /auth/reset-password
body: { "token": "<opaque>", "new_password": "..." }
```

Invalidates all refresh tokens on success.

---

## URLs

### Create URL

```text
POST /urls
Authorization: Bearer <user token>
```

Request:

```json
{ "original_url": "https://example.com" }
```

Response:

```json
{
  "id": "uuid",
  "short_code": "abc123",
  "short_url": "https://short.ly/abc123"
}
```

Blocked with `EMAIL_VERIFICATION_REQUIRED` (403) if the caller is past
the 7-day grace window and still unverified.

### List URLs

```text
GET /urls
```

### Delete URL

```text
DELETE /urls/:id
```

---

## Plans (public)

```text
GET /plans
```

Returns all plan tiers with public-facing fields (price, limits, feature
flags). Full matrix in [22-subscription-plans.md §2](22-subscription-plans.md).

---

## Subscription

```text
POST /subscription/checkout     (requires verified email)
POST /subscription/webhook      (Stripe → server)
GET  /subscription/status
GET  /subscription/entitlements (effective entitlements for current user)
```

---

## Redirect (public)

```text
GET /r/:short_code
```

---

## Admin

All routes require `role=admin` JWT. Non-admin → 403 `ADMIN_REQUIRED`.

```text
POST   /admin/login                    separate bucket from /auth/login
GET    /admin/users                    paginated
GET    /admin/users/:id
GET    /admin/plans
PATCH  /admin/plans/:code/features     body: { "webhooks": true }
GET    /admin/audit                    audit log, paginated
```

Admins cannot edit price / capacity fields (`price_cents`, `max_urls`,
etc.) — those require a migration. Only `features` JSONB flags are
writable from the UI. See [25-admin-accounts.md §4](25-admin-accounts.md).

---

## Error Codes (new)

Added in this spec revision:

| Code                          | HTTP | Meaning                                            |
| ----------------------------- | ---- | -------------------------------------------------- |
| `EMAIL_VERIFICATION_REQUIRED` | 403  | past grace window, email not verified              |
| `TOKEN_INVALID`               | 410  | verification/reset token expired or used           |
| `ALREADY_VERIFIED`            | 409  | resend-verification on an already-verified account |
| `ADMIN_REQUIRED`              | 403  | non-admin JWT on `/admin/*`                        |
| `FEATURE_NOT_EDITABLE`        | 400  | admin tried to PATCH a migration-only field        |

Existing error codes remain unchanged. Full list:
[18-error-handling-contract.md](18-error-handling-contract.md).
