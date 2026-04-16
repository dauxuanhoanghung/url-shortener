# Database Design

## users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    plan_type VARCHAR(50) NOT NULL DEFAULT 'free',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    status VARCHAR(50),
    expires_at TIMESTAMP,
    created_at TIMESTAMP
);

CREATE TABLE short_urls (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    short_code VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    click_count BIGINT DEFAULT 0,
    last_accessed_at TIMESTAMP,
    created_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_short_code ON short_urls(short_code);
CREATE INDEX idx_last_accessed ON short_urls(last_accessed_at);
CREATE INDEX idx_user_id ON short_urls(user_id);
```
