package modelapis

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestHostedModelAPILifecycleDefaultsToAsync(t *testing.T) {
	handler := &recordingModelAPIHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterModelAPITools(mcpServer, handler, &config.Config{AsyncLifecycleOnly: true})

	for _, test := range []struct {
		tool string
		args map[string]any
	}{
		{tool: "create_model_api", args: map[string]any{"name": "async-model"}},
		{tool: "delete_model_api", args: map[string]any{"name": "async-model"}},
	} {
		result := callModelAPILifecycleTool(t, mcpServer, test.tool, test.args)
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

	for _, toolName := range []string{"create_model_api", "delete_model_api"} {
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

func TestStandaloneModelAPILifecycleDefaultsToSynchronous(t *testing.T) {
	handler := &recordingModelAPIHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterModelAPITools(mcpServer, handler, &config.Config{})

	for _, test := range []struct {
		tool string
		args map[string]any
	}{
		{tool: "create_model_api", args: map[string]any{"name": "sync-model"}},
		{tool: "delete_model_api", args: map[string]any{"name": "sync-model"}},
	} {
		result := callModelAPILifecycleTool(t, mcpServer, test.tool, test.args)
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

func TestHostedModelAPILifecycleRejectsUnsupportedWaitBeforeHandler(t *testing.T) {
	handler := &recordingModelAPIHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterModelAPITools(mcpServer, handler, &config.Config{AsyncLifecycleOnly: true})

	for _, wait := range []string{"true", "TRUE", "eventually"} {
		for _, test := range []struct {
			tool       string
			wantStatus string
		}{
			{tool: "create_model_api", wantStatus: "DEPLOYED or FAILED"},
			{tool: "delete_model_api", wantStatus: "not found"},
		} {
			result := callModelAPILifecycleTool(t, mcpServer, test.tool, map[string]any{
				"name": "sync-model", "waitForCompletion": wait,
			})
			text := modelAPIToolResultText(result)
			if !result.IsError || !strings.Contains(text, "Use false or omit") || !strings.Contains(text, "get_model_api") || !strings.Contains(text, test.wantStatus) {
				t.Fatalf("%s wait=%q error is not actionable: %q", test.tool, wait, text)
			}
		}
	}
	if handler.createCalls != 0 || handler.deleteCalls != 0 {
		t.Fatalf("unsupported wait reached handler: create=%d delete=%d", handler.createCalls, handler.deleteCalls)
	}
}

func callModelAPILifecycleTool(t *testing.T, mcpServer *server.MCPServer, tool string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := mcpServer.GetTool(tool).Handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
	if err != nil {
		t.Fatalf("%s returned protocol error: %v", tool, err)
	}
	return result
}

func modelAPIToolResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	text, _ := result.Content[0].(mcp.TextContent)
	return text.Text
}

func TestCreateModelAPIBindsOptionalIntegrationConfiguration(t *testing.T) {
	handler := &recordingModelAPIHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterModelAPITools(mcpServer, handler, &config.Config{})

	registered := mcpServer.GetTool("create_model_api")
	if registered == nil {
		t.Fatal("create_model_api is not registered")
	}
	result, err := registered.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{
			"name":              "model-one",
			"model":             "gpt-4o-mini",
			"endpoint":          "provider-endpoint",
			"provider":          "openai",
			"apiKey":            "secret",
			"config":            map[string]any{"baseURL": "https://api.example.test/v1", "organization": "org-one"},
			"waitForCompletion": "false",
		}},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned tool error: %v", result.Content)
	}
	if handler.endpoint != "provider-endpoint" || handler.provider != "openai" || handler.apiKey != "secret" {
		t.Fatalf("optional scalar arguments were not preserved: endpoint=%q provider=%q apiKey=%q", handler.endpoint, handler.provider, handler.apiKey)
	}
	wantConfig := map[string]any{"baseURL": "https://api.example.test/v1", "organization": "org-one"}
	if !reflect.DeepEqual(handler.config, wantConfig) {
		t.Fatalf("config = %#v, want %#v", handler.config, wantConfig)
	}
}

type recordingModelAPIHandler struct {
	endpoint    string
	provider    string
	apiKey      string
	config      map[string]any
	createCalls int
	deleteCalls int
	createWait  string
	deleteWait  string
}

func (*recordingModelAPIHandler) ListModelAPIs(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (*recordingModelAPIHandler) GetModelAPI(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *recordingModelAPIHandler) CreateModelAPI(_ context.Context, _, _, endpoint, _, provider, apiKey, wait string, config map[string]any) ([]byte, error) {
	h.endpoint = endpoint
	h.provider = provider
	h.apiKey = apiKey
	h.config = config
	h.createCalls++
	h.createWait = wait
	return []byte(`{"success":true}`), nil
}
func (h *recordingModelAPIHandler) DeleteModelAPI(_ context.Context, _ string, wait string) ([]byte, error) {
	h.deleteCalls++
	h.deleteWait = wait
	return []byte(`{"success":true}`), nil
}
