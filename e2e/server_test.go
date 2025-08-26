package e2e

import (
	"testing"
)

func TestServerInitialization(t *testing.T) {
	client := NewMCPTestClient(t, testEnv())
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
	client := NewMCPTestClient(t, testEnv())
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
				if tool.InputSchema.Properties == nil || len(tool.InputSchema.Properties) == 0 {
					t.Errorf("Expected create_mcp_server to have properties")
				}
				break
			}
		}
	})
}

func TestReadOnlyMode(t *testing.T) {
	env := testEnv()
	env["BLAXEL_READ_ONLY"] = "true"

	client := NewMCPTestClient(t, env)
	defer client.Close()

	result, err := client.ListTools()
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Check that write tools are not present
	for _, tool := range result.Tools {
		if tool.Name == "create_mcp_server" ||
			tool.Name == "delete_agent" ||
			tool.Name == "create_model_api" {
			t.Errorf("Write tool %s should not be available in read-only mode", tool.Name)
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
