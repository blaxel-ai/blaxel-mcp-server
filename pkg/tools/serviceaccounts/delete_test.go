package serviceaccounts

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

func TestDeleteServiceAccountNotFoundIsActionableToolError(t *testing.T) {
	const clientID = "missing-service-account"
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
	RegisterServiceAccountTools(mcpServer, handler, cfg)
	registered := mcpServer.GetTool("delete_service_account")
	result, err := registered.Handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"name": clientID}}})
	if err != nil {
		t.Fatalf("delete_service_account returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("delete_service_account IsError = false, content = %#v", result.Content)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(strings.ToLower(text), "service account") || !strings.Contains(text, clientID) || !strings.Contains(strings.ToLower(text), "not found") {
		t.Fatalf("delete_service_account error is not actionable: %q", text)
	}
	if strings.Contains(strings.ToLower(text), "failed with status 404") {
		t.Fatalf("delete_service_account error uses generic status-only wording: %q", text)
	}
}
