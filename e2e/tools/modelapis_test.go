package tools

import (
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestModelAPIsTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	// Get OpenAI API key from environment
	// openaiAPIKey := os.Getenv("OPENAI_API_KEY")

	// Generate random test names to avoid conflicts
	testModelAPIName := e2e.GenerateRandomTestName("test-model")

	// t.Run("full_lifecycle_test", func(t *testing.T) {
	// 	t.Run("create_blaxel_model_api", func(t *testing.T) {
	// 		// Create a model API with OpenAI integration
	// 		args := map[string]interface{}{
	// 			"name":     testModelAPIName,
	// 			"provider": "openai",
	// 			"apiKey":   openaiAPIKey,
	// 			"model":    "gpt-4",
	// 		}

	// 		result, err := client.CallTool("create_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call create_model_api: %v", err)
	// 		}

	// 		// Check if the tool returned an error
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if isError {
	// 			// Check if it's an expected error (like already exists or auth issues)
	// 			if strings.Contains(errorMsg, "already exists") {
	// 				t.Logf("Model API already exists, continuing with test: %s", errorMsg)
	// 				return
	// 			} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
	// 				strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
	// 				t.Logf("API call failed as expected with test credentials: %s", errorMsg)
	// 				return
	// 			}
	// 			t.Fatalf("Unexpected error from create_model_api: %s", errorMsg)
	// 		}

	// 		t.Logf("Successfully created model API: %s", testModelAPIName)
	// 	})

	// 	// Note: The create_model_api tool now handles status polling internally
	// 	// and waits for the model API to reach a final status before returning success

	// 	t.Run("list_model_apis", func(t *testing.T) {
	// 		args := map[string]interface{}{}

	// 		result, err := client.CallTool("list_model_apis", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call list_model_apis: %v", err)
	// 		}

	// 		// Check if the tool returned an error
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if isError {
	// 			// API errors are expected with test credentials
	// 			if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
	// 				strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
	// 				t.Logf("API call failed as expected with test credentials: %s", errorMsg)
	// 				return
	// 			}
	// 			t.Fatalf("Unexpected error from list_model_apis: %s", errorMsg)
	// 		}

	// 		t.Logf("Successfully listed model APIs")
	// 	})

	// 	t.Run("get_model_api", func(t *testing.T) {
	// 		args := map[string]interface{}{
	// 			"name": testModelAPIName,
	// 		}

	// 		result, err := client.CallTool("get_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call get_model_api: %v", err)
	// 		}

	// 		// Check if the tool returned an error
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if isError {
	// 			// Check if it's an expected error
	// 			if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
	// 				strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
	// 				strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
	// 				t.Logf("API call failed as expected: %s", errorMsg)
	// 				return
	// 			}
	// 			t.Fatalf("Unexpected error from get_model_api: %s", errorMsg)
	// 		}

	// 		t.Logf("Successfully retrieved model API: %s", testModelAPIName)
	// 	})

	// 	t.Run("run_model_api", func(t *testing.T) {
	// 		args := map[string]interface{}{
	// 			"name": testModelAPIName,
	// 			"body": map[string]interface{}{
	// 				"prompt": "Hello, how are you?",
	// 			},
	// 		}

	// 		result, err := client.CallTool("run_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call run_model_api: %v", err)
	// 		}
	// 		t.Logf("Result: %v", result)

	// 		t.Logf("Successfully ran model API: %s", testModelAPIName)
	// 	})

	// 	t.Run("delete_model_api", func(t *testing.T) {
	// 		args := map[string]interface{}{
	// 			"name": testModelAPIName,
	// 		}

	// 		result, err := client.CallTool("delete_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call delete_model_api: %v", err)
	// 		}

	// 		// Check if the tool returned an error
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if isError {
	// 			// Check if it's an expected error
	// 			if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
	// 				strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
	// 				strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
	// 				t.Logf("API call failed as expected: %s", errorMsg)
	// 				return
	// 			}
	// 			t.Fatalf("Unexpected error from delete_model_api: %s", errorMsg)
	// 		}

	// 		t.Logf("Successfully deleted model API: %s", testModelAPIName)
	// 	})

	// 	t.Run("verify_model_api_deleted", func(t *testing.T) {
	// 		args := map[string]interface{}{
	// 			"name": testModelAPIName,
	// 		}

	// 		result, err := client.CallTool("get_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call get_model_api: %v", err)
	// 		}

	// 		// Check if the tool returned an error
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if isError {
	// 			// Accept any error as valid for deleted model API
	// 			if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
	// 				strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
	// 				strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
	// 				t.Logf("Successfully verified model API is deleted (got expected error): %s", errorMsg)
	// 				return
	// 			}
	// 			t.Logf("Got error when getting deleted model API: %s", errorMsg)
	// 		} else {
	// 			t.Logf("Model API still exists after deletion (this might be expected depending on API behavior)")
	// 		}

	// 		t.Logf("Verification complete for model API: %s", testModelAPIName)
	// 	})
	// })

	// t.Run("error_handling_tests", func(t *testing.T) {
	// 	t.Run("create_model_api_missing_name", func(t *testing.T) {
	// 		args := map[string]interface{}{
	// 			"provider": "openai",
	// 			"apiKey":   openaiAPIKey,
	// 		}

	// 		result, err := client.CallTool("create_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call create_model_api: %v", err)
	// 		}

	// 		// Check for tool error in result
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if !isError {
	// 			t.Fatal("Expected error for missing name field")
	// 		}
	// 		if !strings.Contains(errorMsg, "name") {
	// 			t.Errorf("Expected error to mention 'name', got: %s", errorMsg)
	// 		}
	// 	})

	// 	t.Run("create_model_api_missing_integration", func(t *testing.T) {
	// 		args := map[string]interface{}{
	// 			"name": e2e.GenerateRandomTestName("test-model-missing-integration"),
	// 		}

	// 		result, err := client.CallTool("create_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call create_model_api: %v", err)
	// 		}

	// 		// Check for tool error in result
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if !isError {
	// 			t.Fatal("Expected error for missing integration parameters")
	// 		}
	// 		if !strings.Contains(errorMsg, "must provide") && !strings.Contains(errorMsg, "integration") {
	// 			t.Errorf("Expected error about missing integration params, got: %s", errorMsg)
	// 		}
	// 	})

	// 	t.Run("create_model_api_missing_apikey", func(t *testing.T) {
	// 		args := map[string]interface{}{
	// 			"name":     e2e.GenerateRandomTestName("test-model-no-key"),
	// 			"provider": "openai",
	// 			// Missing apiKey
	// 		}

	// 		result, err := client.CallTool("create_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call create_model_api: %v", err)
	// 		}

	// 		// Check for tool error in result
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if !isError {
	// 			t.Fatal("Expected error for missing API key")
	// 		}
	// 		if !strings.Contains(errorMsg, "api key") {
	// 			t.Errorf("Expected error about API key, got: %s", errorMsg)
	// 		}
	// 	})

	// 	t.Run("get_model_api_missing_name", func(t *testing.T) {
	// 		args := map[string]interface{}{}

	// 		result, err := client.CallTool("get_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call get_model_api: %v", err)
	// 		}

	// 		// Check for tool error in result
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if !isError {
	// 			t.Fatal("Expected error for missing name field")
	// 		}
	// 		if !strings.Contains(errorMsg, "name") {
	// 			t.Errorf("Expected error to mention 'name', got: %s", errorMsg)
	// 		}
	// 	})

	// 	t.Run("delete_model_api_missing_name", func(t *testing.T) {
	// 		args := map[string]interface{}{}

	// 		result, err := client.CallTool("delete_model_api", args)
	// 		if err != nil {
	// 			t.Fatalf("Failed to call delete_model_api: %v", err)
	// 		}

	// 		// Check for tool error in result
	// 		isError, errorMsg := e2e.CheckToolError(result)
	// 		if !isError {
	// 			t.Fatal("Expected error for missing name field")
	// 		}
	// 		if !strings.Contains(errorMsg, "name") {
	// 			t.Errorf("Expected error to mention 'name', got: %s", errorMsg)
	// 		}
	// 	})
	// })

	// Cleanup any test model APIs and integrations that might have been created
	t.Cleanup(func() {
		// Try to delete the test model API if it still exists
		cleanupArgs := map[string]interface{}{
			"name": testModelAPIName,
		}
		_, _ = client.CallTool("delete_model_api", cleanupArgs)

		// Cleanup integrations using the generalized function
		patterns := []string{
			`^test-model-\d{1,4}-.*$`,
			`^test-model-wait-\d{1,4}-.*$`,
			`^test-model-no-wait-\d{1,4}-.*$`,
		}
		e2e.CleanupTestIntegrations(patterns)
	})
}
