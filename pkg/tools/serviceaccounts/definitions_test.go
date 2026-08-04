package serviceaccounts

import (
	"context"
	"reflect"
	"sort"
	"strings"
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

func TestCreateServiceAccountSchemaRequiresExplicitSecretDisclosure(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	RegisterServiceAccountTools(mcpServer, &recordingServiceAccountHandler{}, &config.Config{})

	registered := mcpServer.GetTool("create_service_account")
	if registered == nil {
		t.Fatal("create_service_account is not registered")
	}

	properties := make([]string, 0, len(registered.Tool.InputSchema.Properties))
	for property := range registered.Tool.InputSchema.Properties {
		properties = append(properties, property)
	}
	sort.Strings(properties)
	if want := []string{"name", "revealSecret", "workspace"}; !reflect.DeepEqual(properties, want) {
		t.Fatalf("properties = %v, want %v", properties, want)
	}
	if want := []string{"name"}; !reflect.DeepEqual(registered.Tool.InputSchema.Required, want) {
		t.Fatalf("required properties = %v, want %v", registered.Tool.InputSchema.Required, want)
	}
	revealSchema, ok := registered.Tool.InputSchema.Properties["revealSecret"].(map[string]any)
	if !ok || revealSchema["type"] != "boolean" {
		t.Fatalf("revealSecret schema = %#v, want boolean", registered.Tool.InputSchema.Properties["revealSecret"])
	}
	if !strings.Contains(registered.Tool.Description, "redacted by default") {
		t.Fatalf("tool description does not explain the safe default: %q", registered.Tool.Description)
	}
}

func TestCreateServiceAccountBindsRevealSecret(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments map[string]any
		want      bool
	}{
		{name: "defaults to false", arguments: map[string]any{"name": "ci"}},
		{name: "explicit true", arguments: map[string]any{"name": "ci", "revealSecret": true}, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := &recordingServiceAccountHandler{}
			mcpServer := server.NewMCPServer("test", "1.0.0")
			RegisterServiceAccountTools(mcpServer, handler, &config.Config{})

			result, err := mcpServer.GetTool("create_service_account").Handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Arguments: test.arguments},
			})
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if result.IsError {
				t.Fatalf("handler returned tool error: %v", result.Content)
			}
			if handler.createdName != "ci" {
				t.Fatalf("created name = %q, want ci", handler.createdName)
			}
			if handler.revealSecret != test.want {
				t.Fatalf("revealSecret = %t, want %t", handler.revealSecret, test.want)
			}
		})
	}
}

type recordingServiceAccountHandler struct {
	updatedClientID string
	updatedName     string
	createdName     string
	revealSecret    bool
}

func (*recordingServiceAccountHandler) ListServiceAccounts(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (*recordingServiceAccountHandler) GetServiceAccount(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *recordingServiceAccountHandler) CreateServiceAccount(_ context.Context, name string, revealSecret bool) ([]byte, error) {
	h.createdName = name
	h.revealSecret = revealSecret
	return []byte(`{"success":true}`), nil
}
func (*recordingServiceAccountHandler) DeleteServiceAccount(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *recordingServiceAccountHandler) UpdateServiceAccount(_ context.Context, clientID, name string) ([]byte, error) {
	h.updatedClientID = clientID
	h.updatedName = name
	return []byte(`{"success":true}`), nil
}
