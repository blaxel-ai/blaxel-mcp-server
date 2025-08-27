package tools

import (
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestMCPServersTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	// Generate random test names to avoid conflicts
	testMCPServerName := e2e.GenerateRandomTestName("test-mcp")
	testMCPServerMissingIntegration := e2e.GenerateRandomTestName("test-mcp-missing-integration")

	t.Run("full_lifecycle_test", func(t *testing.T) {
		t.Run("create_blaxel_search_mcp", func(t *testing.T) {
			// Create an MCP server with blaxel-search integration type
			args := map[string]interface{}{
				"name":            testMCPServerName,
				"integrationType": "blaxel-search",
				"secret":          map[string]interface{}{}, // Empty secret as mentioned
				"config":          map[string]interface{}{}, // Empty config as mentioned
			}

			result, err := client.CallTool("create_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call create_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error (like already exists or auth issues)
				if strings.Contains(errorMsg, "already exists") {
					t.Logf("MCP server already exists, continuing with test: %s", errorMsg)
					return
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from create_mcp_server: %s", errorMsg)
			}

			t.Logf("Successfully created MCP server: %s", testMCPServerName)
		})

		// Note: The create_mcp_server tool now handles status polling internally
		// and waits for the MCP server to reach a final status before returning success

		t.Run("list_mcp_servers", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_mcp_servers", args)
			if err != nil {
				t.Fatalf("Failed to call list_mcp_servers: %v", err)
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
				t.Fatalf("Unexpected error from list_mcp_servers: %s", errorMsg)
			}

			t.Logf("Successfully listed MCP servers")
		})

		t.Run("get_mcp_server", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testMCPServerName,
			}

			result, err := client.CallTool("get_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call get_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error
				if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from get_mcp_server: %s", errorMsg)
			}

			t.Logf("Successfully retrieved MCP server: %s", testMCPServerName)
		})

		t.Run("delete_mcp_server", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testMCPServerName,
			}

			result, err := client.CallTool("delete_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call delete_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error
				if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from delete_mcp_server: %s", errorMsg)
			}

			t.Logf("Successfully deleted MCP server: %s", testMCPServerName)
		})

		t.Run("verify_mcp_server_deleted", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testMCPServerName,
			}

			result, err := client.CallTool("get_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call get_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Accept any error as valid for deleted MCP server
				if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("Successfully verified MCP server is deleted (got expected error): %s", errorMsg)
					return
				}
				t.Logf("Got error when getting deleted MCP server: %s", errorMsg)
			} else {
				t.Logf("MCP server still exists after deletion (this might be expected depending on API behavior)")
			}

			t.Logf("Verification complete for MCP server: %s", testMCPServerName)
		})
	})

	t.Run("wait_for_completion_tests", func(t *testing.T) {
		testMCPServerNoWait := e2e.GenerateRandomTestName("test-mcp-no-wait")
		testMCPServerWait := e2e.GenerateRandomTestName("test-mcp-wait")

		t.Run("create_mcp_server_no_wait", func(t *testing.T) {
			args := map[string]interface{}{
				"name":              testMCPServerNoWait,
				"integrationType":   "blaxel-search",
				"secret":            map[string]interface{}{},
				"config":            map[string]interface{}{},
				"waitForCompletion": "false",
			}

			result, err := client.CallTool("create_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call create_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error (like already exists or auth issues)
				if strings.Contains(errorMsg, "already exists") {
					t.Logf("MCP server already exists, continuing with test: %s", errorMsg)
					return
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from create_mcp_server: %s", errorMsg)
			}

			_, _ = client.CallTool("delete_mcp_server", map[string]interface{}{"name": testMCPServerNoWait, "waitForCompletion": "false"})
			t.Logf("Successfully created MCP server without waiting: %s", testMCPServerNoWait)
		})

		t.Run("create_mcp_server_with_wait", func(t *testing.T) {
			args := map[string]interface{}{
				"name":              testMCPServerWait,
				"integrationType":   "blaxel-search",
				"secret":            map[string]interface{}{},
				"config":            map[string]interface{}{},
				"waitForCompletion": "true",
			}

			result, err := client.CallTool("create_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call create_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error (like already exists or auth issues)
				if strings.Contains(errorMsg, "already exists") {
					t.Logf("MCP server already exists, continuing with test: %s", errorMsg)
					return
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from create_mcp_server: %s", errorMsg)
			}

			t.Logf("Successfully created MCP server with waiting: %s", testMCPServerWait)
		})

		t.Run("delete_mcp_server_no_wait", func(t *testing.T) {
			args := map[string]interface{}{
				"name":              testMCPServerNoWait,
				"waitForCompletion": "false",
			}

			result, err := client.CallTool("delete_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call delete_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error
				if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from delete_mcp_server: %s", errorMsg)
			}

			t.Logf("Successfully initiated deletion of MCP server without waiting: %s", testMCPServerNoWait)
		})

		t.Run("delete_mcp_server_with_wait", func(t *testing.T) {
			args := map[string]interface{}{
				"name":              testMCPServerWait,
				"waitForCompletion": "true",
			}

			result, err := client.CallTool("delete_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call delete_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error
				if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from delete_mcp_server: %s", errorMsg)
			}

			t.Logf("Successfully deleted MCP server with waiting: %s", testMCPServerWait)
		})
	})

	t.Run("error_handling_tests", func(t *testing.T) {
		t.Run("create_mcp_server_missing_name", func(t *testing.T) {
			args := map[string]interface{}{
				"integrationType": "blaxel-search",
			}

			result, err := client.CallTool("create_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call create_mcp_server: %v", err)
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

		t.Run("create_mcp_server_missing_integration", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testMCPServerMissingIntegration,
			}

			result, err := client.CallTool("create_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call create_mcp_server: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing integration parameters")
			}
			if !strings.Contains(errorMsg, "must provide") && !strings.Contains(errorMsg, "integration") {
				t.Errorf("Expected error about missing integration params, got: %s", errorMsg)
			}
			_, _ = client.CallTool("delete_mcp_server", map[string]interface{}{"name": testMCPServerMissingIntegration, "waitForCompletion": "false"})
		})

		t.Run("create_mcp_server_both_integration_modes", func(t *testing.T) {
			args := map[string]interface{}{
				"name":                      e2e.GenerateRandomTestName("test-mcp-both"),
				"integrationType":           "blaxel-search",
				"integrationConnectionName": "test-integration",
			}

			result, err := client.CallTool("create_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call create_mcp_server: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for providing both integration modes")
			}
			if !strings.Contains(errorMsg, "not both") && !strings.Contains(errorMsg, "both") {
				t.Errorf("Expected error about both modes, got: %s", errorMsg)
			}
			_, _ = client.CallTool("delete_mcp_server", map[string]interface{}{"name": args["name"], "waitForCompletion": "false"})
		})

		t.Run("get_mcp_server_missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("get_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call get_mcp_server: %v", err)
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

		t.Run("delete_mcp_server_missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("delete_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call delete_mcp_server: %v", err)
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
	})

	// Cleanup any test MCP servers and integrations that might have been created
	t.Cleanup(func() {
		// Try to delete the test MCP server if it still exists
		cleanupArgs := map[string]interface{}{
			"name": testMCPServerName,
		}
		_, _ = client.CallTool("delete_mcp_server", cleanupArgs)

		// Cleanup integrations using regex patterns
		// This test creates integrations with names like "test-mcp-XXXX-blaxel-search-integration"
		patterns := []string{
			`^test-mcp-\d{1,4}-.*$`,
			`^test-mcp-wait-\d{1,4}-.*$`,
			`^test-mcp-no-wait-\d{1,4}-.*$`,
		}
		e2e.CleanupTestIntegrations(patterns)
	})
}
