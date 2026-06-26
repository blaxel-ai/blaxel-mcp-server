package e2e

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestServerInitialization(t *testing.T) {
	client := NewMCPTestClient(t, TestEnv())
	defer client.Close()

	t.Run("server_capabilities", func(t *testing.T) {
		caps := client.GetServerCapabilities()

		// Check that server supports tools
		if caps.Tools == nil {
			t.Error("Server should support tools")
		}
	})

	t.Run("ping", func(t *testing.T) {
		err := client.Ping()
		if err != nil {
			t.Fatalf("Failed to ping: %v", err)
		}
	})
}

func TestToolsDiscovery(t *testing.T) {
	client := NewMCPTestClient(t, TestEnv())
	defer client.Close()

	t.Run("list_tools", func(t *testing.T) {
		result, err := client.ListTools()
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		// Check for expected tools
		expectedTools := []string{
			"list_agents",
			"get_agent",
			"delete_agent",
			"list_model_apis",
			"create_model_api",
			"delete_model_api",
			"list_mcp_servers",
			"create_mcp_server",
			"delete_mcp_server",
			"list_integrations",
			"get_integration",
			"delete_integration",
		}

		toolNames := make(map[string]bool)
		for _, tool := range result.Tools {
			toolNames[tool.Name] = true
		}

		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("Expected tool %s not found", expected)
			}
		}
	})

	t.Run("tool_schemas", func(t *testing.T) {
		result, err := client.ListTools()
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		// Check create_mcp_server schema
		for _, tool := range result.Tools {
			if tool.Name == "create_mcp_server" {
				// Check the input schema fields directly
				if tool.InputSchema.Type != "object" {
					t.Errorf("Expected create_mcp_server input schema type to be object, got %s", tool.InputSchema.Type)
				}
				if tool.InputSchema.Properties == nil {
					t.Errorf("Expected create_mcp_server to have properties")
				}
				break
			}
		}
	})
}

const hostedAnthropicToolsets = "agents,modelapis,mcpservers,sandboxes,jobs,integrations,users,serviceaccounts,runtime"

func TestAnthropicToolReviewReadiness(t *testing.T) {
	client := NewMCPTestClient(t, TestEnv())
	defer client.Close()

	result, err := client.ListTools()
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	assertAnthropicToolReviewReadiness(t, result)
}

func TestAnthropicHostedToolsetReviewReadiness(t *testing.T) {
	client := NewMCPTestClientWithArgs(t, TestEnv(), "--toolsets", hostedAnthropicToolsets)
	defer client.Close()

	result, err := client.ListTools()
	if err != nil {
		t.Fatalf("Failed to list hosted toolset tools: %v", err)
	}

	assertHostedAnthropicToolReviewReadiness(t, result)
}

func TestHTTPTransportHostedAnthropicToolsetReviewReadiness(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a local HTTP transport test port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to release local HTTP test listener: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serverCtx, stopServer := context.WithCancel(ctx)
	cmd := exec.CommandContext(serverCtx, ServerBinaryPath(t), "--transport", "http", "--http-addr", addr, "--toolsets", hostedAnthropicToolsets)
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start local HTTP MCP server: %v", err)
	}
	defer func() {
		stopServer()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("local HTTP server stdout:\n%s", stdout.String())
			t.Logf("local HTTP server stderr:\n%s", stderr.String())
		}
	}()

	waitForTCPPort(t, ctx, addr)

	client, err := mcpclient.NewStreamableHttpClient("http://" + addr)
	if err != nil {
		t.Fatalf("failed to create streamable HTTP client: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	if err := client.Start(ctx); err != nil {
		t.Fatalf("failed to start streamable HTTP client: %v", err)
	}

	_, err = client.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "HTTP Transport Readiness Test Client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize HTTP MCP client: %v", err)
	}

	result, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("failed to list hosted HTTP tools: %v", err)
	}

	assertHostedAnthropicToolReviewReadiness(t, result)
}

