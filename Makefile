.PHONY: codegen database-attach database-remove database-reset database-run database-stop fmt install lint server-build server-run test

# Lint the code with golangci-lint
lint:
	make fmt; \
	if ! [ -x "$$(command -v golangci-lint)" ]; then \
		docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:latest golangci-lint run -v; \
	else \
		golangci-lint run --config configs/golangci/golangci.yml; \
	fi

# Install requirements
install:
	go mod download
	go mod tidy

# Build the server executable
VERSION=$(shell git describe --tags --always)
LDVERSION=-X 'ctf01d/internal/handler.version=$(VERSION)'
BUILDTIME=$(shell date -R)
LDBUILDTIME=-X 'ctf01d/internal/handler.buildTime=$(BUILDTIME)'
server-build:
	go build -ldflags "$(LDVERSION) $(LDBUILDTIME)" -o server ./cmd/main.go

# Run the local development server
server-run:
	go run cmd/main.go

# Format go files
fmt:
	go fmt ./internal/...; \
	go fmt ./cmd/...;

docker-compose-up-build:
	docker compose -f build/docker-compose.yml up --build

# Run PostgreSQL container for local development
database-run:
	@if [ $$(docker ps -a -q -f name=ctf01d-postgres) ]; then \
		echo "Container ctf01d-postgres already exists. Restarting..."; \
		docker start ctf01d-postgres; \
	else \
		echo "Creating and starting container ctf01d-postgres..."; \
		docker run --rm -d \
			--name ctf01d-postgres \
			-e POSTGRES_DB=ctf01d_training_platform \
			-e POSTGRES_USER=postgres \
			-e POSTGRES_PASSWORD=postgres \
			-e PGPORT=4112 \
			-p 4112:4112 postgres:17.4; \
	fi

# Attach to the running PostgreSQL container
database-attach:
	docker exec -it ctf01d-postgres psql -U postgres -d ctf01d_training_platform

# Stop PostgreSQL container
database-stop:
	@if [ $$(docker ps -q -f name=ctf01d-postgres) ]; then \
		echo "Stopping container ctf01d-postgres..."; \
		docker stop ctf01d-postgres; \
	else \
		echo "Container ctf01d-postgres is not running."; \
	fi

# Cleanup db and restart db and rebuild main app
database-reset:
	make stop-db; \
	sudo rm -rf docker_tmp/pg_data; \
	make run-db; \
	make build;

# Remove PostgreSQL container
database-remove:
	@if [ $$(docker ps -a -q -f name=ctf01d-postgres) ]; then \
		echo "Removing container ctf01d-postgres..."; \
		docker rm -f ctf01d-postgres; \
	else \
		echo "Container ctf01d-postgres does not exist."; \
	fi

# Setup test database and run tests
test:
	# Ensure the PostgreSQL container is running
	@if ! [ $$(docker ps -q -f name=ctf01d-postgres) ]; then \
		echo "Starting PostgreSQL container..."; \
		docker start ctf01d-postgres; \
	fi
	# Remove test database if it exists
	@docker exec -it ctf01d-postgres psql -U postgres -c "DROP DATABASE IF EXISTS ctf01d_training_platform_test;"
	# Create a new test database
	@docker exec -it ctf01d-postgres psql -U postgres -c "CREATE DATABASE ctf01d_training_platform_test;"
	# Run the tests
	@go test -v ./test/server_integration_test.go

# Generate Go server boilerplate from OpenAPI 3
code-gen-backend:
	oapi-codegen -generate models,gin -o internal/httpserver/httpserver.gen.go --package httpserver api/openapi.yaml

code-gen-frontend:
	openapi-generator generate \
	-i api/openapi.yaml \
	-g javascript \
	-o static/js/api-client \
	--global-property apis,models,supportingFiles=,apiTests=false,modelTests=false,apiDocs=false,modelDocs=false \
    --additional-properties=useES6=false
