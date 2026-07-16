package sandboxes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestDeleteSandboxNotFoundIsActionableToolError(t *testing.T) {
	const name = "missing-sandbox"
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(api.Close)
	cfg := &config.Config{APIEndpoint: api.URL, RunEndpoint: api.URL, Workspace: "test-workspace", Credentials: sdk.Credentials{APIKey: "test-api-key"}}
	handler, err := NewSDKHandler(cfg)
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterSandboxTools(mcpServer, handler, cfg)
	registered := mcpServer.GetTool("delete_sandbox")
	result, err := registered.Handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"name": name}}})
	if err != nil {
		t.Fatalf("delete_sandbox returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("delete_sandbox IsError = false, content = %#v", result.Content)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(strings.ToLower(text), "sandbox") || !strings.Contains(text, name) || !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("delete_sandbox error is not actionable: %q", text)
	}
	if strings.Contains(strings.ToLower(text), "failed with status 404") {
		t.Fatalf("delete_sandbox error uses generic status-only wording: %q", text)
	}
}
