# Cleanup and Retention

## Policy

### Inactive

180 days unused

### Delete

365 days unused

---

## Scheduled Job

Run daily at 02:00

---

## SQL Example

```sql
DELETE FROM short_urls
WHERE last_accessed_at < NOW() - INTERVAL '365 days';
```

---

## Soft Delete

Optional first stage:

```sql
UPDATE short_urls
SET deleted_at = NOW()
WHERE last_accessed_at < NOW() - INTERVAL '180 days';
```