func waitForTCPPort(t *testing.T, ctx context.Context, address string) {
	t.Helper()

	for {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s: %v", address, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func assertHostedAnthropicToolReviewReadiness(t *testing.T, result *mcp.ListToolsResult) {
	t.Helper()

	assertAnthropicToolReviewReadiness(t, result)

	if got, want := len(result.Tools), 41; got != want {
		t.Errorf("hosted Anthropic toolset count changed: got %d, want %d; review the submission tool list before shipping", got, want)
	}

	for _, tool := range result.Tools {
		if strings.HasPrefix(tool.Name, "local_") {
			t.Errorf("hosted Anthropic toolset must not expose local-only tool %s", tool.Name)
		}
	}
}

func assertAnthropicToolReviewReadiness(t *testing.T, result *mcp.ListToolsResult) {
	t.Helper()

	toolsByName := make(map[string]mcp.Tool, len(result.Tools))
	for _, tool := range result.Tools {
		toolsByName[tool.Name] = tool

		if len(tool.Name) > 64 {
			t.Errorf("tool %s exceeds Anthropic 64-character name limit", tool.Name)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Errorf("tool %s is missing a description", tool.Name)
		}
		if strings.TrimSpace(tool.Title) == "" {
			t.Errorf("tool %s is missing top-level title", tool.Name)
		}
		if strings.TrimSpace(tool.Annotations.Title) == "" {
			t.Errorf("tool %s is missing Anthropic-required title annotation", tool.Name)
		}
		if tool.Title != "" && tool.Annotations.Title != "" && tool.Title != tool.Annotations.Title {
			t.Errorf("tool %s top-level title %q does not match annotation title %q", tool.Name, tool.Title, tool.Annotations.Title)
		}
		if tool.Annotations.ReadOnlyHint == nil {
			t.Errorf("tool %s is missing readOnlyHint annotation", tool.Name)
		}
		if tool.Annotations.DestructiveHint == nil {
			t.Errorf("tool %s is missing destructiveHint annotation", tool.Name)
		}
		if tool.Annotations.ReadOnlyHint != nil && tool.Annotations.DestructiveHint != nil && *tool.Annotations.ReadOnlyHint && *tool.Annotations.DestructiveHint {
			t.Errorf("tool %s cannot be both read-only and destructive", tool.Name)
		}
		if toolNameRequiresReadOnlyHint(tool.Name) && (tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint) {
			t.Errorf("tool %s should be annotated read-only for Anthropic auto-permissions", tool.Name)
		}
		if toolNameRequiresDestructiveHint(tool.Name) && (tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint) {
			t.Errorf("tool %s should be annotated destructive because it performs an unsafe action", tool.Name)
		}
		if toolNameRequiresAdditiveHint(tool.Name) && (tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint) {
			t.Errorf("tool %s should be annotated non-destructive because it only performs additive updates", tool.Name)
		}
	}

	if _, ok := toolsByName["run_sandbox"]; ok {
		t.Fatalf("run_sandbox must be split into method-specific tools for Anthropic review")
	}

	for _, name := range []string{
		"run_sandbox_command",
		"list_sandbox_processes",
		"get_sandbox_process",
		"get_sandbox_process_logs",
		"stop_sandbox_process",
		"kill_sandbox_process",
	} {
		tool, ok := toolsByName[name]
		if !ok {
			t.Errorf("expected split sandbox tool %s", name)
			continue
		}
		for _, forbidden := range []string{"method", "path", "body"} {
			if _, exists := tool.InputSchema.Properties[forbidden]; exists {
				t.Errorf("split sandbox tool %s must not expose catch-all %q parameter", name, forbidden)
			}
		}
	}

	runModel, ok := toolsByName["run_model"]
	if !ok {
		t.Fatalf("expected run_model tool")
	}
	if _, exists := runModel.InputSchema.Properties["method"]; exists {
		t.Error("run_model must not expose an HTTP method parameter")
	}
	pathSchema, hasPath := runModel.InputSchema.Properties["path"].(map[string]any)
	if !hasPath {
		t.Fatal("run_model must expose path schema so callers can target supported model API routes")
	}
	pathDescription, _ := pathSchema["description"].(string)
	if !strings.Contains(pathDescription, "https://docs.blaxel.ai") && !strings.Contains(pathDescription, "Blaxel Model API") {
		t.Errorf("run_model path description must reference the target API docs or explicit API name, got %q", pathDescription)
	}

	getWorkspaceUser, ok := toolsByName["get_workspace_user"]
	if !ok {
		t.Fatalf("expected get_workspace_user tool")
	}
	if _, exists := getWorkspaceUser.InputSchema.Properties["name"]; exists {
		t.Error("get_workspace_user must not expose misleading name parameter")
	}
	if _, exists := getWorkspaceUser.InputSchema.Properties["email"]; !exists {
		t.Error("get_workspace_user must expose email parameter")
	}
}

func toolNameRequiresReadOnlyHint(name string) bool {
	return strings.HasPrefix(name, "list_") ||
		strings.HasPrefix(name, "get_") ||
		strings.HasPrefix(name, "local_list_") ||
		name == "local_quick_start_guide"
}

func toolNameRequiresDestructiveHint(name string) bool {
	for _, exactName := range []string{
		"create_integration",
		"create_mcp_server",
		"create_sandbox",
		"update_service_account",
		"update_workspace_user_role",
	} {
		if name == exactName {
			return true
		}
	}
	for _, prefix := range []string{
		"delete_",
		"remove_",
		"run_",
		"stop_",
		"kill_",
		"local_create_",
		"local_deploy_",
		"local_run_",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func toolNameRequiresAdditiveHint(name string) bool {
	for _, exactName := range []string{
		"create_model_api",
		"create_service_account",
		"invite_workspace_user",
	} {
		if name == exactName {
			return true
		}
	}
	return false
}

func TestReadOnlyMode(t *testing.T) {
	env := TestEnv()
	env["BL_READ_ONLY"] = "true"

	client := NewMCPTestClient(t, env)
	defer client.Close()

	result, err := client.ListTools()
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Check that write/runtime tools are not present
	for _, tool := range result.Tools {
		if tool.Name == "create_mcp_server" ||
			tool.Name == "delete_agent" ||
			tool.Name == "create_model_api" ||
			tool.Name == "run_agent" ||
			tool.Name == "run_model" ||
			strings.HasPrefix(tool.Name, "run_sandbox") ||
			strings.HasSuffix(tool.Name, "sandbox_process") ||
			strings.HasSuffix(tool.Name, "sandbox_processes") ||
			strings.HasSuffix(tool.Name, "sandbox_process_logs") {
			t.Errorf("Write/runtime tool %s should not be available in read-only mode", tool.Name)
		}
	}

	// Check that read tools are still present
	hasListAgents := false
	for _, tool := range result.Tools {
		if tool.Name == "list_agents" {
			hasListAgents = true
			break
		}
	}
	if !hasListAgents {
		t.Error("Read tool list_agents should be available in read-only mode")
	}
}
