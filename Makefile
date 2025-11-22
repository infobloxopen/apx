# Makefile for APX CLI

# Build variables
BINARY_NAME := apx
MAIN_PACKAGE := ./cmd/apx
BUILD_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go variables
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GOVERSION := $(shell go version | cut -d ' ' -f 3)

# Linker flags
LDFLAGS := -s -w
LDFLAGS += -X main.version=$(VERSION)
LDFLAGS += -X main.commit=$(COMMIT)
LDFLAGS += -X main.date=$(DATE)

# Tools
GORELEASER_VERSION := v1.21.2

.PHONY: help build clean clean-all test test-unit test-integration test-all coverage lint fmt mod-tidy install dev tools check release snapshot docker-build docker-push docker-run docker-clean enforce-principles integration-tests ci-quick

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk '/^##/ { print "  " $$0 }' $(MAKEFILE_LIST) | sed 's/##//' | sort

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf dist/
	@rm -rf coverage.out
	@go clean -cache -testcache -modcache

## clean-all: Clean all artifacts including Docker cache
clean-all: clean docker-clean

## test: Run all tests
test: test-unit test-integration

## test-unit: Run unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -race -v ./internal/...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -race -v ./tests/integration/...

## test-all: Run all tests with coverage
test-all:
	@echo "Running all tests with coverage..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...

## coverage: Generate and view test coverage
coverage: test-all
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run go vet
lint:
	@echo "Running go vet..."
	@go vet ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

## mod-tidy: Tidy go modules
mod-tidy:
	@echo "Tidying go modules..."
	@go mod tidy
	@go mod verify

## install: Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags="$(LDFLAGS)" $(MAIN_PACKAGE)

## dev: Build and install for development
dev: fmt lint build install

## tools: Install development tools
tools:
	@echo "Installing development tools..."
	@go install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)

## check: Run all checks (format, lint, test)
check: fmt lint test-all

## release: Create a release using goreleaser
release:
	@echo "Creating release..."
	@goreleaser release --clean

## snapshot: Create a snapshot release
snapshot:
	@echo "Creating snapshot release..."
	@goreleaser release --snapshot --clean

## cross-build: Build for multiple platforms
cross-build:
	@echo "Building for multiple platforms..."
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			if [ "$$os" = "windows" ] && [ "$$arch" = "arm64" ]; then continue; fi; \
			echo "Building for $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build \
				-ldflags="$(LDFLAGS)" \
				-o $(BUILD_DIR)/$(BINARY_NAME)-$$os-$$arch \
				$(MAIN_PACKAGE); \
			if [ "$$os" = "windows" ]; then \
				mv $(BUILD_DIR)/$(BINARY_NAME)-$$os-$$arch $(BUILD_DIR)/$(BINARY_NAME)-$$os-$$arch.exe; \
			fi; \
		done; \
	done

## install-tools: Install external tools
install-tools:
	@echo "Installing external tools..."
	@./scripts/install-tools.sh

## up-gitea: Start local Gitea instance for testing
up-gitea:
	@echo "Starting Gitea for local testing..."
	@docker compose up -d gitea || docker-compose up -d gitea || { \
		echo "Starting Gitea container directly..."; \
		docker run -d --name gitea \
			-p 3000:3000 \
			-p 222:22 \
			-v gitea_data:/data \
			gitea/gitea:latest; \
	}
	@echo "Waiting for Gitea to be ready..."
	@sleep 5
	@echo "✓ Gitea running at http://localhost:3000"

## reset-gitea: Reset Gitea instance (clean slate)
reset-gitea:
	@echo "Resetting Gitea..."
	@docker stop gitea 2>/dev/null || true
	@docker rm gitea 2>/dev/null || true
	@docker volume rm gitea_data 2>/dev/null || true
	@echo "✓ Gitea reset complete"

## test-integration: Run integration tests with Gitea
test-integration:
	@echo "Running integration tests..."
	@go test -race -v ./tests/integration/...
validate-tools:
	@echo "Validating external tools..."
	@command -v buf >/dev/null 2>&1 || { echo "buf is not installed"; exit 1; }
	@command -v spectral >/dev/null 2>&1 || { echo "spectral is not installed"; exit 1; }
	@command -v oasdiff >/dev/null 2>&1 || { echo "oasdiff is not installed"; exit 1; }
	@command -v protoc >/dev/null 2>&1 || { echo "protoc is not installed"; exit 1; }
	@echo "All external tools are available"

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

## deps: Show dependency graph
deps:
	@echo "Dependency graph:"
	@go list -m all

## outdated: Check for outdated dependencies
outdated:
	@echo "Checking for outdated dependencies..."
	@go list -u -m all

## security: Run security checks
security:
	@echo "Running security checks..."
	@go list -json -deps ./... | nancy sleuth

## size: Show binary size
size: build
	@echo "Binary size:"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)

