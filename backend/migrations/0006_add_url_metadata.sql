-- Migration 0006: url_metadata table for background origin-fetch results.
-- One row per short_url. fetch_status tracks the lifecycle: pending → ok | failed.
-- Dead-link detection (4xx) triggers a soft-delete on short_urls; this table
-- stores the evidence (http_status, fetched_at) for auditing and the UI metadata card.

CREATE TABLE url_metadata (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    url_id       UUID        NOT NULL REFERENCES short_urls(id) ON DELETE CASCADE,
    title        TEXT,
    description  TEXT,
    og_image     TEXT,
    favicon_url  TEXT,
    http_status  INTEGER,
    fetch_status VARCHAR(20) NOT NULL DEFAULT 'pending'
                     CHECK (fetch_status IN ('pending', 'ok', 'failed')),
    fetched_at   TIMESTAMP,
    created_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_url_metadata_url UNIQUE (url_id)
);

CREATE INDEX idx_url_metadata_url_id      ON url_metadata(url_id);
CREATE INDEX idx_url_metadata_fetch_status ON url_metadata(fetch_status)
    WHERE fetch_status = 'pending';
