package tools

import (
	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
	"strings"
	"testing"
)

func TestLocalTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	t.Run("local_quick_start_guide", func(t *testing.T) {
		t.Run("basic_call", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("local_quick_start_guide", args)
			if err != nil {
				t.Fatalf("Failed to call local_quick_start_guide: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Unexpected error from local_quick_start_guide: %s", errorMsg)
			}

			// This should return a guide text
			if len(result.Content) == 0 {
				t.Error("Expected content from local_quick_start_guide")
			}
		})
	})

	t.Run("local_list_templates", func(t *testing.T) {
		t.Run("basic_list", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("local_list_templates", args)
			if err != nil {
				t.Fatalf("Failed to call local_list_templates: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				t.Fatalf("Unexpected error from local_list_templates: %s", errorMsg)
			}

			// This should return a list of templates
			if len(result.Content) == 0 {
				t.Error("Expected content from local_list_templates")
			}
		})
	})

	t.Run("local_create_agent", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("local_create_agent", args)
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
				"name": "test-local-agent",
			}

			result, err := client.CallTool("local_create_agent", args)
			if err != nil {
				t.Fatalf("Failed to call local_create_agent: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_create_agent: %s", errorMsg)
				}
			}
		})

		t.Run("with_template", func(t *testing.T) {
			args := map[string]interface{}{
				"name":     "test-local-agent-template",
				"template": "basic",
			}

			result, err := client.CallTool("local_create_agent", args)
			if err != nil {
				t.Fatalf("Failed to call local_create_agent: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept template parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept template parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_create_agent: %s", errorMsg)
				}
			}
		})
	})

	t.Run("local_create_job", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("local_create_job", args)
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
				"name": "test-local-job",
			}

			result, err := client.CallTool("local_create_job", args)
			if err != nil {
				t.Fatalf("Failed to call local_create_job: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_create_job: %s", errorMsg)
				}
			}
		})
	})

	t.Run("local_create_mcp_server", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("local_create_mcp_server", args)
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
				"name": "test-local-mcp-server",
			}

			result, err := client.CallTool("local_create_mcp_server", args)
			if err != nil {
				t.Fatalf("Failed to call local_create_mcp_server: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_create_mcp_server: %s", errorMsg)
				}
			}
		})
	})

	t.Run("local_create_sandbox", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("local_create_sandbox", args)
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
				"name": "test-local-sandbox",
			}

			result, err := client.CallTool("local_create_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call local_create_sandbox: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_create_sandbox: %s", errorMsg)
				}
			}
		})
	})

	t.Run("local_deploy_directory", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'directory' field
			}

			result, err := client.CallTool("local_deploy_directory", args)
			if err != nil {
				// Check if it's an error that mentions "directory"
				if strings.Contains(err.Error(), "directory") {
					return // Expected error
				}
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing directory field")
			}
			if !strings.Contains(errorMsg, "directory") {
				t.Errorf("Expected error to mention 'directory', got: %s", errorMsg)
			}
		})

		t.Run("with_directory", func(t *testing.T) {
			args := map[string]interface{}{
				"directory": "./test-dir",
			}

			result, err := client.CallTool("local_deploy_directory", args)
			if err != nil {
				t.Fatalf("Failed to call local_deploy_directory: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept directory parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept directory parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_deploy_directory: %s", errorMsg)
				}
			}
		})
	})

	t.Run("local_run_deployed_resource", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("local_run_deployed_resource", args)
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
				"name": "test-deployed-resource",
			}

			result, err := client.CallTool("local_run_deployed_resource", args)
			if err != nil {
				t.Fatalf("Failed to call local_run_deployed_resource: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_run_deployed_resource: %s", errorMsg)
				}
			}
		})

		t.Run("with_input", func(t *testing.T) {
			args := map[string]interface{}{
				"name":  "test-deployed-resource",
				"input": "test input data",
			}

			result, err := client.CallTool("local_run_deployed_resource", args)
			if err != nil {
				t.Fatalf("Failed to call local_run_deployed_resource: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept input parameter
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept input parameter, got error: %s", errorMsg)
				} else {
					t.Logf("Expected error from local_run_deployed_resource: %s", errorMsg)
				}
			}
		})
	})
}
