package tools

import (
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestAgentsTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	t.Run("list_agents", func(t *testing.T) {
		t.Run("basic_list", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_agents", args)
			if err != nil {
				t.Fatalf("Failed to call list_agents: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// API errors are expected with test credentials
				if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from list_agents: %s", errorMsg)
			}

			// If successful, just log that the call succeeded
			t.Logf("list_agents call succeeded")
		})

		t.Run("with_filter", func(t *testing.T) {
			args := map[string]interface{}{
				"filter": "test",
			}

			result, err := client.CallTool("list_agents", args)
			if err != nil {
				t.Fatalf("Failed to call list_agents with filter: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Filter parameter should be accepted even if API fails
				if strings.Contains(errorMsg, "filter") {
					t.Errorf("Tool should accept filter parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") ||
					strings.Contains(errorMsg, "404") || strings.Contains(errorMsg, "not found") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from list_agents with filter: %s", errorMsg)
			}
		})
	})

	t.Run("get_agent", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("get_agent", args)
			if err != nil {
				// Check if it's an error that mentions "name"
				if strings.Contains(err.Error(), "name") {
					return // Expected error
				}
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing name field")
			}
			if !strings.Contains(errorMsg, "name") {
				t.Errorf("Expected error to mention 'name', got: %s", errorMsg)
			}
		})

		t.Run("with_name", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "test-agent",
			}

			result, err := client.CallTool("get_agent", args)
			if err != nil {
				t.Fatalf("Failed to call get_agent: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if agent not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from get_agent: %s", errorMsg)
			}
		})
	})

	t.Run("delete_agent", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("delete_agent", args)
			if err != nil {
				// Check if it's an error that mentions "name"
				if strings.Contains(err.Error(), "name") {
					return // Expected error
				}
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing name field")
			}
			if !strings.Contains(errorMsg, "name") {
				t.Errorf("Expected error to mention 'name', got: %s", errorMsg)
			}
		})

		t.Run("with_name", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "test-agent",
			}

			result, err := client.CallTool("delete_agent", args)
			if err != nil {
				t.Fatalf("Failed to call delete_agent: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if agent not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from delete_agent: %s", errorMsg)
			} else {
				// Log the successful response for debugging
				resp, _ := e2e.ExtractJSONResult(result)
				t.Logf("delete_agent returned success: %+v", resp)
			}
		})
	})
}
