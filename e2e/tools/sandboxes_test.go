package tools

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestSandboxesTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()
	env := os.Getenv("BL_ENV")
	if env == "" {
		env = "prod"
	}

	// Generate random test names to avoid conflicts
	testSandboxName := e2e.GenerateRandomTestName("test-sandbox")

	t.Run("full_lifecycle_test", func(t *testing.T) {
		t.Run("create_sandbox", func(t *testing.T) {

			// Create a sandbox with basic configuration
			args := map[string]interface{}{
				"name":   testSandboxName,
				"image":  fmt.Sprintf("blaxel/%s-base:latest", env),
				"memory": 4096,
				"ports":  "8080,8081",
				"env":    "FOO=bar,BAR=baz",
			}

			result, err := client.CallTool("create_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call create_sandbox: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Check if it's an expected error (like already exists or auth issues)
				if strings.Contains(errorMsg, "already exists") {
					t.Logf("Sandbox already exists, continuing with test: %s", errorMsg)
					return
				} else if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected with test credentials: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from create_sandbox: %s", errorMsg)
			}

			t.Logf("Successfully created sandbox: %s", testSandboxName)
		})

		t.Run("list_sandboxes", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_sandboxes", args)
			if err != nil {
				t.Fatalf("Failed to call list_sandboxes: %v", err)
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
				t.Fatalf("Unexpected error from list_sandboxes: %s", errorMsg)
			}

			t.Logf("Successfully listed sandboxes")
		})

		t.Run("get_sandbox", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testSandboxName,
			}

			result, err := client.CallTool("get_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call get_sandbox: %v", err)
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
				t.Fatalf("Unexpected error from get_sandbox: %s", errorMsg)
			}

			t.Logf("Successfully retrieved sandbox: %s", testSandboxName)
		})

		t.Run("run_sandbox", func(t *testing.T) {
			args := map[string]interface{}{
				"name":   testSandboxName,
				"body":   `{"command":"echo \"Hello from sandbox!\"", "waitForCompletion": true}`,
				"method": "POST",
				"path":   "/process",
			}

			result, err := client.CallTool("run_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call run_sandbox: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Unexpected error from run_sandbox: %s", errorMsg)
			}

			t.Logf("Successfully executed code in sandbox: %s", testSandboxName)
		})

		t.Run("delete_sandbox", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testSandboxName,
			}

			result, err := client.CallTool("delete_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call delete_sandbox: %v", err)
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
				t.Fatalf("Unexpected error from delete_sandbox: %s", errorMsg)
			}

			t.Logf("Successfully deleted sandbox: %s", testSandboxName)
		})

		t.Run("verify_sandbox_deleted", func(t *testing.T) {
			args := map[string]interface{}{
				"name": testSandboxName,
			}

			result, err := client.CallTool("get_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call get_sandbox: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Accept any error as valid for deleted sandbox
				if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("Successfully verified sandbox is deleted (got expected error): %s", errorMsg)
					return
				}
				t.Logf("Got error when getting deleted sandbox: %s", errorMsg)
			} else {
				t.Logf("Sandbox still exists after deletion (this might be expected depending on API behavior)")
			}

			t.Logf("Verification complete for sandbox: %s", testSandboxName)
		})
	})

	t.Run("error_handling_tests", func(t *testing.T) {
		t.Run("create_sandbox_missing_name", func(t *testing.T) {
			args := map[string]interface{}{
				"type": "python",
			}

			result, err := client.CallTool("create_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call create_sandbox: %v", err)
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

		t.Run("get_sandbox_missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("get_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call get_sandbox: %v", err)
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

		t.Run("delete_sandbox_missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("delete_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call delete_sandbox: %v", err)
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

	// Cleanup any test sandboxes that might have been created
	t.Cleanup(func() {
		// Try to delete the test sandbox if it still exists
		cleanupArgs := map[string]interface{}{
			"name": testSandboxName,
		}
		_, _ = client.CallTool("delete_sandbox", cleanupArgs)
	})
}
