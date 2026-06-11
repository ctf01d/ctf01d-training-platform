# Go Development Guide

## Prerequisites

- Go 1.26+
- Docker & Docker Compose (for local database)
- yq v4 (for OpenAPI merge pipeline)
- Node.js 22+ (for frontend and OpenAPI TypeScript generation)
- golangci-lint (optional, for linting)

## Quick Start

1. Start the development database:

   ```bash
   docker compose -f docker-compose.dev.yml up -d
   ```

2. Copy the sample env file and adjust if needed:

   ```bash
   cp .env.sample .env
   ```

   The defaults work out of the box with `docker-compose.dev.yml`.

3. Build and run the server:

   ```bash
   make go-build
   make go-run
   ```

   Or use `go run ./cmd/server` directly.

4. Verify everything is green:

   ```bash
   make go-test
   make go-vet
   ```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `go-build` | Compile the server binary to `./ctf01d-server` |
| `go-run` | Build and run the server |
| `go-test` | Run all Go tests |
| `go-vet` | Run `go vet` on all packages |
| `go-fmt` | Format code with `gofmt` (and `gofumpt` if installed) |
| `go-tidy` | Run `go mod tidy` |

## Database

The dev database runs on port 5432 (configurable via `GO_DEV_DB_PORT`).

- **Host**: `localhost`
- **Port**: `5432`
- **User**: `postgres`
- **Password**: `postgres`
- **Database**: `ctf01d_development`

Adminer is available at `http://localhost:8081` (configurable via `GO_DEV_ADMINER_PORT`).

## Environment Variables

See `.env.sample` for the full list. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment (`development` / `production`) |
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/ctf01d_development?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | *(empty)* | JWT signing secret (required in production) |
| `JWT_TTL_HOURS` | `24` | JWT token lifetime |
| `LOG_LEVEL` | `info` | Log level (`debug` / `info` / `warn` / `error`) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated CORS origins |
| `STORAGE_DIR` | `./storage` | Local file storage directory |
| `STORAGE_MAX_UPLOAD_BYTES` | `209715200` | Max upload size (200 MiB) |
| `RUN_MIGRATIONS` | `false` | Run DB migrations on startup |

## Integration Tests

Integration tests require a running PostgreSQL database:

```bash
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/ctf01d_test?sslmode=disable go test ./test/integration/...
```

Tests are automatically skipped if `TEST_DATABASE_URL` is not set.

## Adding a New API Endpoint

1. Edit or create a fragment in `api/fragments/`
2. Run `make openapi` (merges fragments, generates Go server interface + TS types)
3. Implement the handler method in `internal/server/handler/`
4. Add SQL queries in `internal/repository/queries/`
5. Run `make sqlc-gen` (generates Go from SQL)
6. Implement service layer in `internal/service/<entity>/`

## Project Structure

```
cmd/server/           - Application entrypoint
cmd/seed/             - Database seeder
cmd/import-rails/     - Rails data import tool
internal/config/      - Configuration loading
internal/server/      - HTTP server setup and handlers
internal/service/     - Business logic layer
internal/repository/  - Database access (sqlc + pgx)
internal/auth/        - JWT and bcrypt helpers
internal/storage/     - File storage abstraction (local)
internal/domain/errs/ - Domain error types
internal/testutil/    - Test helpers
pkg/logger/           - Zap logger wrapper
migrations/           - Goose SQL migrations
api/fragments/        - OpenAPI fragment files
gen/httpserver/       - Generated Go HTTP server code
configs/              - Tool configs (oapi-codegen, sqlc, golangci, spectral)
test/integration/     - Integration tests (require TEST_DATABASE_URL)
deploy/               - Production Caddyfile
web/                  - React + TypeScript SPA (Vite)
tools/                - Tool dependencies (oapi-codegen, sqlc, goose)
```