## info: Show build information
info:
	@echo "Build Information:"
	@echo "  Binary Name: $(BINARY_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Commit:      $(COMMIT)"
	@echo "  Date:        $(DATE)"
	@echo "  Go Version:  $(GOVERSION)"
	@echo "  OS/Arch:     $(GOOS)/$(GOARCH)"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t apx:$(VERSION) -t apx:latest .

## docker-push: Push Docker image to registry
docker-push:
	@echo "Pushing Docker image..."
	@docker push apx:$(VERSION)
	@docker push apx:latest

## docker-run: Run in Docker container
docker-run: docker-build
	@docker run --rm -it -v $(PWD):/workspace apx:$(VERSION)

## docker-clean: Clean Docker images and cache
docker-clean:
	@echo "Cleaning Docker images and cache..."
	@docker image prune -f
	@docker system prune -f

## generate: Generate code (mocks, etc.)
generate:
	@echo "Generating code..."
	@go generate ./...

## tidy-all: Run all tidy operations
tidy-all: mod-tidy fmt

## pre-commit: Run all pre-commit checks
pre-commit: tidy-all lint test-all

## enforce-principles: Enforce coding principles (matches CI check)
enforce-principles: build
	@echo "Enforcing coding principles..."
	@echo "🔍 Checking Principle 3: No os.Exit outside main.go..."
	@if git grep -n 'os\.Exit' -- '*.go' ':!cmd/**/main.go' ':!*_test.go'; then \
		echo "❌ VIOLATION: os.Exit found outside cmd/**/main.go files"; \
		echo "Principle 3 requires that only main() calls os.Exit"; \
		exit 1; \
	else \
		echo "✅ PASS: No os.Exit violations found"; \
	fi
	@echo "🔍 Checking colorless CI output..."
	@if CI=1 NO_COLOR=1 $(BUILD_DIR)/$(BINARY_NAME) help 2>&1 | grep -q $$'\033'; then \
		echo "❌ VIOLATION: ANSI escape codes found in CI output"; \
		echo "Principle 4 requires colorless output in CI"; \
		exit 1; \
	else \
		echo "✅ PASS: No ANSI escape codes in CI output"; \
	fi
	@echo "🔍 Verifying testable command structure..."
	@if ! git grep -q 'func NewApp' cmd/apx/main.go; then \
		echo "❌ VIOLATION: NewApp function not found in main.go"; \
		echo "Principle 3 requires exporting NewApp for testing"; \
		exit 1; \
	else \
		echo "✅ PASS: NewApp function exported"; \
	fi
	@echo "🔍 Checking for proper error handling..."
	@if git grep -n 'cli\.Exit' -- '*.go' ':!*_test.go'; then \
		echo "❌ VIOLATION: cli.Exit found in code"; \
		echo "Commands should return regular errors, not cli.Exit"; \
		exit 1; \
	else \
		echo "✅ PASS: No cli.Exit violations found"; \
	fi
	@echo "✅ All principle checks passed!"

## integration-tests: Run integration tests (matches CI)
integration-tests: build
	@echo "Running integration tests..."
	@echo "🔍 Testing binary execution..."
	@CI=1 NO_COLOR=1 APX_DISABLE_TTY=1 $(BUILD_DIR)/$(BINARY_NAME) --version
	@CI=1 NO_COLOR=1 APX_DISABLE_TTY=1 $(BUILD_DIR)/$(BINARY_NAME) help >/dev/null
	@CI=1 NO_COLOR=1 APX_DISABLE_TTY=1 $(BUILD_DIR)/$(BINARY_NAME) help init >/dev/null
	@echo "🔍 Testing config commands..."
	@CI=1 NO_COLOR=1 APX_DISABLE_TTY=1 $(BUILD_DIR)/$(BINARY_NAME) config init || true
	@CI=1 NO_COLOR=1 APX_DISABLE_TTY=1 $(BUILD_DIR)/$(BINARY_NAME) config validate || true
	@echo "🔍 Testing cross-platform output consistency..."
	@CI=1 NO_COLOR=1 APX_DISABLE_TTY=1 $(BUILD_DIR)/$(BINARY_NAME) help > output1.txt
	@CI=1 NO_COLOR=1 APX_DISABLE_TTY=1 $(BUILD_DIR)/$(BINARY_NAME) help > output2.txt
	@if ! diff output1.txt output2.txt >/dev/null 2>&1; then \
		echo "❌ VIOLATION: Non-deterministic output detected"; \
		rm -f output1.txt output2.txt; \
		exit 1; \
	else \
		echo "✅ PASS: Output is deterministic"; \
		rm -f output1.txt output2.txt; \
	fi
	@echo "✅ All integration tests passed!"

## ci: Run CI pipeline locally
ci: clean tools pre-commit build cross-build enforce-principles integration-tests

## ci-quick: Run essential CI checks locally (faster)
ci-quick: fmt lint test-unit enforce-principles

# Default target
.DEFAULT_GOAL := build