# Security

## Authentication

- JWT, separate roles for user and admin (see [05-authentication.md](05-authentication.md))
- bcrypt (cost 12) for passwords
- refresh tokens stored server-side (hashed), revocable on password reset / logout
- admin accounts created only via CLI (see [25-admin-accounts.md](25-admin-accounts.md))

---

## Email Verification

Soft-gated — unverified users can read but cannot create URLs, check out,
or buy features past the 7-day grace window. Billing always requires
verified. See [24-user-account-lifecycle.md](24-user-account-lifecycle.md).

---

## Token Handling

All short-lived one-time tokens (email verification, password reset) share
one table with `purpose`. Tokens stored as SHA-256 hashes, never raw.
Details: [24-user-account-lifecycle.md §4](24-user-account-lifecycle.md).

---

## Redirect Validation

Reject:

- `javascript:`
- `data:`
- `file:`
- malformed URLs

---

## Rate Limiting

Applied at middleware level, keyed in Redis.

| Endpoint                         | Limit                             |
| -------------------------------- | --------------------------------- |
| `POST /auth/register`            | 5 / hour / IP                     |
| `POST /auth/login`               | 10 / hour / IP, 5 / hour / email  |
| `POST /admin/login`              | 5 / hour / IP (separate bucket)   |
| `POST /auth/forgot-password`     | 3 / hour / IP, 5 / day / email    |
| `POST /auth/resend-verification` | 1 / 60s / user, 5 / hour / user   |
| `POST /auth/reset-password`      | 10 / hour / IP                    |
| `POST /auth/verify-email`        | 10 / hour / IP                    |
| `POST /urls`                     | 60 / min / user                   |
| `GET /r/:short_code`             | 1000 / min / IP                   |

Rate limiting must use a **shared** store (Redis). Do not use the
in-memory cache fallback here — per-replica counters break the global
limit. See [23-backend-architecture.md §4.7](23-backend-architecture.md).

---

## Enumeration Protection

- `POST /auth/forgot-password` always returns 200, regardless of whether
  the email exists.
- `POST /auth/login` returns a single generic error code for both
  "unknown email" and "wrong password".
- `POST /auth/register` with an existing email returns the same 200 as a
  fresh signup (verification email is simply not sent; no client-facing
  difference). Trade-off: slightly noisy UX ("why didn't I get the
  email?") for users who forgot they already had an account, in exchange
  for closing the enumeration leak.

---

## SQL Safety

Always use prepared statements. sqlc-generated queries use `$N`
parameters; repositories that still use raw pgxpool (none at time of
writing, but if any return) must do the same. Never interpolate into
SQL strings.

---

## Secrets

Store in environment variables only. Config loader in
[backend/internal/config/config.go](../backend/internal/config/config.go)
is the only place that reads `os.Getenv`. Never log a secret; zap
logger should strip known-sensitive fields.

---

## Admin-Specific

- shorter JWT TTL (5 min access, 8 h refresh)
- admin accounts created only via CLI
- every admin write paired with an `admin_audit` row in the same
  transaction
- price / capacity fields on `plans` are **not** admin-editable — must
  go through a migration (auditable in git)

Details: [25-admin-accounts.md](25-admin-accounts.md).

---

## MFA

Deferred to v1.1. When implemented:

- TOTP only (no SMS)
- required for admin, optional for user
- recovery codes stored hashed, single-use
