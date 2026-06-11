CTF01D Training Platform

Go + Gin + PostgreSQL training platform, built around an OpenAPI-first contract with a React+TypeScript SPA.

Architecture
- Backend: Go 1.26, Gin, pgx/v5, sqlc, goose migrations
- Frontend: React + TypeScript (Vite), types generated from OpenAPI spec
- Layers: handler -> service -> repository
- Auth: JWT (bcrypt-compatible with Rails password_digest)

Development
- Requirements: Go 1.26, Node.js 22, PostgreSQL 16, yq v4
- Quick start:
  1. cp .env.sample .env
  2. docker compose -f docker-compose.dev.yml up -d
  3. make migrate-up
  4. make go-run
- Frontend dev: make web-install && make web-dev
- Docs: docs/GO_DEV.md

Useful Make targets
- go-build / go-run / go-test / go-vet / go-fmt
- openapi (merge + codegen + TypeScript types)
- sqlc-gen (generate Go from SQL queries)
- migrate-up / migrate-down / migrate-status
- lint / lint-fix / verify-codegen
- web-install / web-build / web-dev

Production (Docker Compose)
- Requirements: Docker 24+, Docker Compose v2
- Setup:
  1. cp .env.sample .env and set POSTGRES_PASSWORD, JWT_SECRET, ACME_EMAIL
  2. docker compose -f docker-compose.prod.yml up -d --build
- Services:
  - db: PostgreSQL 16 with healthcheck
  - app: Go server with auto-migrations (RUN_MIGRATIONS=true)
  - reverse-proxy: Caddy with auto-HTTPS (Let's Encrypt)
- Migrations run automatically on startup when RUN_MIGRATIONS=true

Legacy Rails code (app/, config/, db/, Gemfile) is preserved for reference.
