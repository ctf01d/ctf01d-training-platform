# Go Development Guide

## Prerequisites

- Go 1.26+
- Docker & Docker Compose (for local database)
- yq v4 (for OpenAPI merge pipeline)
- Node.js 18+ (for frontend and OpenAPI TypeScript generation)
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

## Project Structure

```
cmd/server/           - Application entrypoint
internal/config/      - Configuration loading
internal/server/      - HTTP server setup and handlers
internal/service/     - Business logic layer
internal/repository/  - Database access (sqlc + pgx)
internal/domain/errs/ - Domain error types
pkg/logger/           - Zap logger wrapper
migrations/           - Goose SQL migrations
api/fragments/        - OpenAPI fragment files
gen/httpserver/       - Generated Go HTTP server code
tools/                - Tool dependencies (oapi-codegen, sqlc, goose)
```
