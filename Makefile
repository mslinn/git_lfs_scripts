.PHONY: build test clean install run help version all

BIN_DIR=bin
INSTALL_PATH=/usr/local/bin
VERSION=$(shell cat VERSION)

# All executables to build
COMMANDS=lfst-checksum lfst-import lfst-run lfst-query lfst-scenario lfst-config

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build all binaries
	@echo "Building all commands..."
	@mkdir -p $(BIN_DIR)
	@for cmd in $(COMMANDS); do \
		echo "Building $$cmd..."; \
		go build -ldflags "-X main.version=$(VERSION)" -o $(BIN_DIR)/$$cmd ./cmd/$$cmd || exit 1; \
	done
	@echo "Build complete"

build-checksum: ## Build only lfst-checksum
	@echo "Building lfst-checksum..."
	@mkdir -p $(BIN_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BIN_DIR)/lfst-checksum ./cmd/lfst-checksum
	@echo "Build complete: $(BIN_DIR)/lfst-checksum"

test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -cover ./...

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	go test -race ./...

clean: ## Remove built binaries
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -f lfs-*
	@echo "Clean complete"

install: build ## Install binaries to system path
	@echo "Installing to $(INSTALL_PATH)..."
	@for cmd in $(COMMANDS); do \
		echo "Installing $$cmd..."; \
		sudo cp $(BIN_DIR)/$$cmd $(INSTALL_PATH)/ || exit 1; \
	done
	@echo "Installation complete"

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"

check: fmt vet test ## Run all checks (fmt, vet, test)
	@echo "All checks passed"

version: ## Show the current version
	@echo "Version: $(VERSION)"

all: clean deps check build ## Clean, get deps, run checks, and build
	@echo "All done"
