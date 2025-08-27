package tools

import (
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/e2e"
)

func TestJobsTools(t *testing.T) {
	client := e2e.NewMCPTestClient(t, e2e.TestEnv())
	defer client.Close()

	t.Run("list_jobs", func(t *testing.T) {
		t.Run("basic_list", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("list_jobs", args)
			if err != nil {
				t.Fatalf("Failed to call list_jobs: %v", err)
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
				t.Fatalf("Unexpected error from list_jobs: %s", errorMsg)
			}
		})
	})

	t.Run("get_job", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("get_job", args)
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
			if !strings.Contains(errorMsg, "name") && !strings.Contains(errorMsg, "job ID") {
				t.Errorf("Expected error to mention 'name' or 'job ID', got: %s", errorMsg)
			}
		})

		t.Run("with_name", func(t *testing.T) {
			args := map[string]interface{}{
				"id": "test-job",
			}

			result, err := client.CallTool("get_job", args)
			if err != nil {
				t.Fatalf("Failed to call get_job: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if job not found
				if strings.Contains(errorMsg, "required") && !strings.Contains(errorMsg, "job ID") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from get_job: %s", errorMsg)
			}
		})
	})

	t.Run("delete_job", func(t *testing.T) {
		t.Run("missing_name", func(t *testing.T) {
			args := map[string]interface{}{}

			result, err := client.CallTool("delete_job", args)
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
			if !strings.Contains(errorMsg, "name") && !strings.Contains(errorMsg, "job ID") {
				t.Errorf("Expected error to mention 'name' or 'job ID', got: %s", errorMsg)
			}
		})

		t.Run("with_name", func(t *testing.T) {
			args := map[string]interface{}{
				"id": "test-job",
			}

			result, err := client.CallTool("delete_job", args)
			if err != nil {
				t.Fatalf("Failed to call delete_job: %v", err)
			}

			// Check if the tool returned an error
			isError, errorMsg := e2e.CheckToolError(result)
			if isError {
				// Should accept name parameter even if job not found
				if strings.Contains(errorMsg, "required") && !strings.Contains(errorMsg, "job ID") {
					t.Errorf("Should accept name parameter, got error: %s", errorMsg)
				} else if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "404") ||
					strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") ||
					strings.Contains(errorMsg, "forbidden") || strings.Contains(errorMsg, "invalid") {
					t.Logf("API call failed as expected: %s", errorMsg)
					return
				}
				t.Fatalf("Unexpected error from delete_job: %s", errorMsg)
			} else {
				// Log the successful response for debugging
				resp, _ := e2e.ExtractJSONResult(result)
				t.Logf("delete_job returned success: %+v", resp)
			}
		})
	})
}
