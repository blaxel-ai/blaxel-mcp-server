package agents

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestDeleteAgentNotFoundIsActionableToolError(t *testing.T) {
	const name = "missing-agent"
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(api.Close)

	cfg := deleteAgentTestConfig(api.URL)
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		t.Fatalf("create SDK client: %v", err)
	}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterAgentTools(mcpServer, NewSDKAgentHandler(sdkClient, false), cfg)

	assertDeleteAgentToolNotFound(t, mcpServer, name)
}

func deleteAgentTestConfig(endpoint string) *config.Config {
	return &config.Config{APIEndpoint: endpoint, RunEndpoint: endpoint, Workspace: "test-workspace", Credentials: sdk.Credentials{APIKey: "test-api-key"}}
}

func assertDeleteAgentToolNotFound(t *testing.T, mcpServer *server.MCPServer, name string) {
	t.Helper()
	registered := mcpServer.GetTool("delete_agent")
	result, err := registered.Handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"name": name}}})
	if err != nil {
		t.Fatalf("delete_agent returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("delete_agent IsError = false, content = %#v", result.Content)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(strings.ToLower(text), "agent") || !strings.Contains(text, name) || !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("delete_agent error is not actionable: %q", text)
	}
	if strings.Contains(strings.ToLower(text), "failed with status 404") {
		t.Fatalf("delete_agent error uses generic status-only wording: %q", text)
	}
}
