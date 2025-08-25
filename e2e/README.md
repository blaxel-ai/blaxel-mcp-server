# E2E Tests for Blaxel MCP Server

This directory contains end-to-end tests for the Blaxel MCP Server, written in Go.

## Running the Tests

### Prerequisites

1. Build the server binary first:
```bash
make build
```

2. Set environment variables (optional, will use test defaults):
```bash
export BLAXEL_API_KEY="your-api-key"
export BLAXEL_WORKSPACE="your-workspace"
```

### Running Tests

Run all e2e tests:
```bash
make test-e2e
```

Run specific test:
```bash
cd e2e
go test -v -run TestServerInitialization
```

Run with coverage:
```bash
cd e2e
go test -v -cover ./...
```

## Test Structure

- **`mcp_test.go`** - Test client implementation and helpers
- **`server_test.go`** - Server initialization and basic functionality tests
- **`tools_test.go`** - Tool-specific tests (create, list, delete operations)

## Test Coverage

The e2e tests cover:

1. **Server Initialization**
   - Protocol version verification
   - Server info validation
   - Ping/pong functionality

2. **Tool Discovery**
   - Tool listing
   - Schema validation
   - Read-only mode verification

3. **Tool Functionality**
   - Create operations (MCP servers, Model APIs)
   - Dual-mode creation (new vs existing integrations)
   - List operations with filtering
   - Delete operations
   - Parameter validation

## Writing New Tests

To add new e2e tests:

1. Create a new test file or add to existing ones
2. Use the `MCPTestClient` from `mcp_test.go`
3. Always initialize the server before calling tools
4. Check both success and error cases

Example:
```go
func TestNewFeature(t *testing.T) {
    client := NewMCPTestClient(t, testEnv())
    defer client.Close()

    // Initialize first
    if _, err := client.SendRequest("initialize", nil); err != nil {
        t.Fatalf("Failed to initialize: %v", err)
    }

    // Test your feature
    params := map[string]interface{}{
        "name": "your_tool",
        "arguments": map[string]interface{}{
            // tool arguments
        },
    }

    result, err := client.SendRequest("tools/call", params)
    // Add assertions...
}
```

## CI Integration

The e2e tests are designed to run in CI environments. They use test defaults when environment variables are not set, making them suitable for automated testing pipelines.
