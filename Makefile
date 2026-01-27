.PHONY: generate service frontend dev-service dev-frontend

# Generate protocol code
generate:
	cd protocol/proto && buf generate --template ../buf.gen.yaml

# Build the Go service
service:
	cd service && go build -o easypour-service .

# Build the frontend
frontend:
	cd frontend && npm install && npm run build

# Run the Go service
dev-service:
	cd service && go run main.go

# Run the frontend dev server
dev-frontend:
	cd frontend && npm install && npm run dev

# Install dependencies
install:
	cd service && go mod download
	cd frontend && npm install

# Setup everything
setup: install generate
	@echo "Setup complete! Run 'make dev-service' in one terminal and 'make dev-frontend' in another."
