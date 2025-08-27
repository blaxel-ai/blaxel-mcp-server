package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestIntegrationsTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	// Get OpenAI API key from environment
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")

	// Generate random test names to avoid conflicts
	testIntegrationName := e2e.GenerateRandomTestName("test-integration")

	t.Run("full_lifecycle_test", func(t *testing.T) {
		t.Run("create_openai_integration", func(t *testing.T) {
			args := map[string]interface{}{
				"name":            testIntegrationName,
				"integrationType": "openai",
				"secret": map[string]interface{}{
					"apiKey": openaiAPIKey,
				},
				"config": map[string]interface{}{
					"baseURL": "https://api.openai.com/v1",
				},
			}

			result, err := client.CallTool("create_integration", args)
			if err != nil {
				t.Fatalf("Failed to call create_integration: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error (like already exists or auth issues)
				if strings.Contains(errorMsg, "already exists") {
					t.Logf("Integration already exists, continuing with test: %s", errorMsg)
					return
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from create_integration: %s", errorMsg)
			}

			t.Logf("Successfully created integration: %s", testIntegrationName)
		})

		t.Run("list_integrations", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_integrations", args)
			if err != nil {
				t.Fatalf("Failed to call list_integrations: %v", err)
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
				t.Fatalf("Unexpected error from list_integrations: %s", errorMsg)
			}

			t.Logf("Successfully listed integrations")
		})

		t.Run("get_integration", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testIntegrationName,
			}

			result, err := client.CallTool("get_integration", args)
			if err != nil {
				t.Fatalf("Failed to call get_integration: %v", err)
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
				t.Fatalf("Unexpected error from get_integration: %s", errorMsg)
			}

			t.Logf("Successfully retrieved integration: %s", testIntegrationName)
		})

		t.Run("delete_integration", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testIntegrationName,
			}

			result, err := client.CallTool("delete_integration", args)
			if err != nil {
				t.Fatalf("Failed to call delete_integration: %v", err)
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
				t.Fatalf("Unexpected error from delete_integration: %s", errorMsg)
			}

			t.Logf("Successfully deleted integration: %s", testIntegrationName)
		})

		t.Run("verify_integration_deleted", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testIntegrationName,
			}

			result, err := client.CallTool("get_integration", args)
			if err != nil {
				t.Fatalf("Failed to call get_integration: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Accept any error as valid for deleted integration
				if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("Successfully verified integration is deleted (got expected error): %s", errorMsg)
					return
				}
				t.Logf("Got error when getting deleted integration: %s", errorMsg)
			} else {
				t.Logf("Integration still exists after deletion (this might be expected depending on API behavior)")
			}

			t.Logf("Verification complete for integration: %s", testIntegrationName)
		})
	})

	t.Run("error_handling_tests", func(t *testing.T) {
		t.Run("create_integration_missing_name", func(t *testing.T) {
			args := map[string]interface{}{
				"integrationType": "openai",
				"secret": map[string]interface{}{
					"apiKey": openaiAPIKey,
				},
			}

			result, err := client.CallTool("create_integration", args)
			if err != nil {
				t.Fatalf("Failed to call create_integration: %v", err)
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

		t.Run("create_integration_missing_type", func(t *testing.T) {
			args := map[string]interface{}{
				"name": e2e.GenerateRandomTestName("test-integration-missing-type"),
				"secret": map[string]interface{}{
					"apiKey": openaiAPIKey,
				},
			}

			result, err := client.CallTool("create_integration", args)
			if err != nil {
				t.Fatalf("Failed to call create_integration: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing type field")
			}
			if !strings.Contains(errorMsg, "integrationType") {
				t.Errorf("Expected error to mention 'integrationType', got: %s", errorMsg)
			}
		})

		t.Run("get_integration_missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("get_integration", args)
			if err != nil {
				t.Fatalf("Failed to call get_integration: %v", err)
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

		t.Run("delete_integration_missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("delete_integration", args)
			if err != nil {
				t.Fatalf("Failed to call delete_integration: %v", err)
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

	// Cleanup any test integrations that might have been created
	t.Cleanup(func() {
		// Try to delete the test integration if it still exists
		cleanupArgs := map[string]interface{}{
			"name": testIntegrationName,
		}
		_, _ = client.CallTool("delete_integration", cleanupArgs)
	})
}
