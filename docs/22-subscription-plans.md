# Subscription Plans

This document defines the **plan tiers, feature entitlements, and enforcement rules** for the subscription system.

It expands [06-subscription-billing.md](06-subscription-billing.md) (which covers Stripe flow) with the product-level definition of what each plan includes.

Goal: one authoritative source for *what each plan can do* so handlers, services, UI, and billing stay aligned.

---

## 1. Plan Tiers

Three tiers. Free is default on signup. Pro and Business are paid (Stripe).

| Tier         | Price (monthly) | Target user                       |
| ------------ | --------------- | --------------------------------- |
| **Free**     | $0              | casual users, evaluation          |
| **Pro**      | $9              | power users, small creators       |
| **Business** | $29             | teams, brands needing own domain  |

Yearly billing: 2 months free (≈17% discount) applied at the Stripe price level.

---

## 2. Feature Matrix

The matrix below is the **source of truth** for entitlements.
Every feature flag in code must map to a row here.

| Feature                          | Free        | Pro         | Business     |
| -------------------------------- | ----------- | ----------- | ------------ |
| Max active URLs                  | 100         | 10,000      | 100,000      |
| Custom short code (alias)        | No          | Yes         | Yes          |
| Custom domain (e.g. `go.acme.com`) | No        | 1 domain    | 5 domains    |
| Link expiration date             | No          | Yes         | Yes          |
| Password-protected links         | No          | Yes         | Yes          |
| Click analytics retention        | 7 days      | 90 days     | 365 days     |
| Geographic / device analytics    | No          | Yes         | Yes          |
| QR code generation               | Basic PNG   | PNG + SVG   | PNG + SVG + branded |
| Bulk URL import (CSV)            | No          | 500/import  | 10,000/import |
| API access                       | No          | Yes         | Yes          |
| API rate limit (req/min)         | —           | 60          | 600          |
| Team members                     | 1           | 1           | 10           |
| Webhook notifications            | No          | No          | Yes          |
| Remove branding on redirect page | No          | Yes         | Yes          |
| Priority support                 | No          | Email       | Email + SLA  |

---

## 3. Feature Details

### 3.1 Custom Short Code (Alias)

Users choose the code instead of auto-generated.

Rules:

- 3–30 chars
- allowed: `[a-zA-Z0-9-_]`
- globally unique across all users
- reserved words blocked (`api`, `admin`, `r`, `login`, `app`, `settings`)

### 3.2 Custom Domain

Business / Pro users can redirect via their own domain.

Flow:

