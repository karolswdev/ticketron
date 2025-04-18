# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOINSTALL=$(GOCMD) install
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVULNCHECK=govulncheck
GOLINT=golangci-lint

# Binary Name
BINARY_NAME=tix
BINARY_DIR=./bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)

# Build flags
# Example: Inject version info using ldflags
# VERSION ?= $(shell git describe --tags --always --dirty)
# LDFLAGS = -ldflags="-X main.version=$(VERSION)"

.PHONY: all build install test test-integration lint fmt vulncheck run clean help

all: help

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_PATH) ./main.go # $(LDFLAGS)

# Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOINSTALL) ./cmd/tix

# Run unit tests
test:
	@echo "Running unit tests..."
	$(GOTEST) -race -cover ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -tags=integration -v ./...

# Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run vulnerability check
vulncheck:
	@echo "Running vulnerability check..."
	$(GOVULNCHECK) ./...

# Build and run with sample args
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH) --version

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)

# Optional: Build with version injection (uncomment LDFLAGS above if needed)
# build-release:
#	@echo "Building release version $(VERSION)..."
#	@mkdir -p $(BINARY_DIR)
#	$(GOBUILD) -o $(BINARY_PATH) $(LDFLAGS) ./cmd/tix

# Show help
help:
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build            Build the $(BINARY_NAME) binary"
	@echo "  install          Install the $(BINARY_NAME) binary"
	@echo "  test             Run unit tests"
	@echo "  test-integration Run integration tests"
	@echo "  lint             Run the linter"
	@echo "  fmt              Format the code"
	@echo "  vulncheck        Run vulnerability check"
	@echo "  run              Build and run $(BINARY_NAME) --version"
	@echo "  clean            Remove build artifacts"
	@echo "  help             Show this help message"
	@echo ""