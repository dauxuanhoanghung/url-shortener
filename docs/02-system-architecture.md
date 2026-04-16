# System Architecture

## High-Level Architecture

Browser
↓
VueJS Frontend
↓
Go REST API
↓
PostgreSQL
↓
Redis / Valkey

---

## Services

### 1. Auth Service

Responsible for:

- login
- registration
- token validation

---

### 2. URL Service

Responsible for:

- create short URLs
- delete URLs
- usage validation

---

### 3. Redirect Service

Responsible for:

- short code lookup
- redirect response
- click updates

---

### 4. Billing Service

Responsible for:

- Stripe checkout
- webhook handling
- subscription sync

---

### 5. Worker Service

Responsible for:

- cleanup jobs
- retries
- scheduled tasks

---

## Request Flow

### Create URL

Frontend
→ auth middleware
→ plan limit validation
→ generate short code
→ save DB
→ return response

---

### Redirect

Browser request
→ Redis lookup
→ fallback Postgres
→ update access data
→ HTTP redirect
