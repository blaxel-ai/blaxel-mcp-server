package e2e

import (
	"encoding/json"
	"testing"
)

func TestServerInitialization(t *testing.T) {
	client := NewMCPTestClient(t, testEnv())
	defer client.Close()

	t.Run("initialize", func(t *testing.T) {
		result, err := client.SendRequest("initialize", nil)
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}

		var init struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
			Capabilities struct {
				Tools interface{} `json:"tools"`
			} `json:"capabilities"`
		}

		if err := json.Unmarshal(result, &init); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		// Protocol version varies by implementation
		if init.ProtocolVersion == "" {
			t.Error("Protocol version is empty")
		}

		if init.ServerInfo.Name != "blaxel-mcp-server" {
			t.Errorf("Expected server name blaxel-mcp-server, got %s", init.ServerInfo.Name)
		}

		if init.ServerInfo.Version == "" {
			t.Error("Server version is empty")
		}
	})

	t.Run("ping", func(t *testing.T) {
		result, err := client.SendRequest("ping", nil)
		if err != nil {
			t.Fatalf("Failed to ping: %v", err)
		}

		var pong map[string]interface{}
		if err := json.Unmarshal(result, &pong); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		if len(pong) != 0 {
			t.Errorf("Expected empty response, got %v", pong)
		}
	})
}

func TestToolsDiscovery(t *testing.T) {
	client := NewMCPTestClient(t, testEnv())
	defer client.Close()

	// Initialize first
	if _, err := client.SendRequest("initialize", nil); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Run("list_tools", func(t *testing.T) {
		result, err := client.SendRequest("tools/list", nil)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		var toolsResp struct {
			Tools []struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				InputSchema map[string]interface{} `json:"inputSchema"`
			} `json:"tools"`
		}

		if err := json.Unmarshal(result, &toolsResp); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
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
		for _, tool := range toolsResp.Tools {
			toolNames[tool.Name] = true
		}

		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("Expected tool %s not found", expected)
			}
		}
	})

	t.Run("tool_schemas", func(t *testing.T) {
		result, err := client.SendRequest("tools/list", nil)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		var toolsResp struct {
			Tools []struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				InputSchema map[string]interface{} `json:"inputSchema"`
			} `json:"tools"`
		}

		if err := json.Unmarshal(result, &toolsResp); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		// Check create_mcp_server schema
		for _, tool := range toolsResp.Tools {
			if tool.Name == "create_mcp_server" {
				if tool.InputSchema["type"] != "object" {
					t.Errorf("Expected create_mcp_server input schema type to be object")
				}
				if tool.InputSchema["properties"] == nil {
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

	// Initialize first
	if _, err := client.SendRequest("initialize", nil); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	result, err := client.SendRequest("tools/list", nil)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	var toolsResp struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(result, &toolsResp); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Check that write tools are not present
	for _, tool := range toolsResp.Tools {
		if tool.Name == "create_mcp_server" ||
			tool.Name == "delete_agent" ||
			tool.Name == "create_model_api" {
			t.Errorf("Write tool %s should not be available in read-only mode", tool.Name)
		}
	}

	// Check that read tools are still present
	hasListAgents := false
	for _, tool := range toolsResp.Tools {
		if tool.Name == "list_agents" {
			hasListAgents = true
			break
		}
	}
	if !hasListAgents {
		t.Error("Read tool list_agents should be available in read-only mode")
	}
}
