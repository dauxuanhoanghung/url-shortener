# User Account Lifecycle

This document covers the flows around **getting into** and **recovering**
a user account: email verification on signup, and forgot-password reset.

Scope:

- email verification (post-registration)
- forgot-password request + reset
- token storage and cleanup
- enforcement rules (what an unverified user can and cannot do)

Related: [05-authentication.md](05-authentication.md) for the
login/JWT mechanics, [18-error-handling-contract.md](18-error-handling-contract.md)
for response shape, [25-admin-accounts.md](25-admin-accounts.md) for the
separate admin role.

---

## 1. Goals

- Confirm a new user actually owns the email they signed up with,
  without forcing verification before they can see the product.
- Let a user who forgets their password recover access safely.
- Keep token handling uniform across both flows so the cleanup job and
  rate-limiting rules have one shape to reason about.
- Avoid user enumeration: "does this email exist?" must not leak from
  any public endpoint.

---

## 2. Email Verification

### 2.1 Enforcement model — soft gate

After `POST /auth/register`, the user is **logged in immediately** with
a JWT and can explore the dashboard. They have 7 days to verify their
email. After that, protected **mutation** endpoints return
`EMAIL_VERIFICATION_REQUIRED` until the email is confirmed.

| State                     | Can read dashboard | Can create URL | Can upgrade plan |
| ------------------------- | ------------------ | -------------- | ---------------- |
| Unverified, <7d old       | yes                | yes            | no (billing)     |
| Unverified, ≥7d old       | yes                | **no**         | no               |
| Verified                  | yes                | yes            | yes              |

