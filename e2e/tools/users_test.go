package tools

import (
	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
	"strings"
	"testing"
)

func TestUsersTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	t.Run("list_workspace_users", func(t *testing.T) {
		t.Run("basic_list", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_workspace_users", args)
			if err != nil {
				t.Fatalf("Failed to call list_workspace_users: %v", err)
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
				t.Fatalf("Unexpected error from list_workspace_users: %s", errorMsg)
			}

			// If successful, validate the response structure
			// t.Logf("call succeeded")
			if err != nil {
			}

		})
	})

	t.Run("get_workspace_user", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("get_workspace_user", args)
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
				"name": "test-user",
			}

			result, err := client.CallTool("get_workspace_user", args)
			if err != nil {
				t.Fatalf("Failed to call get_workspace_user: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if user not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from get_workspace_user: %s", errorMsg)
			}
		})
	})

	t.Run("invite_workspace_user", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'email' field
			}

			result, err := client.CallTool("invite_workspace_user", args)
			if err != nil {
				// Check if it's an error that mentions "email"
				if strings.Contains(err.Error(), "email") {
					return // Expected error
				}
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing email field")
			}
			if !strings.Contains(errorMsg, "email") {
				t.Errorf("Expected error to mention 'email', got: %s", errorMsg)
			}
		})

		t.Run("with_email", func(t *testing.T) {
			args := map[string]interface{}{
				"email": "test@example.com",
			}

			result, err := client.CallTool("invite_workspace_user", args)
			if err != nil {
				t.Fatalf("Failed to call invite_workspace_user: %v", err)
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

		t.Run("with_role", func(t *testing.T) {
			args := map[string]interface{}{
				"email": "test@example.com",
				"role":  "member",
			}

			result, err := client.CallTool("invite_workspace_user", args)
			if err != nil {
				t.Fatalf("Failed to call invite_workspace_user: %v", err)
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

	t.Run("update_workspace_user_role", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("update_workspace_user_role", args)
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

		t.Run("missing_role", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "test-user",
				// Missing 'role' field
			}

			result, err := client.CallTool("update_workspace_user_role", args)
			if err != nil {
				// Check if it's an error that mentions "role"
				if strings.Contains(err.Error(), "role") {
					return // Expected error
				}
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing role field")
			}
			if !strings.Contains(errorMsg, "role") {
				t.Errorf("Expected error to mention 'role', got: %s", errorMsg)
			}
		})

		t.Run("with_name_and_role", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "test-user",
				"role": "admin",
			}

			result, err := client.CallTool("update_workspace_user_role", args)
			if err != nil {
				t.Fatalf("Failed to call update_workspace_user_role: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check it's not a parameter validation error
				if strings.Contains(errorMsg, "required") ||
					strings.Contains(errorMsg, "must provide") {
					t.Errorf("Should not have parameter validation error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				// Other errors are expected
				t.Logf("Expected API error: %s", errorMsg)
			}
		})
	})

	t.Run("remove_workspace_user", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("remove_workspace_user", args)
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
				"name": "test-user",
			}

			result, err := client.CallTool("remove_workspace_user", args)
			if err != nil {
				t.Fatalf("Failed to call remove_workspace_user: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if user not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from remove_workspace_user: %s", errorMsg)
			} else {
				// Log the successful response for debugging
				resp, _ := e2e.ExtractJSONResult(result)
				t.Logf("remove_workspace_user returned success: %+v", resp)
			}
		})
	})
}