1. user adds domain in dashboard
2. user sets DNS CNAME → `cname.short.ly`
3. system verifies CNAME
4. system issues TLS cert (Let's Encrypt)
5. domain status → `active`
6. `/r/:short_code` now works on custom domain

Domain states:

- `pending_dns`
- `pending_ssl`
- `active`
- `failed`

One short code is globally unique **per domain**, not across all domains. That means `acme.com/promo` and `short.ly/promo` can coexist.

### 3.3 Link Expiration

User sets `expires_at`. After expiry, redirect returns `410 Gone` and link is excluded from analytics aggregation.

### 3.4 Password-Protected Links

Before redirect, user sees a password prompt. Password stored hashed (bcrypt). Attempts rate-limited per IP.

### 3.5 Analytics Retention

Click events older than the plan's retention window are aggregated into daily rollups and raw rows deleted. Downgrades trigger immediate retention re-evaluation via background job (see [10-background-jobs.md](10-background-jobs.md)).

### 3.6 Team Members (Business)

Business workspaces have roles:

- `owner` — billing + all permissions
- `admin` — manage URLs + members
- `member` — create/edit own URLs

Seats counted toward the plan limit; invites over the limit are rejected at the service layer.

### 3.7 API Access

API keys scoped per user (Pro) or per workspace (Business). Rate-limited per key. Keys hashed at rest (only prefix stored for UI display).

### 3.8 Webhooks (Business)

Outbound webhooks for events: `url.created`, `url.clicked`, `url.expired`. HMAC-signed. Retries with exponential backoff up to 24h.

---

## 4. Enforcement

Enforcement happens in the **service layer**, never in handlers.

### 4.1 Entitlement Check

Every feature-gated service method calls:

```go
entitlements.Require(ctx, userID, entitlement.CustomDomain)
```

Returns error code `PLAN_LIMIT_REACHED` or `FEATURE_NOT_AVAILABLE` per [18-error-handling-contract.md](18-error-handling-contract.md).

### 4.2 Limit Check

Quantitative limits (max URLs, max domains, max team members) checked before insert:

```text
count(existing) < plan_limit(user)
```

Use `SELECT ... FOR UPDATE` when the check-then-insert must be atomic (e.g. seat purchase).

### 4.3 Cache

Plan entitlements cached in Redis:

```text
plan:{user_id}
```

TTL: 1 hour. Invalidated on:

- Stripe webhook (upgrade/downgrade/cancel)
- manual admin change

---

## 5. Upgrade / Downgrade Behavior

### Upgrade (Free → Pro, Pro → Business)

- effective immediately
- prorated by Stripe
- cache invalidated on webhook

### Downgrade (Pro → Free, Business → Pro)

- effective at **end of billing period**
- user keeps features until `expires_at`
- at expiry, background job:
  - disables custom domains (status → `suspended`)
  - soft-deletes URLs over new limit (oldest-first by `last_accessed_at`)
  - revokes API keys if new plan lacks API access
  - removes extra team members (demoted, not deleted)

Nothing is hard-deleted on downgrade. Data is reclaimable if user upgrades within 30 days.

### Cancellation

Treated as downgrade to Free.

---

## 6. Database Impact

### 6.1 Applied (migration 0002)

```sql
CREATE TABLE plans (
    code VARCHAR(50) PRIMARY KEY,           -- 'free' | 'pro' | 'business'
    name VARCHAR(100) NOT NULL,
    price_cents INTEGER NOT NULL,
    max_urls INTEGER NOT NULL,
    max_domains INTEGER NOT NULL,
    max_team_members INTEGER NOT NULL,
    analytics_retention_days INTEGER NOT NULL,
    api_rate_limit_per_min INTEGER,         -- NULL = no API access
    features JSONB NOT NULL,                -- boolean flags
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE users
    ADD COLUMN plan_code VARCHAR(50) NOT NULL DEFAULT 'free' REFERENCES plans(code);
CREATE INDEX idx_users_plan_code ON users(plan_code);
```

Free / Pro / Business rows are **seeded** inside the same migration — not
inserted by application code. Changes to a tier's entitlements must go through
a new migration, never a runtime `UPDATE`, so the history is auditable.

The legacy `users.plan_type` column is kept during transition and duplicated to
`plan_code` on register. A later migration will drop it once all reads go
through `PlanRepository`.

Access from Go: [backend/internal/repository/plan_repository.go](../backend/internal/repository/plan_repository.go)
exposes `GetByCode(ctx, code)` and `List(ctx)`. Features come back as
`map[string]bool`; use `plan.Feature("custom_domain")` — unknown flags
return `false` so a typo in caller code fails closed.

### 6.2 Pending (later migrations)

These are specified here but not yet created — they arrive with their
respective features:

```sql
CREATE TABLE custom_domains (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    domain VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL,            -- pending_dns|pending_ssl|active|failed|suspended
    verified_at TIMESTAMP,
    ssl_expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    key_prefix VARCHAR(12) NOT NULL,
    key_hash TEXT NOT NULL,
    last_used_at TIMESTAMP,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);

-- short_urls additions
ALTER TABLE short_urls ADD COLUMN domain_id UUID REFERENCES custom_domains(id);
ALTER TABLE short_urls ADD COLUMN expires_at TIMESTAMP;
ALTER TABLE short_urls ADD COLUMN password_hash TEXT;

CREATE UNIQUE INDEX idx_short_code_per_domain ON short_urls(domain_id, short_code);
```

### 6.3 Who can edit plans

Plan editing is split by field. Not every tweak needs a deploy, but
billing-relevant fields stay in git so the history is reviewable.

| Field                                                 | Editable via          |
| ----------------------------------------------------- | --------------------- |
| `price_cents`                                         | migration only        |
| `max_urls`, `max_domains`, `max_team_members`         | migration only        |
| `analytics_retention_days`, `api_rate_limit_per_min`  | migration only        |
| `code`, `name`                                        | migration only        |
| `features` (JSONB flags)                              | admin UI (audited)    |

Regular users — even on the Business plan — **cannot** edit any plan
fields. Admins (site operators, `role=admin`) can toggle feature flags
via the admin UI; every toggle writes an `admin_audit` row in the same
transaction. Full contract: [25-admin-accounts.md §4](25-admin-accounts.md).

---

## 7. API Additions

New endpoints (detailed specs belong in [04-api-specification.md](04-api-specification.md)):

```text
GET    /plans                       # public, list tiers + entitlements
GET    /subscription/entitlements   # current user's effective entitlements
POST   /domains                     # add custom domain
GET    /domains
DELETE /domains/:id
POST   /domains/:id/verify
POST   /api-keys
DELETE /api-keys/:id
```

All entitlement-gated endpoints return:

```json
{
  "success": false,
  "error": {
    "code": "FEATURE_NOT_AVAILABLE",
    "message": "Custom domains require the Business plan",
    "required_plan": "business"
  }
}
```

---

## 8. Phasing

Plans doc covers full target design. Implementation is staged to avoid bloating MVP:

| Phase   | Scope                                                                 |
| ------- | --------------------------------------------------------------------- |
| **MVP** | Free + Pro, max-URL limit only, Stripe checkout/webhook               |
| **v1.1**| Custom alias, link expiration, password-protected links               |
| **v1.2**| Analytics retention tiers, QR codes, API keys + rate limits           |
| **v2.0**| Business plan: custom domains, team members, webhooks, bulk import    |

Note: [00-overview.md](00-overview.md) lists custom domains as an MVP non-goal — that remains true. This doc defines the target plan design so schema and entitlement code are forward-compatible.

---

## 9. Open Questions

- annual billing UI: single toggle or separate SKUs?
- grandfathering: if prices change, do existing subscribers keep old price?
- region-based pricing (VND, USD)?
- refund policy for mid-cycle downgrades?

These are product decisions, not engineering. Flagged here so they're not forgotten.
