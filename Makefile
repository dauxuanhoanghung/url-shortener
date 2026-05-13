.PHONY: dev build sqlc-generate sqlc-install create-admin test test-pkg test-cover

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