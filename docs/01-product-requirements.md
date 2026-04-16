# Product Requirements

## Functional Requirements

---

## 1. User Authentication

Only authenticated users can create shortened URLs.

Supported methods:

- email/password
- future OAuth support

---

## 2. URL Creation

Authenticated users can create short URLs.

Input:

- original URL

Output:

- generated short code
- shortened URL

Example:

https://example.com/product/123

↓

https://short.ly/abc123

---

## 3. Redirect

Any public visitor can access:

/{short_code}

System redirects to original URL.

---

## 4. Subscription Plans

### Free Plan

- max URLs: 100

### Pro Plan

- max URLs: 10,000

---

## 5. URL Deletion

Users can manually delete URLs.

System also deletes inactive URLs automatically.

---

## 6. Auto Cleanup

Inactive URLs:

- 180 days → mark inactive
- 365 days → hard delete

---

## Non Functional Requirements

- redirect latency < 100ms (cached)
- API response < 300ms
- high availability
- secure authentication
