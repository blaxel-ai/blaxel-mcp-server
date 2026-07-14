package serviceaccounts

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestUpdateServiceAccountSchema(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterServiceAccountTools(mcpServer, &recordingServiceAccountHandler{}, &config.Config{})

	registered := mcpServer.GetTool("update_service_account")
	if registered == nil {
		t.Fatal("update_service_account is not registered")
	}

	properties := make([]string, 0, len(registered.Tool.InputSchema.Properties))
	for property := range registered.Tool.InputSchema.Properties {
		properties = append(properties, property)
	}
	sort.Strings(properties)
	if want := []string{"client_id", "name", "workspace"}; !reflect.DeepEqual(properties, want) {
		t.Fatalf("properties = %v, want %v", properties, want)
	}

	required := append([]string(nil), registered.Tool.InputSchema.Required...)
	sort.Strings(required)
	if want := []string{"client_id", "name"}; !reflect.DeepEqual(required, want) {
		t.Fatalf("required properties = %v, want %v", required, want)
	}
}

func TestUpdateServiceAccountBindsClientIDAndName(t *testing.T) {
	handler := &recordingServiceAccountHandler{}
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterServiceAccountTools(mcpServer, handler, &config.Config{})

	registered := mcpServer.GetTool("update_service_account")
	if registered == nil {
		t.Fatal("update_service_account is not registered")
	}
	result, err := registered.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{
			"client_id": "client-123",
			"name":      "Renamed account",
		}},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned tool error: %v", result.Content)
	}
	if handler.updatedClientID != "client-123" {
		t.Errorf("client ID = %q, want %q", handler.updatedClientID, "client-123")
	}
	if handler.updatedName != "Renamed account" {
		t.Errorf("name = %q, want %q", handler.updatedName, "Renamed account")
	}
}

type recordingServiceAccountHandler struct {
	updatedClientID string
	updatedName     string
}

func (*recordingServiceAccountHandler) ListServiceAccounts(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (*recordingServiceAccountHandler) GetServiceAccount(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (*recordingServiceAccountHandler) CreateServiceAccount(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (*recordingServiceAccountHandler) DeleteServiceAccount(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *recordingServiceAccountHandler) UpdateServiceAccount(_ context.Context, clientID, name string) ([]byte, error) {
	h.updatedClientID = clientID
	h.updatedName = name
	return []byte(`{"success":true}`), nil
}
