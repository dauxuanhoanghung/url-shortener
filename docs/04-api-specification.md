# API Specification

## Base URL:

/api/v1

## Auth

### Register

POST /auth/register

Request:

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

### Login

POST /auth/login

---

## URLs

### Create URL

POST /urls
Authorization: Bearer <token>

Request:

```json
{
  "original_url": "https://example.com"
}
```

Response:

```json
{
  "id": "uuid",
  "short_code": "abc123",
  "short_url": "https://short.ly/abc123"
}
```

### List URLs

GET /urls

### Delete URL

DELETE /urls/{id}

---

## Subscription

POST /subscription/checkout
POST /subscription/webhook
GET /subscription/status

---

## Redirect
