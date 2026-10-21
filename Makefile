.PHONY: all generate build test lint install setup migrate dev-service dev-frontend service frontend integration-test screenshots

# Default: build project and all subdirectories
all: build

# Generate protocol code (Go + TS)
generate:
	$(MAKE) -wC protocol

# Build service and frontend
build: service frontend

# Build the Go service
service:
	$(MAKE) -wC service build

# Build the frontend
frontend:
	$(MAKE) -wC frontend build

# Apply SQLite migrations (set DB_PATH to the easypour.db file)
migrate:
	$(MAKE) -wC database/sqlite

# Run all tests
test: generate
	$(MAKE) -wC service test
	$(MAKE) -wC frontend test

# Run linters
lint:
	$(MAKE) -wC service lint
	$(MAKE) -wC frontend lint

# Install dependencies
install:
	cd service && go mod download
	cd frontend && npm install

# Setup: install deps and generate protocol
setup: install generate
	@echo "Setup complete! Run 'make migrate' (with DB_PATH set), then 'make dev-service' and 'make dev-frontend'."

# Run the Go service (development)
dev-service:
	$(MAKE) -wC service run

# Run the frontend dev server
dev-frontend:
	$(MAKE) -wC frontend run

# Run integration tests (builds service, starts backend with -configdir, runs mocha)
integration-test: service frontend
	cd integration-tests && npm install && node run-tests.js

# Regenerate marketing screenshots (needs Chrome + chromedriver; not part of `all`)
screenshots:
	$(MAKE) -wC var/marketing/screenshots
