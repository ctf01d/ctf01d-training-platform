.PHONY: deploy \
	go-build go-run go-test go-vet go-fmt go-tidy \
	openapi-merge openapi-codegen openapi-ts openapi openapi-lint \
	migrate-up migrate-down migrate-status migrate-new \
	database-test-run database-test-stop go-test-e2e \
	sqlc-gen sqlc-vet seed \
	web-install web-build web-gen web-dev \
	lint lint-fix verify-codegen \
	dev dev-up dev-down

# -----------------------------------------------------------------------------
# Environment: load .env if present, then fall back to dev defaults.
# (`-include` ignores a missing file; `?=` only sets when not already defined.)

-include .env

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/ctf01d_development?sslmode=disable
export DATABASE_URL

GO_TEST_DB_PORT ?= 5433
TEST_DATABASE_URL ?= postgres://postgres:postgres@localhost:$(GO_TEST_DB_PORT)/ctf01d_test?sslmode=disable
export TEST_DATABASE_URL

# -----------------------------------------------------------------------------
# Remote deploy helper (rsync with excludes)

DEPLOY_HOST ?= own-vds-france
DEPLOY_TARGET ?= ctf01d-training-platform

RSYNC_EXCLUDES = \
	--exclude .git \
	--exclude .github \
	--exclude .vscode \
	--exclude .idea \
	--exclude .aider.tags.cache.v4 \
	--exclude .cursor-free-vip \
	--exclude .DS_Store \
	--exclude '*.swp' \
	--exclude tmp \
	--exclude log \
	--exclude dist \
	--exclude vendor \
	--exclude node_modules

deploy:
	rsync -az $(RSYNC_EXCLUDES) ./ $(DEPLOY_HOST):$(DEPLOY_TARGET)

# -----------------------------------------------------------------------------
# Go development targets

## go-build: Compile the Go server binary
go-build:
	go build -o ctf01d-server ./cmd/server

## go-run: Build and run the Go server
go-run: go-build
	./ctf01d-server

## go-test: Run all Go tests
go-test:
	go test ./...

## go-test-e2e: Run integration tests against a local test database
go-test-e2e: database-test-run
	go test -v ./test/integration -count=1

## go-vet: Run go vet on all packages
go-vet:
	go vet ./...

## go-fmt: Format Go source files (gofmt + gofumpt if available)
go-fmt:
	gofmt -w .
	@gofumpt -w . 2>/dev/null || true

## go-tidy: Run go mod tidy
go-tidy:
	go mod tidy

# -----------------------------------------------------------------------------
# OpenAPI code generation

OPENAPI_FILE := api/openapi.yaml
OPENAPI_FRAGMENTS_DIR := api/fragments
OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

## openapi-merge: Merge API fragments into a single OpenAPI file
openapi-merge:
	yq eval-all '. as $$item ireduce ({}; . * $$item )' $(OPENAPI_FRAGMENTS_DIR)/*.yaml > $(OPENAPI_FILE)

## openapi-codegen: Generate Go server code from OpenAPI spec
openapi-codegen:
	$(OAPI_CODEGEN) -config configs/oapi-codegen.yaml $(OPENAPI_FILE)

## openapi-ts: Generate TypeScript types from OpenAPI spec
openapi-ts:
	npx openapi-typescript $(OPENAPI_FILE) -o web/src/api/schema.d.ts

## openapi: Full pipeline — merge fragments, generate Go code and TypeScript types
openapi: openapi-merge openapi-codegen openapi-ts

## openapi-lint: Validate OpenAPI specification with Spectral
openapi-lint:
	npx @stoplight/spectral-cli lint $(OPENAPI_FILE) --ruleset configs/spectral.yaml

# -----------------------------------------------------------------------------
# Database migrations (goose)

GOOSE := go run github.com/pressly/goose/v3/cmd/goose
MIGRATIONS_DIR := migrations

## migrate-up: Apply all pending database migrations
migrate-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" up

## migrate-down: Rollback the last database migration
migrate-down:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" down

## migrate-status: Show current migration status
migrate-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" status

## migrate-new: Create a new empty migration (usage: make migrate-new name=add_users)
migrate-new:
	$(GOOSE) -dir $(MIGRATIONS_DIR) create $(name) sql

# -----------------------------------------------------------------------------
# sqlc code generation

SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc

## sqlc-gen: Generate Go code from SQL queries using sqlc
sqlc-gen:
	$(SQLC) generate

## sqlc-vet: Vet SQL queries and generated code
sqlc-vet:
	$(SQLC) vet

## seed: Populate database with test data (idempotent)
seed:
	go run ./cmd/seed

# -----------------------------------------------------------------------------
# Frontend (web/ SPA)

## web-install: Install frontend dependencies
web-install:
	cd web && npm ci

## web-build: Build frontend for production
web-build:
	cd web && npm run build

## web-gen: Regenerate TypeScript API types from OpenAPI spec
web-gen:
	cd web && npm run gen:api

## web-dev: Start frontend dev server with HMR
web-dev:
	cd web && npm run dev

# -----------------------------------------------------------------------------
# Local development orchestration

## dev-up: Start dev infra (Postgres) and apply migrations + seed
dev-up:
	docker compose -f docker-compose.dev.yml up -d
	$(MAKE) migrate-up
	$(MAKE) seed

## dev-down: Stop dev infra
dev-down:
	docker compose -f docker-compose.dev.yml down

## dev: Run backend (:8080) and frontend (:5173) together
dev:
	$(MAKE) -j2 go-run web-dev

## database-test-run: Start local PostgreSQL for e2e tests on GO_TEST_DB_PORT
database-test-run:
	@docker container inspect ctf01d_test_db >/dev/null 2>&1 || { \
		echo "Creating and starting container ctf01d_test_db..."; \
		docker run -d \
			--name ctf01d_test_db \
			-e POSTGRES_DB=ctf01d_test \
			-e POSTGRES_USER=postgres \
			-e POSTGRES_PASSWORD=postgres \
			-p $(GO_TEST_DB_PORT):5432 postgres:16 >/dev/null; \
	}
	@docker container inspect -f '{{.State.Running}}' ctf01d_test_db | grep -q true || { \
		echo "Starting container ctf01d_test_db..."; \
		docker start ctf01d_test_db >/dev/null; \
	}
	@until docker exec ctf01d_test_db pg_isready -U postgres -d ctf01d_test >/dev/null 2>&1; do \
		echo "Waiting for PostgreSQL ctf01d_test_db to be ready..."; \
		sleep 1; \
	done

## database-test-stop: Stop local e2e PostgreSQL
database-test-stop:
	@if [ "$$(docker container inspect -f '{{.State.Running}}' ctf01d_test_db 2>/dev/null)" = "true" ]; then \
		echo "Stopping container ctf01d_test_db..."; \
		docker stop ctf01d_test_db >/dev/null; \
	else \
		echo "Container ctf01d_test_db is not running."; \
	fi

# -----------------------------------------------------------------------------
# Linting and codegen verification

GOLANGCI := golangci-lint

## lint: Run golangci-lint on all Go packages
lint:
	$(GOLANGCI) run --config configs/golangci.yaml ./...

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	$(GOLANGCI) run --config configs/golangci.yaml --fix ./...

## verify-codegen: Regenerate all codegen and check for uncommitted changes
verify-codegen: openapi sqlc-gen
	git diff --exit-code -- gen/ api/openapi.yaml internal/repository/db/ web/src/api/schema.d.ts
