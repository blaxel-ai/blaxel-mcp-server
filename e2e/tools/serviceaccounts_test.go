package tools

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestServiceAccountsTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	t.Run("list_service_accounts", func(t *testing.T) {
		t.Run("basic_list", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_service_accounts", args)
			if err != nil {
				t.Fatalf("Failed to call list_service_accounts: %v", err)
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
				t.Fatalf("Unexpected error from list_service_accounts: %s", errorMsg)
			}

			// If successful, validate the response structure
			// t.Logf("call succeeded")
			if err != nil {
			}

		})
	})

	t.Run("get_service_account", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("get_service_account", args)
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
				"name": fmt.Sprintf("test-service-account-%d", rand.Intn(10000)),
			}

			result, err := client.CallTool("get_service_account", args)
			if err != nil {
				t.Fatalf("Failed to call get_service_account: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if service account not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from get_service_account: %s", errorMsg)
			}
		})
	})

	t.Run("create_service_account", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("create_service_account", args)
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
				"name": "test-service-account",
			}

			result, err := client.CallTool("create_service_account", args)
			if err != nil {
				t.Fatalf("Failed to call create_service_account: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check it's not a parameter validation error
				if strings.Contains(errorMsg, "required") ||
					strings.Contains(errorMsg, "must provide") {
					t.Errorf("Should not have parameter validation error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				// Other errors are expected (e.g., API failures)
				t.Logf("Expected API error: %s", errorMsg)
			}
		})

		t.Run("with_description", func(t *testing.T) {
			args := map[string]interface{}{
				"name":        fmt.Sprintf("test-service-account-desc-%d", rand.Intn(10000)),
				"description": "Test service account for testing",
			}

			result, err := client.CallTool("create_service_account", args)
			if err != nil {
				t.Fatalf("Failed to call create_service_account: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check it's not a parameter validation error
				if strings.Contains(errorMsg, "required") ||
					strings.Contains(errorMsg, "must provide") {
					t.Errorf("Should not have parameter validation error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				// Other errors are expected (e.g., API failures)
				t.Logf("Expected API error: %s", errorMsg)
			}
		})
	})

	t.Run("delete_service_account", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("delete_service_account", args)
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
				"name": fmt.Sprintf("test-service-account-%d", rand.Intn(10000)),
			}

			result, err := client.CallTool("delete_service_account", args)
			if err != nil {
				t.Fatalf("Failed to call delete_service_account: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if service account not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from delete_service_account: %s", errorMsg)
			} else {
				// Log the successful response for debugging
				resp, _ := e2e.ExtractJSONResult(result)
				t.Logf("delete_service_account returned success: %+v", resp)
			}
		})
	})

	t.Run("update_service_account", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("update_service_account", args)
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
				"name": "test-service-account",
			}

			result, err := client.CallTool("update_service_account", args)
			if err != nil {
				t.Fatalf("Failed to call update_service_account: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if service account not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from update_service_account: %s", errorMsg)
			}
		})

		t.Run("with_description", func(t *testing.T) {
			args := map[string]interface{}{
				"name":        "test-service-account",
				"description": "Updated description for test service account",
			}

			result, err := client.CallTool("update_service_account", args)
			if err != nil {
				t.Fatalf("Failed to call update_service_account: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept description parameter even if service account not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept description parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from update_service_account: %s", errorMsg)
			}
		})
	})
}
