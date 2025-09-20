# Makefile for terraform-provider-tofukit

# Variables
PROVIDER_NAME := terraform-provider-tofukit
VERSION := 0.1.0
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
BINARY_NAME := $(PROVIDER_NAME)_v$(VERSION)
BIN_DIR := bin

# Default target
.PHONY: all
all: build

# Build the provider binary into bin/ directory
.PHONY: build
build:
	@echo "Building $(PROVIDER_NAME) to $(BIN_DIR)/..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Install the provider locally for testing with tofu
.PHONY: install
install: build
	@echo "Installing provider for local testing..."
	@mkdir -p ~/.terraform.d/plugins/registry.terraform.io/DimmKirr/tofukit/$(VERSION)/$(GOOS)_$(GOARCH)
	@cp $(BIN_DIR)/$(BINARY_NAME) ~/.terraform.d/plugins/registry.terraform.io/DimmKirr/tofukit/$(VERSION)/$(GOOS)_$(GOARCH)/$(BINARY_NAME)
	@echo "Provider installed to ~/.terraform.d/plugins/"
	@echo "Use provider source: registry.terraform.io/DimmKirr/tofukit"

# Test targets
.PHONY: test test-unit test-integration test-acceptance test-manual test-coverage test-benchmark
test: test-unit ## Run all tests (default: unit tests only)

test-unit: ## Run unit tests with coverage
	@echo "🧪 Running unit tests..."
	go test ./internal/... -v -coverprofile=coverage.out
	@echo "✅ Unit tests completed"

test-unit-short: ## Run unit tests quickly (no coverage)
	@echo "🧪 Running unit tests (short)..."
	go test ./internal/... -short
	@echo "✅ Unit tests completed"

test-integration: ## Run integration tests (requires Claude CLI)
	@echo "🔗 Running integration tests..."
	@echo "⚠️  Note: Requires Claude CLI to be installed and authenticated"
	go test ./internal/claude -v -run TestExecutor_IntegrationTest
	@echo "✅ Integration tests completed"

test-acceptance: ## Run Terraform acceptance tests
	@echo "🏗️  Running acceptance tests..."
	@echo "⚠️  Note: Requires TOFUKIT_ACC=1 environment variable"
	TOFUKIT_ACC=1 go test ./internal/resources -v -run TestAcc
	@echo "✅ Acceptance tests completed"

test-all: ## Run all test types
	@echo "🚀 Running complete test suite..."
	$(MAKE) test-unit
	$(MAKE) test-integration
	$(MAKE) test-acceptance
	@echo "✅ All tests completed"

test-coverage: ## Generate and display test coverage report
	@echo "📊 Generating coverage report..."
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@echo "✅ Coverage report generated"

test-coverage-html: test-coverage ## Generate HTML coverage report and open in browser
	@echo "🌐 Generating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@if command -v open >/dev/null 2>&1; then \
		open coverage.html; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open coverage.html; \
	else \
		echo "📄 Coverage report saved as coverage.html"; \
	fi

test-benchmark: ## Run benchmark tests
	@echo "⚡ Running benchmark tests..."
	go test ./internal/... -bench=. -benchmem
	@echo "✅ Benchmark tests completed"

# Manual testing helpers
.PHONY: test-manual-quick test-manual-hello test-manual-hello-dry test-manual-complex test-manual-validate test-manual-custom
test-manual-quick: ## Run quick manual test
	@echo "🎯 Running quick manual test..."
	go run cmd/test-helper/main.go -type=quick

test-manual-hello: ## Run hello world manual test
	@echo "👋 Running hello world manual test..."
	go run cmd/test-helper/main.go -type=hello -dry-run=false

test-manual-hello-dry: ## Run hello world manual test (dry run)
	@echo "👋 Running hello world manual test (dry run)..."
	go run cmd/test-helper/main.go -type=hello -dry-run=true

test-manual-complex: ## Run complex manual test
	@echo "🏗️  Running complex manual test..."
	go run cmd/test-helper/main.go -type=complex

test-manual-validate: ## Validate Claude CLI setup
	@echo "🔍 Validating Claude CLI setup..."
	go run cmd/test-helper/main.go -type=validate

test-manual-custom: ## Run custom manual test (requires CUSTOM_NAME and CUSTOM_INSTRUCTIONS)
	@echo "🎨 Running custom manual test..."
	@if [ -z "$(CUSTOM_NAME)" ] || [ -z "$(CUSTOM_INSTRUCTIONS)" ]; then \
		echo "❌ Error: CUSTOM_NAME and CUSTOM_INSTRUCTIONS must be set"; \
		echo "Example: make test-manual-custom CUSTOM_NAME=my-project CUSTOM_INSTRUCTIONS='Create a Go web server'"; \
		exit 1; \
	fi
	go run cmd/test-helper/main.go -type=custom -name="$(CUSTOM_NAME)" -instructions="$(CUSTOM_INSTRUCTIONS)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf test-output/
	@echo "Clean complete"

# Format Go code
.PHONY: fmt
fmt:
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Generate documentation
.PHONY: docs
docs:
	go generate ./...

# Development build (with debugging symbols)
.PHONY: dev
dev:
	@echo "Building development version with debug symbols..."
	@mkdir -p $(BIN_DIR)
	go build -gcflags="all=-N -l" -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "Development build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Quick rebuild and test
.PHONY: quick
quick: clean build install
	@echo "Provider rebuilt and installed"

# Variables for custom tests
CUSTOM_NAME ?=
CUSTOM_INSTRUCTIONS ?=

# CI/CD simulation
.PHONY: ci
ci: ## Simulate CI pipeline locally
	@echo "🚀 Running CI pipeline simulation..."
	$(MAKE) fmt
	$(MAKE) lint
	$(MAKE) test-unit
	$(MAKE) build
	@echo "✅ CI pipeline simulation completed"

# Setup helpers
.PHONY: claude-setup terraform-setup
claude-setup: ## Instructions for setting up Claude CLI
	@echo "🤖 Claude CLI Setup Instructions"
	@echo "================================"
	@echo ""
	@echo "1. Install Claude CLI:"
	@echo "   npm install -g @anthropic-ai/claude-code"
	@echo ""
	@echo "2. Authenticate with Claude:"
	@echo "   claude login"
	@echo ""
	@echo "3. Verify installation:"
	@echo "   make test-manual-validate"

terraform-setup: ## Instructions for setting up Terraform development
	@echo "🏗️  Terraform Development Setup"
	@echo "==============================="
	@echo ""
	@echo "1. Install the provider locally:"
	@echo "   make install"
	@echo ""
	@echo "2. Create a test configuration and run:"
	@echo "   terraform init && terraform plan"

.PHONY: help
help:
	@echo "TofuKit Terraform Provider - Development Commands"
	@echo "=================================================="
	@echo ""
	@echo "Build Commands:"
	@echo "  build          - Build the provider binary to bin/ directory"
	@echo "  install        - Build and install the provider for local testing"
	@echo "  dev            - Build with debug symbols"
	@echo "  quick          - Clean, build, and install"
	@echo ""
	@echo "Test Commands:"
	@echo "  test           - Run unit tests (default)"
	@echo "  test-unit      - Run unit tests with coverage"
	@echo "  test-integration - Run integration tests (requires Claude CLI)"
	@echo "  test-acceptance  - Run Terraform acceptance tests"
	@echo "  test-all       - Run all test types"
	@echo "  test-coverage  - Generate coverage report"
	@echo "  test-coverage-html - Generate HTML coverage report"
	@echo "  test-benchmark - Run benchmark tests"
	@echo ""
	@echo "Manual Test Commands:"
	@echo "  test-manual-quick    - Quick hello.txt test"
	@echo "  test-manual-hello    - Hello world test"
	@echo "  test-manual-complex  - Complex multi-requirement test"
	@echo "  test-manual-validate - Validate Claude CLI setup"
	@echo "  test-manual-custom   - Custom test (set CUSTOM_NAME and CUSTOM_INSTRUCTIONS)"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt            - Format Go code"
	@echo "  lint           - Run linter"
	@echo "  docs           - Generate documentation"
	@echo "  clean          - Remove build artifacts"
	@echo ""
	@echo "Setup & CI:"
	@echo "  claude-setup   - Claude CLI setup instructions"
	@echo "  terraform-setup - Terraform development setup"
	@echo "  ci             - Simulate CI pipeline locally"
	@echo ""
	@echo "Examples:"
	@echo "  make test-manual-custom CUSTOM_NAME=web-server CUSTOM_INSTRUCTIONS='Create a Go HTTP server'"
	@echo "  make test-coverage-html   # Generate and open coverage report"
	@echo "  make ci                   # Run full CI pipeline locally"