Rationale: hard-gating (can't log in until verified) causes high signup
abandonment. Soft-gating lets the user see value, then nudges them to
verify before they invest. Billing is always hard-gated — no paid plan
without a verified email.

### 2.2 Signup flow

```text
POST /auth/register (email, password)
  ├─ service: hash password, insert user (email_verified_at = NULL)
  ├─ service: insert token (purpose=verify_email, user_id, expires=24h)
  ├─ service: enqueue email job with the one-time link
  ├─ return 201 + JWT (user is logged in immediately)
  └─ email arrives: https://app/verify-email?token=<opaque>
```

### 2.3 Verification endpoint

```text
POST /auth/verify-email
  body: { "token": "<opaque>" }
  ├─ service: hash token, SELECT FROM tokens WHERE token_hash=$1 AND purpose='verify_email'
  ├─ if expired or used → 410 TOKEN_INVALID
  ├─ UPDATE users SET email_verified_at = NOW() WHERE id = $user_id
  ├─ UPDATE tokens SET used_at = NOW()
  └─ 200 { "success": true }
```

### 2.4 Resend

```text
POST /auth/resend-verification  (requires auth; rate-limited)
  ├─ if already verified → 409 ALREADY_VERIFIED
  ├─ invalidate any previous non-used verify_email tokens for this user
  ├─ issue fresh token, send email
  └─ 200 { "success": true }
```

Rate-limit: **1 request per 60s per user**, **5 per hour**. Prevents
using verification emails as an amplification vector.

---

## 3. Forgot Password

### 3.1 Flow

```text
POST /auth/forgot-password
  body: { "email": "user@example.com" }
  ├─ always return 200 (uniform response prevents email enumeration)
  ├─ if user exists:
  │     insert token (purpose=password_reset, expires=30min)
  │     enqueue email with link
  └─ if user doesn't exist: no-op

POST /auth/reset-password
  body: { "token": "<opaque>", "new_password": "..." }
  ├─ hash token, lookup in tokens
  ├─ if expired / used → 410 TOKEN_INVALID
  ├─ UPDATE users SET password_hash = $new
  ├─ UPDATE tokens SET used_at = NOW()
  ├─ invalidate all refresh tokens for user (force re-login on other devices)
  └─ 200 { "success": true }
```

### 3.2 Why uniform response on request

The `forgot-password` endpoint returns the same 200 whether or not the
email exists. If we 404'd for unknown addresses, any attacker could
enumerate which emails have accounts. This is a standard anti-enumeration
pattern.

### 3.3 Password reset invalidates sessions

Resetting the password is implicit evidence that sessions may be
compromised. All refresh tokens for the user are invalidated; the
access token (15min) will expire on its own. User has to log in again
everywhere.

---

## 4. Token Storage

One table backs both flows — same lifecycle, same cleanup, one code
path to reason about.

```sql
CREATE TYPE token_purpose AS ENUM ('verify_email', 'password_reset');

CREATE TABLE tokens (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     token_purpose NOT NULL,
    token_hash  TEXT NOT NULL,                  -- SHA-256 of the opaque token
    expires_at  TIMESTAMP NOT NULL,
    used_at     TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tokens_hash    ON tokens(token_hash);
CREATE INDEX idx_tokens_user    ON tokens(user_id, purpose);
CREATE INDEX idx_tokens_expires ON tokens(expires_at);
```

### 4.1 Token format

- 32 random bytes from `crypto/rand`, base64url-encoded → ~43 chars in
  the URL.
- Stored as SHA-256 hash (`token_hash`). Raw token never hits the DB.
- DB leak → attacker has hashes, cannot use them without brute force
  (32 bytes of entropy is not brute-forceable).

### 4.2 Single use

Every token has `used_at`. Lookups filter `used_at IS NULL AND expires_at > NOW()`.
After a successful verify / reset, `used_at = NOW()`.

### 4.3 TTLs

| Purpose         | TTL    | Rationale                                   |
| --------------- | ------ | ------------------------------------------- |
| verify_email    | 24h    | enough slack for late readers               |
| password_reset  | 30min  | compromise window should be short           |

### 4.4 Cleanup

Background job (see [10-background-jobs.md](10-background-jobs.md)) deletes
rows where `expires_at < NOW() - INTERVAL '7 days'` OR `used_at < NOW() - INTERVAL '7 days'`.
Keeps the 7-day audit window for support without unbounded growth.

---

## 5. Email Delivery

Services do not call providers directly. They call a `Mailer`
interface:

```go
type Mailer interface {
    Send(ctx context.Context, msg Message) error
}
```

Two implementations planned:

| Env  | Implementation                | Notes                                     |
| ---- | ----------------------------- | ----------------------------------------- |
| dev  | console mailer                | prints rendered email + link to stdout    |
| prod | pluggable (SendGrid/Postmark/SES) | picked at v1.1; doesn't block this spec |

Choice of provider is deferred — pluggable interface avoids lock-in.
Dev doesn't need an API key, so onboarding stays one `go run` away.

Delivery is **enqueued**, not synchronous: the HTTP request returns as
soon as the DB write lands, and a background worker picks up the job.
See [10-background-jobs.md](10-background-jobs.md) for the queue
mechanics. Signup must not fail because the email provider is slow.

---

## 6. Enforcement (service layer)

A new middleware / service helper `RequireVerifiedEmail(ctx)` returns
`ErrEmailVerificationRequired` when the current user is past their
grace window and still unverified.

Applied to:

- `POST /urls` (after grace window)
- `POST /subscription/checkout` (always, no grace)
- `POST /domains` (always)
- `POST /api-keys` (always)

Not applied to read endpoints, profile edits, or `/auth/*`.

Error contract (see [18-error-handling-contract.md](18-error-handling-contract.md)):

```json
{
  "success": false,
  "error": {
    "code": "EMAIL_VERIFICATION_REQUIRED",
    "message": "Please verify your email to continue."
  }
}
```

HTTP status: **403**.

---

## 7. Rate Limiting

| Endpoint                        | Limit                        |
| ------------------------------- | ---------------------------- |
| `POST /auth/register`           | 5 / hour / IP                |
| `POST /auth/forgot-password`    | 3 / hour / IP, 5 / day / email |
| `POST /auth/resend-verification`| 1 / 60s / user, 5 / hour / user |
| `POST /auth/reset-password`     | 10 / hour / IP               |
| `POST /auth/verify-email`       | 10 / hour / IP               |

These live in the existing rate-limit middleware with keys like
`ratelimit:auth:forgot:{email}` backed by Redis. See
[11-security.md](11-security.md) for the full rate-limit map.

---

## 8. Database Additions (planned migration 0003)

```sql
ALTER TABLE users
    ADD COLUMN email_verified_at TIMESTAMP;

CREATE TYPE token_purpose AS ENUM ('verify_email', 'password_reset');

CREATE TABLE tokens (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     token_purpose NOT NULL,
    token_hash  TEXT NOT NULL,
    expires_at  TIMESTAMP NOT NULL,
    used_at     TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tokens_hash    ON tokens(token_hash);
CREATE INDEX idx_tokens_user    ON tokens(user_id, purpose);
CREATE INDEX idx_tokens_expires ON tokens(expires_at);
```

Existing users at migration time get `email_verified_at = created_at`
(grandfathered as verified — they were using the product before this
feature existed).

---

## 9. Open Questions

- Branded from-address per environment (`noreply@short.ly` vs
  `noreply@staging.short.ly`)?
- Do we email the user when their password is changed via reset
  (confirmation email, no action required)? Strong yes from a security
  standpoint, defer to v1.1 UX.
- 2FA / TOTP? Out of scope for this spec; track separately.
