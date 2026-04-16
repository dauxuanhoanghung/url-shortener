# Authentication

## Strategy

Use JWT-based authentication.

---

## Tokens

### Access Token

- expiry: 15 minutes

### Refresh Token

- expiry: 30 days

---

## Flow

Login
→ verify password hash
→ generate JWT
→ return tokens

---

## Protected Routes

- POST /urls
- DELETE /urls/{id}
- subscription routes

---

## Public Routes

- redirect endpoint
- login
- register
