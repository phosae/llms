.PHONY: build-wasm serve clean test

# Build WASM module
build-wasm:
	@echo "Building WASM module..."
	@mkdir -p web
	@chmod +x build-wasm.sh
	@./build-wasm.sh

# Serve the web interface
serve: build-wasm
	@echo "Starting development server on http://localhost:8080"
	@cd web && python3 -m http.server 8080 || python -m http.server 8080

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf web/llm-transformers.wasm web/wasm_exec.js

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./transformer/...

# Build for all targets
build-all: build-wasm
	@echo "Building for all targets..."
	@go build -o bin/llm-transformers ./cmd/main.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run || echo "golangci-lint not installed, skipping..."

# Development workflow
dev: clean fmt test build-wasm serve

# Production build
prod: clean fmt test build-all

help:
	@echo "Available targets:"
	@echo "  build-wasm  - Build WASM module"
	@echo "  serve       - Start development server"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  build-all   - Build for all targets"
	@echo "  deps        - Install dependencies"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  dev         - Development workflow"
	@echo "  prod        - Production build"
	@echo "  help        - Show this help"