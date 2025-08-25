package e2e

import (
	"encoding/json"
	"strings"
	"testing"
)

// Helper function to check if a tool call resulted in an error
func checkToolError(result json.RawMessage) (bool, string) {
	var toolResult struct {
		IsError bool `json:"isError"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(result, &toolResult); err != nil {
		return false, ""
	}

	if toolResult.IsError && len(toolResult.Content) > 0 {
		return true, toolResult.Content[0].Text
	}

	return false, ""
}

func TestCreateMCPServer(t *testing.T) {
	client := NewMCPTestClient(t, testEnv())
	defer client.Close()

	// Initialize first
	if _, err := client.SendRequest("initialize", nil); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Run("missing_required_fields", func(t *testing.T) {
		params := map[string]interface{}{
			"name":      "create_mcp_server",
			"arguments": map[string]interface{}{
				// Missing 'name' field
			},
		}

		result, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check if it's a JSON-RPC error that mentions "name"
			if strings.Contains(err.Error(), "name") {
				return // Expected error
			}
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check for tool error in result
		isError, errorMsg := checkToolError(result)
		if !isError {
			t.Fatal("Expected error for missing name field")
		}
		if !strings.Contains(errorMsg, "name") {
			t.Errorf("Expected error to mention 'name', got: %s", errorMsg)
		}
	})

	t.Run("missing_integration_params", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "create_mcp_server",
			"arguments": map[string]interface{}{
				"name": "test-mcp",
				// Missing both integrationType and integrationConnectionName
			},
		}

		result, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check if it's a JSON-RPC error that mentions integration
			if strings.Contains(err.Error(), "must provide") || strings.Contains(err.Error(), "integration") {
				return // Expected error
			}
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check for tool error in result
		isError, errorMsg := checkToolError(result)
		if !isError {
			t.Fatal("Expected error for missing integration parameters")
		}
		if !strings.Contains(errorMsg, "must provide") {
			t.Errorf("Expected error about missing integration params, got: %s", errorMsg)
		}
	})

	t.Run("both_integration_modes", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "create_mcp_server",
			"arguments": map[string]interface{}{
				"name":                      "test-mcp",
				"integrationType":           "github",
				"integrationConnectionName": "existing-integration",
			},
		}

		result, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check if it's a JSON-RPC error that mentions both modes
			if strings.Contains(err.Error(), "not both") || strings.Contains(err.Error(), "both") {
				return // Expected error
			}
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check for tool error in result
		isError, errorMsg := checkToolError(result)
		if !isError {
			t.Fatal("Expected error for specifying both integration modes")
		}
		if !strings.Contains(errorMsg, "not both") {
			t.Errorf("Expected error about both modes, got: %s", errorMsg)
		}
	})

	t.Run("new_integration_mode", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "create_mcp_server",
			"arguments": map[string]interface{}{
				"name":            "test-github-mcp",
				"integrationType": "github",
				"secret": map[string]interface{}{
					"token": "test-token",
				},
				"config": map[string]interface{}{
					"owner": "test-org",
				},
			},
		}

		// This will fail with API error, but should accept parameters
		_, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check it's not a parameter validation error
			if strings.Contains(err.Error(), "required") ||
				strings.Contains(err.Error(), "must provide") {
				t.Errorf("Should not have parameter validation error: %v", err)
			}
		}
	})

	t.Run("existing_integration_mode", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "create_mcp_server",
			"arguments": map[string]interface{}{
				"name":                      "test-mcp-existing",
				"integrationConnectionName": "my-github-integration",
			},
		}

		// This will fail with API error (integration not found), but should accept parameters
		_, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check it's not a parameter validation error
			if strings.Contains(err.Error(), "required") ||
				strings.Contains(err.Error(), "must provide") {
				t.Errorf("Should not have parameter validation error: %v", err)
			}
		}
	})
}

func TestCreateModelAPI(t *testing.T) {
	client := NewMCPTestClient(t, testEnv())
	defer client.Close()

	// Initialize first
	if _, err := client.SendRequest("initialize", nil); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Run("missing_required_fields", func(t *testing.T) {
		params := map[string]interface{}{
			"name":      "create_model_api",
			"arguments": map[string]interface{}{
				// Missing 'name' field
			},
		}

		result, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check if it's a JSON-RPC error that mentions "name"
			if strings.Contains(err.Error(), "name") {
				return // Expected error
			}
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check for tool error in result
		isError, errorMsg := checkToolError(result)
		if !isError {
			t.Fatal("Expected error for missing name field")
		}
		if !strings.Contains(errorMsg, "name") {
			t.Errorf("Expected error to mention 'name', got: %s", errorMsg)
		}
	})

	t.Run("new_integration_missing_apikey", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "create_model_api",
			"arguments": map[string]interface{}{
				"name":     "test-model",
				"provider": "openai",
				// Missing apiKey
			},
		}

		result, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check if it's a JSON-RPC error that mentions API key
			if strings.Contains(err.Error(), "api key") || strings.Contains(err.Error(), "apiKey") {
				return // Expected error
			}
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check for tool error in result
		isError, errorMsg := checkToolError(result)
		if !isError {
			t.Fatal("Expected error for missing API key")
		}
		if !strings.Contains(errorMsg, "api key") {
			t.Errorf("Expected error about API key, got: %s", errorMsg)
		}
	})

	t.Run("new_integration_mode", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "create_model_api",
			"arguments": map[string]interface{}{
				"name":     "test-gpt4",
				"provider": "openai",
				"apiKey":   "sk-test",
				"model":    "gpt-4",
				"endpoint": "https://api.openai.com/v1",
			},
		}

		// This will fail with API error, but should accept parameters
		_, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check it's not a parameter validation error
			if strings.Contains(err.Error(), "required") ||
				strings.Contains(err.Error(), "must provide") {
				t.Errorf("Should not have parameter validation error: %v", err)
			}
		}
	})

	t.Run("existing_integration_mode", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "create_model_api",
			"arguments": map[string]interface{}{
				"name":                      "test-model-existing",
				"integrationConnectionName": "my-openai-integration",
				"model":                     "gpt-4-turbo",
			},
		}

		// This will fail with API error (integration not found), but should accept parameters
		_, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check it's not a parameter validation error
			if strings.Contains(err.Error(), "required") ||
				strings.Contains(err.Error(), "must provide") {
				t.Errorf("Should not have parameter validation error: %v", err)
			}
		}
	})
}

func TestListTools(t *testing.T) {
	client := NewMCPTestClient(t, testEnv())
	defer client.Close()

	// Initialize first
	if _, err := client.SendRequest("initialize", nil); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Run("list_agents", func(t *testing.T) {
		params := map[string]interface{}{
			"name":      "list_agents",
			"arguments": map[string]interface{}{},
		}

		result, err := client.SendRequest("tools/call", params)
		// May fail if API not available, but should parse response
		if err == nil {
			var resp map[string]interface{}
			if err := json.Unmarshal(result, &resp); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}
			if _, ok := resp["agents"]; ok {
				// Check structure if successful
				if _, ok := resp["count"]; !ok {
					t.Error("Response missing 'count' field")
				}
			}
		}
	})

	t.Run("list_agents_with_filter", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "list_agents",
			"arguments": map[string]interface{}{
				"filter": "test",
			},
		}

		_, err := client.SendRequest("tools/call", params)
		// Should accept filter parameter
		if err != nil && strings.Contains(err.Error(), "filter") {
			t.Errorf("Should accept filter parameter: %v", err)
		}
	})

	t.Run("get_agent", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "get_agent",
			"arguments": map[string]interface{}{
				"name": "test-agent",
			},
		}

		_, err := client.SendRequest("tools/call", params)
		// May fail with "not found", but should accept parameters
		if err != nil && strings.Contains(err.Error(), "required") {
			t.Errorf("Should accept name parameter: %v", err)
		}
	})
}

func TestDeleteTools(t *testing.T) {
	client := NewMCPTestClient(t, testEnv())
	defer client.Close()

	// Initialize first
	if _, err := client.SendRequest("initialize", nil); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Run("delete_agent_missing_name", func(t *testing.T) {
		params := map[string]interface{}{
			"name":      "delete_agent",
			"arguments": map[string]interface{}{},
		}

		result, err := client.SendRequest("tools/call", params)
		if err != nil {
			// Check if it's a JSON-RPC error that mentions "name"
			if strings.Contains(err.Error(), "name") {
				return // Expected error
			}
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check for tool error in result
		isError, errorMsg := checkToolError(result)
		if !isError {
			t.Fatal("Expected error for missing name")
		}
		if !strings.Contains(errorMsg, "name") {
			t.Errorf("Expected error about name, got: %s", errorMsg)
		}
	})

	t.Run("delete_mcp_server", func(t *testing.T) {
		params := map[string]interface{}{
			"name": "delete_mcp_server",
			"arguments": map[string]interface{}{
				"name": "non-existent",
			},
		}

		_, err := client.SendRequest("tools/call", params)
		// May fail with "not found", but should accept parameters
		if err != nil && strings.Contains(err.Error(), "required") {
			t.Errorf("Should accept name parameter: %v", err)
		}
	})
}
