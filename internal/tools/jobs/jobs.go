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

// Request/Response types for type safety
type ListJobsRequest struct {
	Status string `json:"status,omitempty"`
}

type ListJobsResponse struct {
	Jobs  []JobInfo `json:"jobs"`
	Count int       `json:"count"`
}

type JobInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type GetJobRequest struct {
	ID string `json:"id"`
}

type GetJobResponse struct {
	Job json.RawMessage `json:"job"`
}

type CreateJobRequest struct {
	Name        string `json:"name"`
	AgentName   string `json:"agentName,omitempty"`
	Description string `json:"description,omitempty"`
}

type CreateJobResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Job     map[string]interface{} `json:"job"`
}

type DeleteJobRequest struct {
	ID string `json:"id"`
}

type DeleteJobResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all job-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List jobs tool
	listJobsTool := mcp.NewTool("list_jobs",
		mcp.WithDescription("List all jobs in the workspace"),
		mcp.WithString("status",
			mcp.Description("Optional filter by job status"),
		),
	)

	s.AddTool(listJobsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListJobsRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get status directly for backward compatibility
			req.Status = request.GetString("status", "")
		}

		result, err := listJobsHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// Get job tool
	getJobTool := mcp.NewTool("get_job",
		mcp.WithDescription("Get details of a specific job"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("ID of the job to retrieve"),
		),
	)

	s.AddTool(getJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req GetJobRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.ID == "" {
			return mcp.NewToolResultError("job ID is required"), nil
		}

		result, err := getJobHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// Delete job tool (only if not in readonly mode)
	if !cfg.ReadOnly {
		deleteJobTool := mcp.NewTool("delete_job",
			mcp.WithDescription("Delete a job from the workspace"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID of the job to delete"),
			),
		)

		s.AddTool(deleteJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeleteJobRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.ID == "" {
				return mcp.NewToolResultError("job ID is required"), nil
			}

			result, err := deleteJobHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listJobsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListJobsRequest) (*ListJobsResponse, error) {
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
	statusFilter := req.Status

	var filteredJobs []JobInfo
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
		jobInfo := JobInfo{}

		if job.Metadata != nil && job.Metadata.Name != nil {
			jobInfo.ID = *job.Metadata.Name
			jobInfo.Name = *job.Metadata.Name
		}

		if job.Status != nil {
			jobInfo.Status = *job.Status
		}

		filteredJobs = append(filteredJobs, jobInfo)
	}

	return &ListJobsResponse{
		Jobs:  filteredJobs,
		Count: len(filteredJobs),
	}, nil
}

func getJobHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetJobRequest) (*GetJobResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := sdkClient.GetJobWithResponse(ctx, req.ID)
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
		jsonData, _ := json.MarshalIndent(job, "", "  ")
		return &GetJobResponse{
			Job: json.RawMessage(jsonData),
		}, nil
	}

	return nil, fmt.Errorf("unexpected response type")
}

func deleteJobHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteJobRequest) (*DeleteJobResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := sdkClient.DeleteJobWithResponse(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return nil, fmt.Errorf("delete job failed with status %d", resp.StatusCode())
	}

	return &DeleteJobResponse{
		Success: true,
		Message: fmt.Sprintf("Job '%s' deleted successfully", req.ID),
	}, nil
}
