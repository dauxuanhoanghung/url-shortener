# URL Shortener System Overview

## Project Summary

A SaaS platform that allows authenticated users to create shortened URLs.

The system supports:

- URL redirection
- creator authentication
- subscription plans
- usage limits
- inactive URL cleanup
- scalable caching
- future analytics support

---

## Goals

Primary goals:

- Fast redirect response time
- simple subscription-based monetization
- clear maintainable architecture
- scalable backend services
- AI-agent friendly documentation

---

## Tech Stack

### Backend

- Go
- PostgreSQL
- Redis / Valkey

### Frontend

- Vue 3
- Vite
- Pinia
- Vue Router

### Infrastructure

- Docker
- Nginx
- Stripe

---

## Core Domains

- Authentication
- URL Management
- Redirect Handling
- Subscription Billing
- Cleanup Jobs
- Monitoring

---

## Non-Goals (MVP)

Not included in MVP:

- custom domains
- QR code generation
- advanced analytics
- team collaboration
