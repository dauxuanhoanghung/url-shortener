# Caching Strategy

## Redis / Valkey Usage

### Redirect Cache

Key:

```text
url:{short_code}
```

Value:

```json
{
  "original_url": "https://example.com"
}
```

TTL:

24 hours

---

## Subscription Cache

```text
plan:{user_id}
```

TTL: 1 hour

---

## Rate Limit Cache

```text
rate_limit:{ip}
```
