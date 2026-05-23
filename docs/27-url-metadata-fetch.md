# URL Metadata Fetching

## 1. Overview

When a user creates a short URL, the system asynchronously fetches metadata
(page title, description, favicon URL, HTTP status) from the original URL.
The result is stored in a new `url_metadata` table linked to `short_urls`.

If the origin server returns a **4xx / network error** (including 404), the
short URL is **soft-deleted** and the user is notified via **Server-Sent Events
(SSE)** so the frontend can update without polling.

---

## 2. Goals

| Goal                    | Details                                                                                               |
| ----------------------- | ----------------------------------------------------------------------------------------------------- |
| Background execution    | Metadata fetch never blocks `POST /urls` — the endpoint returns as soon as the short URL is persisted |
| Metadata storage        | Title, description, og:image, favicon, HTTP status in `url_metadata`                                  |
| Dead-link detection     | HTTP 4xx or network failure → soft-delete `short_urls.deleted_at` + notify                            |
| Real-time client notify | SSE channel per authenticated user; failed-fetch event delivered < 5 s                                |
| No new external dep     | Go stdlib `net/http` for fetching; native SSE via `text/event-stream`                                 |

---

## 3. Database Changes

### New table: `url_metadata`

Migration file: `0006_add_url_metadata.sql`

```sql
CREATE TABLE url_metadata (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url_id       UUID NOT NULL REFERENCES short_urls(id) ON DELETE CASCADE,
    title        TEXT,
    description  TEXT,
    og_image     TEXT,
    favicon_url  TEXT,
    http_status  INTEGER,           -- actual HTTP status from the fetch
    fetch_status VARCHAR(20) NOT NULL DEFAULT 'pending'
                     CHECK (fetch_status IN ('pending', 'ok', 'failed')),
    fetched_at   TIMESTAMP,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_url_metadata_url UNIQUE (url_id)
);

CREATE INDEX idx_url_metadata_url_id ON url_metadata(url_id);
CREATE INDEX idx_url_metadata_fetch_status ON url_metadata(fetch_status)
    WHERE fetch_status = 'pending';
```

### `short_urls` — no new columns

Dead-link deletion reuses the existing `deleted_at` (soft-delete pattern
already in place).

---

## 4. Fetch Logic

### 4.1 What is fetched

`GET <original_url>` with:

- `User-Agent: urlshortener-bot/1.0`
- Timeout: **10 seconds**
- Follow up to 5 redirects
- Read first 64 KB of body to extract `<title>`, `<meta name="description">`,
  `<meta property="og:image">` and `<link rel="icon">` tags via string parsing
  (no HTML parser dep — `strings.Contains` / `bytes.Index` is enough for the
  meta tags we need)

### 4.2 Success path (`2xx`)

1. Parse metadata from body.
2. `UPDATE url_metadata SET title=…, fetch_status='ok', fetched_at=NOW()`.
3. Publish SSE event `metadata_updated` to the owning user's channel (see §6.4).

### 4.3 Failure path (`4xx` or network error)

1. Record `fetch_status='failed'`, `http_status=<code or 0>`.
2. `UPDATE short_urls SET deleted_at=NOW() WHERE id=?`.
3. Invalidate `url:{short_code}` from cache.
4. Publish SSE event to the owning user's channel (see §6).

### 4.4 `3xx` (redirect loop / too many hops)

Treated as failure. Max 5 redirects; beyond that → `fetch_status='failed'`,
`http_status=0`. URL is **not** soft-deleted — redirect loops are a fetch
config issue, not a dead link.

### 4.5 `5xx` (server error)

Treated as a transient error. URL is **not** deleted. `fetch_status='failed'`
recorded, no SSE event. A future retry job (§7) can re-check.

---

## 5. Background Worker

### 5.1 Location

`backend/internal/worker/metadata_worker.go` — thin orchestrator: dequeues jobs, calls the fetcher, updates the DB, fires SSE.

`backend/internal/service/metadata_fetcher.go` — `MetadataFetcher` service: HTTP fetch + HTML parsing, no DB or SSE concerns.

