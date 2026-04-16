# Security

## Authentication

- JWT
- secure password hashing (bcrypt / argon2)

---

## Redirect Validation

Reject:

- javascript:
- data:
- malformed URLs

---

## Rate Limiting

Apply on:

- login
- redirect endpoint
- create URL

---

## SQL Safety

Always use prepared statements.

---

## Secrets

Store in environment variables only.
