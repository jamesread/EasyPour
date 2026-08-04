.PHONY: all generate build test lint install setup dev-service dev-frontend service frontend integration-test

# Default: build project and all subdirectories
all: build

# Generate protocol code (Go + TS)
generate:
	cd protocol/proto && buf generate --template ../buf.gen.yaml

# Build service and frontend
build: service frontend

# Build the Go service
service:
	$(MAKE) -C service build

# Build the frontend
frontend:
	$(MAKE) -C frontend build

# Run all tests
test: generate
	$(MAKE) -C service test
	$(MAKE) -C frontend test

# Run linters
lint:
	$(MAKE) -C service lint
	$(MAKE) -C frontend lint

# Install dependencies
install:
	cd service && go mod download
	cd frontend && npm install

# Setup: install deps and generate protocol
setup: install generate
	@echo "Setup complete! Run 'make dev-service' in one terminal and 'make dev-frontend' in another."

# Run the Go service (development)
dev-service:
	$(MAKE) -C service run

# Run the frontend dev server
dev-frontend:
	$(MAKE) -C frontend run

# Run integration tests (builds service, starts backend with -configdir, runs mocha)
integration-test: service
	cd integration-tests && npm install && node run-tests.js
