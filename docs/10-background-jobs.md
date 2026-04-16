# Background Jobs

## Queue Recommendation

Preferred:

- PostgreSQL queue (River)

Alternative:

- Redis queue

---

## Jobs

### CleanupExpiredUrlsJob

Delete inactive URLs

---

### SyncSubscriptionJob

Sync Stripe status

---

### AnalyticsAggregationJob

Future use

---

## Schedule

### Daily

- cleanup
- retry failed webhooks

### Hourly

- subscription sync
