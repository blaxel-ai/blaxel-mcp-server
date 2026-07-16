package mcpservers

import (
	"context"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestHostedMCPServerLifecycleDefaultsToAsync(t *testing.T) {
	handler := &recordingMCPServerHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterMCPServerTools(mcpServer, handler, &config.Config{AsyncLifecycleOnly: true})

	for _, test := range []struct {
		tool string
		args map[string]any
	}{
		{tool: "create_mcp_server", args: map[string]any{"name": "async-mcp"}},
		{tool: "delete_mcp_server", args: map[string]any{"name": "async-mcp"}},
	} {
		result := callMCPServerLifecycleTool(t, mcpServer, test.tool, test.args)
		if result.IsError {
			t.Fatalf("%s returned tool error: %v", test.tool, result.Content)
		}
	}

	if handler.createCalls != 1 || handler.createWait != "false" {
		t.Fatalf("create calls/wait = %d/%q, want 1/false", handler.createCalls, handler.createWait)
	}
	if handler.deleteCalls != 1 || handler.deleteWait != "false" {
		t.Fatalf("delete calls/wait = %d/%q, want 1/false", handler.deleteCalls, handler.deleteWait)
	}

	for _, toolName := range []string{"create_mcp_server", "delete_mcp_server"} {
		property := mcpServer.GetTool(toolName).Tool.InputSchema.Properties["waitForCompletion"].(map[string]any)
		if property["default"] != "false" {
			t.Fatalf("%s waitForCompletion default = %#v, want false", toolName, property["default"])
		}
		enum, ok := property["enum"].([]string)
		if !ok || len(enum) != 1 || enum[0] != "false" {
			t.Fatalf("%s waitForCompletion enum = %#v, want [false]", toolName, property["enum"])
		}
	}
}

func TestStandaloneMCPServerLifecycleDefaultsToSynchronous(t *testing.T) {
	handler := &recordingMCPServerHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterMCPServerTools(mcpServer, handler, &config.Config{})

	for _, test := range []struct {
		tool string
		args map[string]any
	}{
		{tool: "create_mcp_server", args: map[string]any{"name": "sync-mcp"}},
		{tool: "delete_mcp_server", args: map[string]any{"name": "sync-mcp"}},
	} {
		result := callMCPServerLifecycleTool(t, mcpServer, test.tool, test.args)
		if result.IsError {
			t.Fatalf("%s returned tool error: %v", test.tool, result.Content)
		}
		property := mcpServer.GetTool(test.tool).Tool.InputSchema.Properties["waitForCompletion"].(map[string]any)
		if property["default"] != "true" {
			t.Fatalf("%s waitForCompletion default = %#v, want true", test.tool, property["default"])
		}
		enum := property["enum"].([]string)
		if len(enum) != 2 || enum[0] != "true" || enum[1] != "false" {
			t.Fatalf("%s waitForCompletion enum = %#v, want [true false]", test.tool, enum)
		}
	}
	if handler.createWait != "true" || handler.deleteWait != "true" {
		t.Fatalf("standalone omitted waits = create %q/delete %q, want true/true", handler.createWait, handler.deleteWait)
	}
}

func TestHostedMCPServerLifecycleRejectsUnsupportedWaitBeforeHandler(t *testing.T) {
	handler := &recordingMCPServerHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterMCPServerTools(mcpServer, handler, &config.Config{AsyncLifecycleOnly: true})

	for _, wait := range []string{"true", "TRUE", "eventually"} {
		for _, test := range []struct {
			tool       string
			wantStatus string
		}{
			{tool: "create_mcp_server", wantStatus: "DEPLOYED or FAILED"},
			{tool: "delete_mcp_server", wantStatus: "not found"},
		} {
			result := callMCPServerLifecycleTool(t, mcpServer, test.tool, map[string]any{
				"name": "sync-mcp", "waitForCompletion": wait,
			})
			text := mcpServerToolResultText(result)
			if !result.IsError || !strings.Contains(text, "Use false or omit") || !strings.Contains(text, "get_mcp_server") || !strings.Contains(text, test.wantStatus) {
				t.Fatalf("%s wait=%q error is not actionable: %q", test.tool, wait, text)
			}
		}
	}
	if handler.createCalls != 0 || handler.deleteCalls != 0 {
		t.Fatalf("unsupported wait reached handler: create=%d delete=%d", handler.createCalls, handler.deleteCalls)
	}
}

func callMCPServerLifecycleTool(t *testing.T, mcpServer *server.MCPServer, tool string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := mcpServer.GetTool(tool).Handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
	if err != nil {
		t.Fatalf("%s returned protocol error: %v", tool, err)
	}
	return result
}

func mcpServerToolResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	text, _ := result.Content[0].(mcp.TextContent)
	return text.Text
}

type recordingMCPServerHandler struct {
	createCalls int
	deleteCalls int
	createWait  string
	deleteWait  string
}

func (*recordingMCPServerHandler) ListMCPServers(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (*recordingMCPServerHandler) GetMCPServer(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *recordingMCPServerHandler) CreateMCPServer(_ context.Context, _, _, _, wait string, _ map[string]string, _ map[string]string) ([]byte, error) {
	h.createCalls++
	h.createWait = wait
	return []byte(`{"success":true}`), nil
}
func (h *recordingMCPServerHandler) DeleteMCPServer(_ context.Context, _, wait string) ([]byte, error) {
	h.deleteCalls++
	h.deleteWait = wait
	return []byte(`{"success":true}`), nil
}
