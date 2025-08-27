package tools

import (
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestRuntimeTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	t.Run("run_agent", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("run_agent", args)
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

		t.Run("missing_message", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "test-agent",
				// Missing 'message' field
			}

			result, err := client.CallTool("run_agent", args)
			if err != nil {
				t.Fatalf("Failed to call run_agent: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing message field")
			}
			if !strings.Contains(errorMsg, "message") {
				t.Errorf("Expected error to mention 'message', got: %s", errorMsg)
			}
		})
	})

	t.Run("run_job", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("run_job", args)
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
	})

	t.Run("run_model", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("run_model", args)
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

		t.Run("missing_body", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "sandbox-openai",
				// Missing 'body' field
			}

			result, err := client.CallTool("run_model", args)
			if err != nil {
				t.Fatalf("Failed to call run_model: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing body field")
			}
			if !strings.Contains(errorMsg, "body") {
				t.Errorf("Expected error to mention 'body', got: %s", errorMsg)
			}
		})

		t.Run("with_body", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "sandbox-openai",
				"path": "/v1/chat/completions",
				"body": `{"messages":[{"role":"user","content":"What is the capital of France?"}]}`,
			}

			result, err := client.CallTool("run_model", args)
			if err != nil {
				t.Fatalf("Failed to call run_model: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept body parameter even if model not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept body parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from run_model: %s", errorMsg)
			}
		})

		t.Run("with_complex_body", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "sandbox-openai",
				"path": "/v1/chat/completions",
				"body": `{"messages":[{"role":"user","content":"Hello, how are you?"}],"temperature":0.7,"max_tokens":100}`,
			}

			result, err := client.CallTool("run_model", args)
			if err != nil {
				t.Fatalf("Failed to call run_model: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept complex body parameter even if model not found
				if strings.Contains(errorMsg, "required") {
					t.Errorf("Should accept complex body parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from run_model: %s", errorMsg)
			}
		})
	})

	t.Run("run_sandbox", func(t *testing.T) {
		t.Run("missing_required_fields", func(t *testing.T) {
			args := map[string]interface{}{
				// Missing 'name' field
			}

			result, err := client.CallTool("run_sandbox", args)
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

		t.Run("missing_code", func(t *testing.T) {
			args := map[string]interface{}{
				"name": "test-sandbox",
				// Missing 'code' field
			}

			result, err := client.CallTool("run_sandbox", args)
			if err != nil {
				t.Fatalf("Failed to call run_sandbox: %v", err)
			}

			// Check for tool error in result
			isError, errorMsg := e2e.CheckToolError(result)
			if !isError {
				t.Fatal("Expected error for missing code field")
			}
			if !strings.Contains(errorMsg, "code") {
				t.Errorf("Expected error to mention 'code', got: %s", errorMsg)
			}
		})
	})
}
