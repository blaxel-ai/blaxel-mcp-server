package modelapis

import (
	"context"
	"reflect"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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
	endpoint string
	provider string
	apiKey   string
	config   map[string]any
}

func (*recordingModelAPIHandler) ListModelAPIs(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (*recordingModelAPIHandler) GetModelAPI(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *recordingModelAPIHandler) CreateModelAPI(_ context.Context, _, _, endpoint, _, provider, apiKey, _ string, config map[string]any) ([]byte, error) {
	h.endpoint = endpoint
	h.provider = provider
	h.apiKey = apiKey
	h.config = config
	return []byte(`{"success":true}`), nil
}
func (*recordingModelAPIHandler) DeleteModelAPI(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
