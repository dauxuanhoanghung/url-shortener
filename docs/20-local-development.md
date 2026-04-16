# Local Development Setup

## Requirements

- Go 1.25+
- PostgreSQL
- Redis / Valkey
- Node 22+
- Docker

---

## Start Services

```bash
docker compose up -d
```

---

## Backend

```bash
cd backend
go mod tidy
go run cmd/api/main.go
```

---

## Frontend

```bash
cd frontend
npm install
npm run dev
```

---

## Ports

- frontend: 5173
- backend: 8080
- postgres: 5432
- redis: 6379
