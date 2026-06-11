.PHONY: codegen database-attach database-remove database-reset database-run database-stop fmt install lint server-build server-run test \
	web-rails-image-build web-rails-image-save web-rails-image-build-and-save deploy \
	go-build go-run go-test go-vet go-fmt go-tidy \
	openapi-merge openapi-codegen openapi-ts openapi openapi-lint
# -----------------------------------------------------------------------------
# Docker images (production)

# Image name can be overridden: make WEB_RAILS_IMAGE=myrepo/web-rails web-rails-image-build
WEB_RAILS_IMAGE ?= ctf01d/web-rails
GIT_TAG := $(shell git describe --tags --always 2>/dev/null || echo dev)

# Build production image for web-rails app
web-rails-image-build:
	docker build -t $(WEB_RAILS_IMAGE):$(GIT_TAG) -f web-rails/Dockerfile web-rails

# Export the built image into a tar file under dist/
web-rails-image-save:
	@mkdir -p dist
	@if ! docker image inspect $(WEB_RAILS_IMAGE):$(GIT_TAG) >/dev/null 2>&1; then \
		echo "Image $(WEB_RAILS_IMAGE):$(GIT_TAG) not found. Run 'make web-rails-image-build' first."; \
		exit 1; \
	fi
	docker image save $(WEB_RAILS_IMAGE):$(GIT_TAG) -o dist/web-rails-$(GIT_TAG).tar
	@echo "Saved to dist/web-rails-$(GIT_TAG).tar"

# Convenience: build then export
web-rails-image-build-and-save: web-rails-image-build web-rails-image-save

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
