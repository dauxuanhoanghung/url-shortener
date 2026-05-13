# Local Development Setup

## Requirements

| Tool | Minimum version | Notes |
| ---- | --------------- | ----- |
| Go | 1.26 | `go version` |
| Node | 22 | `node --version` |
| Docker | any recent | runs Postgres + Redis |

No local Postgres or Redis installation needed — Docker handles both.

---

## 1. First-time setup

```bash
# 1. Copy the example env file and edit as needed
cp .env.example .env

# 2. Start Postgres + Redis in the background
docker compose up -d

# 3. Apply all database migrations
make migrate

# 4. Install backend Go dependencies
cd backend && go mod tidy && cd ..

# 5. Install frontend Node dependencies
cd frontend && npm install && cd ..
```

---

## 2. Running the application

```bash
# Terminal A — backend (Go + Gin on :8080)
cd backend && go run cmd/api/main.go

# Terminal B — frontend (Vite HMR on :5173)
cd frontend && npm run dev
```

Or via the Makefile (runs both; Ctrl-C to stop):

```bash
make dev
```

---

## 3. Migrations

Migrations are plain `.sql` files in `backend/migrations/`, applied in
filename order (`0001_...` → `0002_...` → ...). A `schema_migrations`
table in Postgres tracks which files have been applied so runs are
idempotent.

Migrations run via `docker exec` into the running Postgres container — no
local `psql` needed.

### Apply all pending migrations

```bash
make migrate
```

Output example:

```
  skip  0001_initial_schema.sql (already applied)
  skip  0002_add_plans_table.sql (already applied)
  apply 0003_add_user_lifecycle.sql
  apply 0004_add_admin_accounts.sql
  apply 0005_extract_user_plans.sql
All migrations applied.
```

### Check which migrations have been applied

```bash
make migrate-status
```

### Apply a single file (e.g. when writing a new migration)

```bash
make migrate FILE=0005_extract_user_plans.sql
```

### Run psql directly (for ad-hoc queries)

```bash
docker exec -it urlshortener-postgres \
  psql -U urlshortener -d urlshortener
```

Or with the password inline:

```bash
docker exec -e PGPASSWORD=urlshortener -it urlshortener-postgres \
  psql -U urlshortener -d urlshortener
```

### Writing a new migration

1. Create `backend/migrations/000N_description.sql` (next number in sequence)
2. Write the SQL (additive only — never `DROP` or `ALTER` existing applied migrations)
3. Add the filename to `sqlc.yaml` `schema:` list if it affects tables used by sqlc queries
4. Run `make migrate` then `make sqlc-generate` if the schema changed
5. Fix any compile errors in the repository adapters

**Never modify an applied migration.** If you need to undo something, write a new migration that reverses it.

---

## 4. First admin account

After migrations are applied:

```bash
make create-admin EMAIL=admin@example.com
# → prompts for password (min 12 chars, must include digit + symbol)
```

See [25-admin-accounts.md](25-admin-accounts.md) for the full admin CLI reference.

---

## 5. Ports

| Service | Port | Notes |
| ------- | ---- | ----- |
| Frontend (Vite) | 5173 | `http://localhost:5173` |
| Backend (Gin) | 8080 | `http://localhost:8080` |
| PostgreSQL | 5432 | container name `urlshortener-postgres` |
| Redis | 6379 | container name `urlshortener-redis` |

Override ports in `.env`:

```bash
POSTGRES_PORT=5433
REDIS_PORT=6380
```

---

## 6. Environment variables

Full reference: [`.env.example`](../.env.example). Key vars:

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `SERVER_PORT` | `8080` | Gin listen port |
| `GIN_MODE` | `debug` | `debug` or `release` |
| `BASE_URL` | `http://localhost:8080` | Prefix for generated short URLs |
| `FRONTEND_BASE_URL` | `http://localhost:5173` | Used in verification + reset-password email links |
| `POSTGRES_HOST` | `localhost` | |
| `POSTGRES_PORT` | `5432` | |
| `POSTGRES_USER` | `urlshortener` | |
| `POSTGRES_PASSWORD` | `urlshortener` | |
| `POSTGRES_DB` | `urlshortener` | |
| `REDIS_HOST` | `localhost` | |
| `REDIS_PORT` | `6379` | |
| `JWT_SECRET` | *(empty)* | **Must be set** — at least 32 random chars in prod |
| `STRIPE_SECRET_KEY` | `sk_test_xxx` | Stripe integration (planned) |
| `STRIPE_WEBHOOK_SECRET` | `whsec_xxx` | Stripe integration (planned) |

---

## 7. Makefile reference

```bash
make dev            # run backend + frontend concurrently
make build          # production build (frontend then backend binary)
make migrate        # apply all pending migrations
make migrate FILE=X # apply a single migration file
make migrate-status # show applied vs pending migrations
make sqlc-generate  # regenerate internal/repository/sqlc/ from *.sql queries
make sqlc-install   # install sqlc CLI via go install (once per machine)
make create-admin   # EMAIL=x@y — create an admin account (password via stdin)
make test           # run all backend tests
make test-pkg       # run pkg/ tests only (fastest)
make test-cover     # run all backend tests with coverage per package
```
