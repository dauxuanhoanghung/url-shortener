# AI Agent Context

This file is intended for Claude / ChatGPT / Copilot agents.

---

## Backend Conventions

- use layered architecture
- handler → service → repository
- keep handlers thin
- business logic in services

---

## Frontend Conventions

- Vue Composition API
- Pinia store
- service layer for API calls

---

## Naming Rules

### Database

snake_case

### JSON

camelCase

### Go

PascalCase for exported structs

---

## API Rules

- RESTful endpoints
- DTO validation
- typed responses

---

## Important Business Rules

- only authenticated users create URLs
- redirect endpoint is public
- enforce subscription limits
- delete inactive URLs automatically
