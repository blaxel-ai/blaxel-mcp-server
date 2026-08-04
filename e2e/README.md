# E2E Tests for Blaxel MCP Server

This directory contains end-to-end tests for the Blaxel MCP Server, written in Go.

## Running the Tests

### Prerequisites

1. Build the server binary first:
```bash
make build
```

2. Set environment variables (optional, will use test defaults):

**Option A: Using .env file (recommended)**
Create a `.env` file in the project root:
```bash
# .env file in project root
BL_API_KEY=your-api-key
BL_WORKSPACE=your-workspace
```

**Option B: Export environment variables**
```bash
export BL_API_KEY="your-api-key"
export BL_WORKSPACE="your-workspace"
```

The tests will automatically load the `.env` file if it exists.

### Running Tests

Run the local transport and schema e2e tests:
```bash
make test-e2e
```

The older `e2e/tools` package performs real provider and workspace-user
mutations. It is excluded from the default target and fails closed unless a
dedicated allowlisted workspace is configured plus
`BLAXEL_LEGACY_E2E_ALLOW_UNSAFE=true`,
`BLAXEL_LEGACY_E2E_ALLOW_PROVIDER_MUTATIONS=true`,
`BLAXEL_LEGACY_E2E_ALLOW_USER_MUTATIONS=true`, and an approved disposable
no-email identity in `BLAXEL_LEGACY_E2E_USER_EMAIL`. Use `e2e/live` for
supported acceptance coverage.

### Strict standalone live suite

The acceptance suite is isolated in `e2e/live`; `make test-e2e-live` runs only
that package. It builds and launches the real standalone binary over stdio,
drives it with the official `mcp-go` client, and never substitutes an upstream
HTTP server.

1. Copy `.env.live.example` to `.env.live`.
2. Use a dedicated disposable workspace and workspace-scoped credentials.
3. Copy `e2e/live/testdata/fixtures.example.json` to
   `e2e/live/testdata/fixtures.json`, then set the names of disposable
   `mcp-live-*` echo agent and job resources and keep `delete_after_test: true`.
4. Configure every lane-specific fixture and acknowledgement described in
   `.env.live.example`, or invoke one focused Make target below.
5. Run `make test-e2e-live`.

Authentication is explicitly the `BL_API_KEY` + `BL_WORKSPACE` API-key flow.
The PM-2611 reviewer/full lane additionally requires
`BLAXEL_E2E_REVIEWER_API_KEY_CONFIRMED=true` so a developer token is not passed
as an API key by mistake. The child process gets an empty temporary auth home
and no client credentials, never starts WorkOS/browser OAuth or MFA, and scans
that home after exit to prove the API key was not persisted.

The agent fixture must return JSON containing its exact downstream `path` and
parsed request `body`. The suite requires message shorthand to arrive as the
exact top-level body `{"input":"<nonce>","context":{...}}`: no `inputs`
wrapper, no JSON-as-string, and the MCP `context` JSON string must become a
decoded object. For a body request to `/e2e/echo`, the echoed path must match and
the arbitrary nested body must deep-equal the input. The job fixture receives a
valid execution payload with a `tasks` array and must expose the supplied
`marker` in the immediate execution response. The suite gets, runs, deletes,
and independently proves both fixtures absent.

Invocation is fail-closed: `BL_WORKSPACE` must equal
`BLAXEL_E2E_WORKSPACE`, that workspace must explicitly match
`BLAXEL_E2E_WORKSPACE_ALLOWLIST` or `BLAXEL_E2E_WORKSPACE_PREFIX`,
`BLAXEL_E2E_ALLOW_DESTRUCTIVE` must be exactly `true`, `BL_API_KEY` is
required, and every invocation gets a unique `mcp-live-*` prefix. The agent/job manifest is validated by the runtime-fixture lane rather
than `TestMain`, so focused contract, PM-2609, and ENG-3885 runs remain usable.
The full suite has no credential/setup skips. It calls all seven delete tools
with valid nonexistent prefixed identifiers and requires explicit safe not-found
errors; a generic error or a false-success delete fails. It also rejects invalid
sandbox memory (`-100`, `999999999999`) and sibling port values (nonnumeric,
`0`, `70000`), pre-registering cleanup and proving no resource was created.

Lane-specific gates are deliberately separate:

- `make test-e2e-live-mcp-server` (ENG-3887) requires
  `BLAXEL_E2E_MCP_SERVER_INTEGRATION` naming an existing, deployable MCP
  integration. Name-only timeout errors do not count as executable coverage.
  The lane never creates that external fixture. Create and delete use
  `waitForCompletion=true`; immediate get must be terminal
  `DEPLOYED`/`FAILED` after create and actionable not-found (never `DELETING`)
  after delete.
- `make test-e2e-live-provider-security` (ENG-3888) hard-fails without
  `BLAXEL_E2E_ALLOW_PROVIDER_MUTATIONS=true` and
  `BLAXEL_E2E_INTEGRATION_TYPE`. It generates a dummy secret only in memory,
  registers it with the dynamic leak scanner before create, requires masked
  get/list output, never prints the secret, and strictly deletes the integration.
- The real model lifecycle additionally requires an approved process-only
  `OPENAI_API_KEY`. `create_model_api` creates an inline OpenAI integration,
  applies and reads back its string config and the model endpoint name, requires
  `DEPLOYED`, calls `/v1/chat/completions`, and independently deletes/proves both
  resources absent without exposing the key.
- `make test-e2e-live-user-security` (ENG-3421/ENG-3891) hard-fails without
  `BLAXEL_E2E_ALLOW_USER_MUTATIONS=true`, a reserved non-deliverable
  `BLAXEL_E2E_DISPOSABLE_USER_EMAIL`, and an approved existing user email/role.
  It idempotently sets the existing fixture to its current role, then
  invites/gets/lists/removes the disposable invitation and polls to absence.
  Nonexistent removal must be actionable not-found, while malformed invite
  email must fail local clean validation without AWS/SES request IDs, stacks,
  or internal details.
- `make test-e2e-live-input-validation` (ENG-3890) checks invalid sandbox
  memory and ports, registers cleanup before each call, and proves no resource
  survives even while production still silently defaults invalid values.
- `make test-e2e-live-reviewer-auth` (PM-2611) runs the reviewer-confirmed
  API-key/workspace-only tools/list path and verifies no secret persistence.
- `make test-e2e-live-reviewer-fixtures` calls the configured echo agent and job through the exact standalone MCP binary. It verifies request mapping and never deletes either stable reviewer fixture.

The tools/list contract (ENG-3420/PM-2229) requires exactly 41 hosted tools, nonempty titles/descriptions, all annotation pointers, names of at most 64 bytes, and one explicit safe or unsafe operation per tool (no combined method catch-all). List/get tools are read-only and non-destructive. Create, invite, delete, update, remove, run, stop, and kill tools are destructive. The service-account lifecycle proves the one-time secret is returned on creation and both the secret and disposable account are handled safely during testing. `TestExecutableWorkflowCoverage` remains fail-closed and intentionally RED for any discovered tool without a successful public workflow; validation or missing-argument calls never count.

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

The legacy e2e package can run with local configuration. The strict live suite
has no defaults and is intentionally not wired into a GitHub workflow; invoking
its target without the complete destructive manifest is a hard failure.
