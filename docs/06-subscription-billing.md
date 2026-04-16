# Subscription Billing

## Provider

Stripe

---

## Plans

### Free

- 100 URLs

### Pro

- 10,000 URLs

---

## Upgrade Flow

User clicks upgrade
→ create Stripe checkout session
→ redirect to Stripe
→ payment success
→ Stripe webhook
→ update subscription table
→ refresh cache

---

## Webhook Events

- checkout.session.completed
- invoice.payment_failed
- customer.subscription.deleted

---

## Validation Rule

Before URL creation:

count(user_urls) < plan_limit