Follows the existing `worker/` package mentioned in [23-backend-architecture.md](23-backend-architecture.md).

### 5.2 Interfaces

```go
// MetadataFetcher (internal/service) — fetches and parses a URL.
type MetadataFetcher interface {
    Fetch(targetURL string) model.FetchResult
}

// MetadataWorker (internal/worker) — job queue orchestrator.
type MetadataWorker interface {
    Submit(job MetadataJob)
    Start(ctx context.Context)
    Stop()
}

type MetadataJob struct {
    MetadataID uuid.UUID
    URLID      uuid.UUID
    ShortCode  string
    UserID     uuid.UUID
    TargetURL  string
}
```

### 5.3 Implementation — buffered channel

```go
type metadataWorker struct {
    jobs       chan MetadataJob
    metaRepo   repository.URLMetadataRepository
    urlRepo    repository.URLRepository
    cache      cache.Cache
    notifier   Notifier          // SSE hub (§6)
    httpClient *http.Client
    workerN    int
}
```

- Channel buffer: `workerN * 20` (default `workerN = 4`).
- `Submit` is non-blocking: if the channel is full, the job is dropped and a
  warn-level log is written. A saturated queue means the system is under
  pressure; the URL will remain `fetch_status='pending'` and the retry job
  picks it up later.
- Workers are started as goroutines in `Start`; `Stop` closes the channel and
  waits on a `sync.WaitGroup`.

### 5.4 Wiring into main

```go
worker := worker.NewMetadataWorker(worker.MetadataWorkerConfig{
    MetaRepo: metaRepo,
    URLRepo:  urlRepo,
    Cache:    appCache,
    Notifier: sseHub,
    Workers:  4,
})
worker.Start(ctx)
defer worker.Stop()

urlService := service.NewURLService(urlRepo, planRepo, worker)
```

---

## 6. Real-Time Notification — SSE

### 6.1 Why SSE over WebSocket

- One-way push (server → client) is all that's needed.
- No extra protocol upgrade; works through Nginx without special config.
- Go stdlib `http.Flusher` is sufficient; no external package.

### 6.2 Endpoint

```text
GET /events
Authorization: Bearer <token>
Accept: text/event-stream
```

- Authenticated (JWT middleware).
- Response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`,
  `Connection: keep-alive`.
- Each connected client is registered in the `SSEHub` keyed by `userID`.

### 6.3 SSEHub interface

```go
// Notifier is the interface MetadataWorker depends on — not the concrete hub.
type Notifier interface {
    Notify(userID uuid.UUID, event SSEEvent)
}

type SSEEvent struct {
    Type string // "url_deleted"
    Data any    // marshaled to JSON
}
```

`SSEHub` in `internal/sse/hub.go` implements `Notifier` and manages the
client registry with a mutex-guarded map of `userID → []chan SSEEvent`.

### 6.4 Event payloads

**`url_deleted`** — fired on 4xx / network failure:

```
event: url_deleted
data: {"url_id":"...","short_code":"abc123","reason":"origin_unreachable","http_status":404}

```

**`metadata_updated`** — fired on successful 2xx fetch:

```
event: metadata_updated
data: {"url_id":"...","short_code":"abc123","title":"Example Domain","description":"...","og_image":"...","favicon_url":"...","fetch_status":"ok"}

