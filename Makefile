.PHONY: build run clean test test-e2e test-e2e-live test-e2e-live-mcp-server test-e2e-live-provider-security test-e2e-live-user-security test-e2e-live-input-validation test-e2e-live-reviewer-auth test-e2e-live-reviewer-fixtures test-all install deps fmt lint
ARGS:= $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

# Variables
BINARY_NAME=blaxel-mcp-server
MAIN_PATH=./cmd/blaxel-mcp-server
BUILD_DIR=./build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run:
	go run $(LDFLAGS) $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

# Run unit tests
test:
	@echo "Running unit tests..."
	go test -v $(shell go list ./... | grep -v /e2e)

# Run e2e tests
test-e2e: build
	@echo "Running local transport/schema e2e tests..."
	@cd e2e && go test -v . -timeout 10m

# Run the strict black-box suite against a dedicated live workspace. TestMain
# loads .env.live without echoing it and rejects an unsafe/incomplete manifest.
test-e2e-live: build
	@echo "Running strict standalone MCP live tests..."
	@BLAXEL_E2E_RUN_PREFIX="$${BLAXEL_E2E_RUN_PREFIX:-mcp-live-$$(date -u +%Y%m%d%H%M%S)-$$$$}" \
		go test -count=1 -v ./e2e/live -timeout 30m

# Focused destructive/security lanes retain the same base manifest and never
# default their explicit acknowledgements or fixture identities.
test-e2e-live-mcp-server: build
	@BLAXEL_E2E_RUN_PREFIX="$${BLAXEL_E2E_RUN_PREFIX:-mcp-live-$$(date -u +%Y%m%d%H%M%S)-$$$$}" \
		go test -count=1 -v ./e2e/live -run '^TestMCPServerWaitForCompletionLifecycle$$' -timeout 15m

test-e2e-live-provider-security: build
	@BLAXEL_E2E_RUN_PREFIX="$${BLAXEL_E2E_RUN_PREFIX:-mcp-live-$$(date -u +%Y%m%d%H%M%S)-$$$$}" \
		go test -count=1 -v ./e2e/live -run '^TestProviderMutationSecurityLane$$' -timeout 15m

test-e2e-live-user-security: build
	@BLAXEL_E2E_RUN_PREFIX="$${BLAXEL_E2E_RUN_PREFIX:-mcp-live-$$(date -u +%Y%m%d%H%M%S)-$$$$}" \
		go test -count=1 -v ./e2e/live -run '^(TestDisposableUserRemovalLifecycle|TestMalformedInviteEmailIsLocallyRejected)$$' -timeout 15m

test-e2e-live-input-validation: build
	@BLAXEL_E2E_RUN_PREFIX="$${BLAXEL_E2E_RUN_PREFIX:-mcp-live-$$(date -u +%Y%m%d%H%M%S)-$$$$}" \
		go test -count=1 -v ./e2e/live -run '^TestInvalidSandboxInputsCreateNothing$$' -timeout 15m

test-e2e-live-reviewer-auth: build
	@BLAXEL_E2E_RUN_PREFIX="$${BLAXEL_E2E_RUN_PREFIX:-mcp-live-$$(date -u +%Y%m%d%H%M%S)-$$$$}" \
		go test -count=1 -v ./e2e/live -run '^TestPM2611APIKeyWorkspaceAuthentication$$' -timeout 5m

test-e2e-live-reviewer-fixtures: build
	@BLAXEL_E2E_RUN_PREFIX="$${BLAXEL_E2E_RUN_PREFIX:-mcp-live-$$(date -u +%Y%m%d%H%M%S)-$$$$}" \
		go test -count=1 -v ./e2e/live -run '^(TestENG3423AgentRequestMapping|TestReviewerJobFixture)$$' -timeout 10m

# Run all non-live tests. Live tests are intentionally explicit and destructive.
test-all: test test-e2e

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Install the binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installation complete"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Multi-platform build complete"

# Development mode with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "air not installed. Run: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Help
help:
	@echo "Available targets:"
	@echo "  make build      - Build the binary"
	@echo "  make run        - Run the application"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make test       - Run unit tests"
	@echo "  make test-e2e   - Run local transport/schema e2e tests"
	@echo "  make test-e2e-live - Run strict standalone live tests (destructive manifest required)"
	@echo "  make test-e2e-live-mcp-server - Run ENG-3887 MCP server lifecycle lane"
	@echo "  make test-e2e-live-provider-security - Run provider-mutation secret lane"
	@echo "  make test-e2e-live-user-security - Run disposable-user security lane"
	@echo "  make test-e2e-live-input-validation - Run ENG-3890 sandbox input validation lane"
	@echo "  make test-e2e-live-reviewer-auth - Run reviewer-confirmed API key auth lane"
	@echo "  make test-e2e-live-reviewer-fixtures - Run stable reviewer echo agent and job"
	@echo "  make deps       - Install dependencies"
	@echo "  make fmt        - Format code"
	@echo "  make lint       - Run linter"
	@echo "  make install    - Install binary to GOPATH/bin"
	@echo "  make build-all  - Build for multiple platforms"
	@echo "  make dev        - Run in development mode with hot reload"
	@echo "  make version    - Show version information"
	@echo "  make help       - Show this help message"

tag:
	git tag -a v$(ARGS) -m "Release v$(ARGS)"
	git push origin v$(ARGS)

%:
	@:
