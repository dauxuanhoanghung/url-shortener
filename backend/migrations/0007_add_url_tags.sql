-- migrate:up
CREATE TABLE url_tags (
  url_id UUID NOT NULL REFERENCES short_urls(id) ON DELETE CASCADE,
  tag    TEXT NOT NULL,
  PRIMARY KEY (url_id, tag)
);

CREATE INDEX idx_url_tags_tag ON url_tags (LOWER(tag));
CREATE INDEX idx_url_tags_url_id ON url_tags (url_id);