```

The frontend patches the URL card in the store immediately on receipt — no `GET /urls` poll needed.

### 6.5 Client disconnect

Handler detects `ctx.Done()` (request cancelled) and unregisters the channel
from `SSEHub` to prevent goroutine leak.

---

## 7. API Changes

### 7.1 `POST /urls` — unchanged response

The response shape is unchanged. The metadata fetch is fire-and-forget from
the handler's perspective:

```go
url, err := s.urlService.Create(ctx, ...)
// service internally calls worker.Submit(...)
return url, nil
```

### 7.2 `GET /urls` — add metadata field

Each URL object gains an optional `metadata` field:

```json
{
  "id": "uuid",
  "short_code": "abc123",
  "short_url": "https://short.ly/abc123",
  "original_url": "https://example.com",
  "metadata": {
    "title": "Example Domain",
    "description": "...",
    "og_image": "https://example.com/og.png",
    "favicon_url": "https://example.com/favicon.ico",
    "fetch_status": "ok"
  }
}
```

`metadata` is `null` while `fetch_status = 'pending'`.

### 7.3 New endpoint: `GET /events`

See §6.2.

---

## 8. Error Codes

New codes added to [18-error-handling-contract.md](18-error-handling-contract.md):

| Code              | HTTP | Meaning                              |
| ----------------- | ---- | ------------------------------------ |
| `URL_UNREACHABLE` | —    | SSE event only — origin returned 4xx |

No new HTTP error codes are introduced on the `POST /urls` or `GET /urls`
paths. The deletion is best-effort and asynchronous.

---

## 9. New Repository

### `URLMetadataRepository`

```go
type URLMetadataRepository interface {
    Create(ctx context.Context, m *model.URLMetadata) (*model.URLMetadata, error)
    GetByURLID(ctx context.Context, urlID uuid.UUID) (*model.URLMetadata, error)
    UpdateFetched(ctx context.Context, id uuid.UUID, result model.FetchResult) error
    ListPending(ctx context.Context, limit int) ([]*model.URLMetadata, error)
}
```

SQL in `internal/repository/queries/url_metadata.sql`.
Adapter in `internal/repository/url_metadata_repository.go`.

---

## 10. Sequence Diagram

```text
Client          Handler        URLService      MetadataWorker  MetadataFetcher  SSEHub
  │                │                │                │               │            │
  │──POST /urls───►│                │                │               │            │
  │                │──Create───────►│                │               │            │
  │                │                │──Submit───────►│               │            │
  │                │◄──URLResponse──│                │               │            │
  │◄──201──────────│                │                │               │            │
  │                │                │    ┌───Fetch(url)─────────────►│            │
  │                │                │    │   (async)  │◄──FetchResult─            │
  │                │                │    │            │                           │
  │                │    on 2xx:     │    │            │                           │
  │                │                │    │──UpdateFetched(ok)        │            │
  │                │                │    │──Notify(metadata_updated)────────────►│
  │◄──SSE event────│────────────────│────│────────────│──────────────────────────│
  │  (card updates)│                │    │            │                           │
  │                │    on 4xx:     │    │            │                           │
  │                │                │    │──soft-delete url          │            │
  │                │                │    │──Notify(url_deleted)─────────────────►│
  │◄──SSE event────│────────────────│────│────────────│──────────────────────────│
  │  (URL removed) │                │    │            │                           │
```

---

## 11. Implementation Plan

| Step | Task                                                                        |
| ---- | --------------------------------------------------------------------------- |
| 1    | Migration `0006_add_url_metadata.sql`                                       |
| 2    | SQL queries + `make sqlc-generate`                                          |
| 3    | `model.URLMetadata`, `model.FetchResult`                                    |
| 4    | `URLMetadataRepository` interface + adapter                                 |
| 5    | `internal/sse/` — `SSEHub`, `Notifier` interface, `GET /events` handler     |
| 6    | `internal/service/metadata_fetcher.go` — `MetadataFetcher` interface + impl  |
| 6b   | `internal/worker/metadata_worker.go` — thin orchestrator, depends on fetcher |
| 7    | `URLService.Create` — call `worker.Submit` after persisting                 |
| 8    | `GET /urls` response DTO — add `metadata` field (join query)                |
| 9    | Wire everything in `cmd/api/main.go`                                        |
| 10   | Frontend — subscribe to SSE, remove deleted URL from store on `url_deleted` |

---

## 12. What Is Explicitly Out of Scope

- HTML parsing library (use stdlib string scanning only)
- Retry queue for failed fetches (planned as a separate background job)
- Metadata for URLs created before this migration (fetch_status stays NULL)
- WebSocket (SSE covers the use case)
- Storing screenshots or full page snapshots
