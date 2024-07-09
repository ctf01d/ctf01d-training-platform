.PHONY: lint install build run-server run-db attach-db fuzz-api

# Lint the code with golangci-lint
lint:
	make fmt; \
	if ! [ -x "$$(command -v golangci-lint)" ]; then \
		docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:latest golangci-lint run -v; \
	else \
		golangci-lint run; \
	fi

# Install requirements
install:
	go mod download
	go mod tidy

# Build the server executable
build:
	go build cmd/ctf01d/main.go

# Build the server executable in docker
build-in-docker:
	docker run --rm -v $(PWD):/app -w /app golang:1.22-bookworm go build cmd/ctf01d/main.go

# format go files
fmt:
	go fmt ./internal/...; \
	go fmt ./cmd/...;

# Run the local development server
run-server:
	go run cmd/ctf01d/main.go

# Run PostgreSQL container for local development
run-db:
	@if [ $$(docker ps -a -q -f name=ctf01d-postgres) ]; then \
		echo "Container ctf01d-postgres already exists. Restarting..."; \
		docker start ctf01d-postgres; \
	else \
		echo "Creating and starting container ctf01d-postgres..."; \
		docker run --rm -d \
			-v $(PWD)/docker_tmp/pg_data:/var/lib/postgresql/data/ \
			--name ctf01d-postgres \
			-e POSTGRES_DB=ctf01d_training_platform \
			-e POSTGRES_USER=postgres \
			-e POSTGRES_PASSWORD=postgres \
			-e PGPORT=4112 \
			-p 4112:4112 postgres:16.3; \
	fi

# Stop PostgreSQL container
stop-db:
	@if [ $$(docker ps -q -f name=ctf01d-postgres) ]; then \
		echo "Stopping container ctf01d-postgres..."; \
		docker stop ctf01d-postgres; \
	else \
		echo "Container ctf01d-postgres is not running."; \
	fi

# cleanup db and restart db and rebuild main app
test-updates-db:
	make stop-db; \
	sudo rm -rf docker_tmp/pg_data; \
	make run-db; \
	make build;

# Revome PostgreSQL container
remove-db:
	@if [ $$(docker ps -a -q -f name=ctf01d-postgres) ]; then \
		echo "Removing container ctf01d-postgres..."; \
		docker rm -f ctf01d-postgres; \
	else \
		echo "Container ctf01d-postgres does not exist."; \
	fi

# Attach to the running PostgreSQL container
attach-db:
	docker exec -it ctf01d-postgres psql -U postgres -d ctf01d_training_platform

# Generate Go server boilerplate from OpenAPI 3
server-codegen:
	oapi-codegen -generate models,chi -o internal/app/server/server.gen.go --package server api/openapi.yaml

client-codegen:
	docker run --rm -v ${PWD}/api/openapi.yaml:/local/api/openapi.yaml -v ${PWD}/html/assets/js/generated:/local/html/assets/js/generated openapitools/openapi-generator-cli generate -i /local/api/openapi.yaml -g javascript -o /local/html/assets/js/generated  --global-property apiDocs=false,modelDocs=false,apiTests=false,modelTests=false
