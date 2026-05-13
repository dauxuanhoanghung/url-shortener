# Authentication

## Strategy

JWT-based authentication with separate flows for **users** and **admins**.
Admins have `role=admin` in the JWT claim; users have `role=user`.

Details:

- Account lifecycle (verify email, forgot password): [24-user-account-lifecycle.md](24-user-account-lifecycle.md)
- Admin accounts: [25-admin-accounts.md](25-admin-accounts.md)

---

## Tokens

### User access token

- expiry: 15 minutes
- claims: `sub`, `email`, `role="user"`, `exp`, `iat`

### User refresh token

- expiry: 30 days
- stored server-side (hashed) so reset-password can revoke

### Admin access token

- expiry: 5 minutes (tighter than user; admins are higher-value targets)
- claims: `sub`, `email`, `role="admin"`, `exp`, `iat`

### Admin refresh token

- expiry: 8 hours (must re-login daily even with a valid refresh token)

---

## Flows

### Register (user)

```text
POST /auth/register (email, password)
  → hash password
  → insert user with role='user', email_verified_at=NULL
  → issue verify_email token, enqueue verification email
  → return JWT (user is logged in immediately — soft email gate, see doc 24)
```

### Login (user)

```text
POST /auth/login (email, password)
  → verify password hash (bcrypt)
  → check user not disabled
  → issue access + refresh tokens
```

### Login (admin)

```text
POST /admin/login (email, password)
  → same password check, but require role='admin'
  → shorter-TTL tokens (see above)
  → rate limit: separate bucket from /auth/login
```

### Refresh

```text
POST /auth/refresh (refresh_token)
  → verify refresh token exists and isn't revoked
  → issue new access token (refresh token stays the same until it expires)
```

### Logout

```text
POST /auth/logout
  → revoke the refresh token server-side
  → client discards access token
```

### Verify email / forgot password / reset

See [24-user-account-lifecycle.md](24-user-account-lifecycle.md).

---

## Password storage

bcrypt, cost factor 12. No argon2 yet — adding an argon2 dependency is
not worth it for this scale, and bcrypt at cost 12 is well above any
realistic brute-force threshold.

---

## Protected Routes

User JWT required:

- `POST /urls`, `DELETE /urls/:id`
- `POST /subscription/checkout`, `GET /subscription/status`
- `POST /auth/resend-verification`, `POST /auth/logout`

Admin JWT required (all `/admin/*`, see [25-admin-accounts.md](25-admin-accounts.md)):

- `GET /admin/users`, `GET /admin/users/:id`
- `PATCH /admin/plans/:code/features`
- `GET /admin/audit`

Mutations requiring a **verified** email (past 7-day grace window):

- `POST /urls`
- `POST /subscription/checkout`
- `POST /domains`, `POST /api-keys` (future)

---

## Public Routes

- redirect endpoint (`GET /r/:short_code`)
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/forgot-password`
- `POST /auth/reset-password`
- `POST /auth/verify-email`
- `GET /plans` (list public pricing)
