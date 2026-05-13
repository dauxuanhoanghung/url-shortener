.PHONY: dev build sqlc-generate sqlc-install create-admin test test-pkg test-cover \
        migrate migrate-status

dev:
	cd backend && go run cmd/api/main.go & \
	cd frontend && npm run dev

build:
	cd frontend && npm run build
	cd backend && go build -o bin/app .

# Install the sqlc CLI into $GOPATH/bin. Run once per machine.
sqlc-install:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Regenerate backend/internal/repository/sqlc/ from queries/*.sql.
# Uses sqlc from PATH, else falls back to $GOPATH/bin/sqlc.
sqlc-generate:
	cd backend && $$(command -v sqlc || echo $$(go env GOPATH)/bin/sqlc) generate

# ---------------------------------------------------------------------------
# Database migrations
# Migrations are plain SQL files in backend/migrations/, numbered 0001..N.
# They are applied with psql running inside the Docker container so you don't
# need a local psql installation.
#
# Usage:
#   make migrate                  # apply all pending migrations in order
#   make migrate FILE=0003_...sql # apply a single specific file
#   make migrate-status           # list which migrations have been applied
#
# The container name matches docker-compose.yml (urlshortener-postgres).
# Credentials are read from .env if present, otherwise the defaults are used.
# ---------------------------------------------------------------------------

# Load .env so POSTGRES_* vars are available when running make targets.
-include .env
export

POSTGRES_USER     ?= urlshortener
POSTGRES_PASSWORD ?= urlshortener
POSTGRES_DB       ?= urlshortener
PG_CONTAINER      := urlshortener-postgres
MIGRATIONS_DIR    := backend/migrations

# Ensure the schema_migrations tracking table exists in the DB.
_ensure_migrations_table:
	@docker exec -e PGPASSWORD=$(POSTGRES_PASSWORD) $(PG_CONTAINER) \
		psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -q -c \
		"CREATE TABLE IF NOT EXISTS schema_migrations ( \
		    filename TEXT PRIMARY KEY, \
		    applied_at TIMESTAMP NOT NULL DEFAULT NOW() \
		);" 2>/dev/null || true

migrate: _ensure_migrations_table
ifdef FILE
	@echo "Applying single migration: $(FILE)"
	@docker exec -e PGPASSWORD=$(POSTGRES_PASSWORD) $(PG_CONTAINER) \
		psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -q \
		-c "\i /dev/stdin" < $(MIGRATIONS_DIR)/$(FILE)
	@docker exec -e PGPASSWORD=$(POSTGRES_PASSWORD) $(PG_CONTAINER) \
		psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -q \
		-c "INSERT INTO schema_migrations (filename) VALUES ('$(FILE)') ON CONFLICT DO NOTHING;"
	@echo "Done: $(FILE)"
else
	@echo "Applying all pending migrations..."
	@for f in $(sort $(wildcard $(MIGRATIONS_DIR)/[0-9]*.sql)); do \
		name=$$(basename $$f); \
		applied=$$(docker exec -e PGPASSWORD=$(POSTGRES_PASSWORD) $(PG_CONTAINER) \
			psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -tAq \
			-c "SELECT 1 FROM schema_migrations WHERE filename='$$name'" 2>/dev/null); \
		if [ "$$applied" = "1" ]; then \
			echo "  skip  $$name (already applied)"; \
		else \
			echo "  apply $$name"; \
			docker exec -i -e PGPASSWORD=$(POSTGRES_PASSWORD) $(PG_CONTAINER) \
				psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -q < $$f && \
			docker exec -e PGPASSWORD=$(POSTGRES_PASSWORD) $(PG_CONTAINER) \
				psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -q \
				-c "INSERT INTO schema_migrations (filename) VALUES ('$$name') ON CONFLICT DO NOTHING;" || \
			{ echo "  FAILED $$name"; exit 1; }; \
		fi; \
	done
	@echo "All migrations applied."
endif

migrate-status: _ensure_migrations_table
	@echo "Applied migrations:"
	@docker exec -e PGPASSWORD=$(POSTGRES_PASSWORD) $(PG_CONTAINER) \
		psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -q \
		-c "SELECT filename, applied_at FROM schema_migrations ORDER BY filename;"
	@echo ""
	@echo "Files on disk:"
	@ls $(MIGRATIONS_DIR)/[0-9]*.sql | xargs -n1 basename

# Create an admin account. Usage: make create-admin EMAIL=admin@example.com
# Password is read from stdin (prompted interactively). See docs/25-admin-accounts.md.
create-admin:
	@test -n "$(EMAIL)" || (echo "usage: make create-admin EMAIL=<email>"; exit 1)
	cd backend && go run ./cmd/admin create-admin --email $(EMAIL)

# Run all backend tests. See docs/19-testing-strategy.md.
test:
	cd backend && go test ./...

# Run only pkg/ tests (pure, no deps — fastest feedback loop).
test-pkg:
	cd backend && go test ./pkg/...

# Run tests with coverage report, grouped by package.
test-cover:
	cd backend && go test ./... -cover