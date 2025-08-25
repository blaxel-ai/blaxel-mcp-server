package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all job-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List jobs tool
	listJobsSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"description": "Optional filter by job status"
			}
		}
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("list_jobs", "List all jobs in the workspace", listJobsSchema),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse arguments
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}

			result, err := listJobsHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(err.Error()),
					},
					IsError: true,
				}, nil
			}

			// Convert result to JSON
			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(string(jsonResult)),
				},
			}, nil
		},
	)

	// Get job tool
	getJobSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "ID of the job to retrieve"
			}
		},
		"required": ["id"]
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("get_job", "Get details of a specific job", getJobSchema),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse arguments
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Invalid arguments"),
					},
					IsError: true,
				}, nil
			}

			result, err := getJobHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(err.Error()),
					},
					IsError: true,
				}, nil
			}

			// Convert result to JSON
			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(string(jsonResult)),
				},
			}, nil
		},
	)

	// Delete job tool (only if not in readonly mode)
	if !cfg.ReadOnly {
		deleteJobSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {
					"type": "string",
					"description": "ID of the job to delete"
				}
			},
			"required": ["id"]
		}`)

		s.AddTool(
			mcp.NewToolWithRawSchema("delete_job", "Delete a job from the workspace", deleteJobSchema),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				// Parse arguments
				args, ok := request.Params.Arguments.(map[string]interface{})
				if !ok {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent("Invalid arguments"),
						},
						IsError: true,
					}, nil
				}

				result, err := deleteJobHandler(ctx, sdkClient, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent(err.Error()),
						},
						IsError: true,
					}, nil
				}

				// Convert result to JSON
				jsonResult, _ := json.MarshalIndent(result, "", "  ")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(string(jsonResult)),
					},
				}, nil
			},
		)
	}
}

func listJobsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := sdkClient.ListJobsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list jobs failed with status %d", resp.StatusCode())
	}

	jobs := []sdk.Job{}
	if resp.JSON200 != nil {
		jobs = *resp.JSON200
	}

	// Apply optional status filter
	statusFilter, _ := params["status"].(string)

	var filteredJobs []map[string]interface{}
	for _, job := range jobs {
		// Check if status filter matches
		if statusFilter != "" {
			status := ""
			if job.Status != nil {
				status = *job.Status
			}
			// Skip if status doesn't match filter
			if status == "" || !strings.EqualFold(status, statusFilter) {
				continue
			}
		}

		// Build job info
		jobInfo := map[string]interface{}{
			"id":     "",
			"status": "",
		}

		if job.Metadata != nil && job.Metadata.Name != nil {
			jobInfo["id"] = *job.Metadata.Name
		}

		if job.Status != nil {
			jobInfo["status"] = *job.Status
		}

		if job.Spec != nil {
			// Add any relevant spec fields here
		}

		filteredJobs = append(filteredJobs, jobInfo)
	}

	return map[string]interface{}{
		"jobs":  filteredJobs,
		"count": len(filteredJobs),
	}, nil
}

func getJobHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	id, ok := params["id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	resp, err := sdkClient.GetJobWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get job failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("job not found")
	}

	// SDK bug workaround: GetJobResponse returns *Model instead of *Job
	// Convert the Model to Job (they have similar structure)
	if model, ok := interface{}(resp.JSON200).(*sdk.Model); ok {
		job := &sdk.Job{
			Metadata: model.Metadata,
			Spec:     &sdk.JobSpec{
				// Convert ModelSpec to JobSpec if needed
			},
			Status: model.Status,
		}
		return job, nil
	}

	return nil, fmt.Errorf("unexpected response type")
}

func deleteJobHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	id, ok := params["id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	resp, err := sdkClient.DeleteJobWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return nil, fmt.Errorf("delete job failed with status %d", resp.StatusCode())
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Job '%s' deleted successfully", id),
	}, nil
}
