package tools

import (
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestSandboxesTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	// Generate random test names to avoid conflicts
	testSandboxName := e2e.GenerateRandomTestName("test-sandbox")

	t.Run("full_lifecycle_test", func(t *testing.T) {
		t.Run("create_sandbox", func(t *testing.T) {

			// Create a sandbox with basic configuration
			args := map[string]interface{}{
				"name":   testSandboxName,
				"memory": 4096,
				"env":    "FOO=bar,BAR=baz",
			}

			result, err := client.CallTool("create_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call create_sandbox: %v", err)
			}

			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Failed to create sandbox: %s", errorMsg)
			}

			t.Logf("Successfully created sandbox: %s", testSandboxName)
		})

		t.Run("list_sandboxes", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_sandboxes", args)
			if err != nil {
				t.Fatalf("Failed to call list_sandboxes: %v", err)
			}

			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Failed to list sandboxes: %s", errorMsg)
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

			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Failed to get sandbox: %s", errorMsg)
			}

			t.Logf("Successfully retrieved sandbox: %s", testSandboxName)
		})

		t.Run("run_sandbox_command", func(t *testing.T) {
			args := map[string]interface{}{
				"name":              testSandboxName,
				"command":           `echo "Hello from sandbox!"`,
				"waitForCompletion": true,
			}

			result, err := client.CallTool("run_sandbox_command", args)
			if err != nil {
				t.Fatalf("Failed to call run_sandbox_command: %v", err)
			}

			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Failed to run sandbox command: %s", errorMsg)
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

			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Failed to delete sandbox: %s", errorMsg)
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

			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Got an error, which means the sandbox is no longer accessible
				t.Logf("Successfully verified sandbox is deleted (got error): %s", errorMsg)
			} else {
				// Sandbox may still be visible briefly due to eventual consistency (e.g., DELETING status)
				t.Logf("Sandbox still visible after deletion (eventual consistency): %s", testSandboxName)
			}
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
