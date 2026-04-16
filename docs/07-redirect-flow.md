# Redirect Flow

## Endpoint

GET /r/{short_code}

---

## Redirect Logic

1. Lookup Redis
2. If miss → query PostgreSQL
3. Cache result
4. update usage stats
5. return redirect

---

## Response Type

Use: 302 Found

For permanent URLs, optional: 301 Moved Permanently

---

## Usage Tracking

Update:

- click_count
- last_accessed_at
