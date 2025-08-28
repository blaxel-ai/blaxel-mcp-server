package jobs

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// JobHandler defines the interface for job operations
type JobHandler interface {
	ListJobs(ctx context.Context, status string) ([]byte, error)
	GetJob(ctx context.Context, id string) ([]byte, error)
	DeleteJob(ctx context.Context, id string) ([]byte, error)
}

// JobHandlerWithReadOnly extends JobHandler with readonly capability
type JobHandlerWithReadOnly interface {
	JobHandler
	IsReadOnly() bool
}

// RegisterJobTools registers job tools with the given handler
func RegisterJobTools(s *server.MCPServer, handler JobHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(JobHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List jobs tool
	listJobsTool := mcp.NewTool("list_jobs",
		mcp.WithDescription("List all jobs in the workspace"),
		mcp.WithString("status",
			mcp.Description("Optional filter by job status"),
		),
	)

	s.AddTool(listJobsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		status := request.GetString("status", "")

		result, err := handler.ListJobs(ctx, status)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
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
		id := request.GetString("id", "")
		if id == "" {
			return mcp.NewToolResultError("job ID is required"), nil
		}

		result, err := handler.GetJob(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Delete job tool
		deleteJobTool := mcp.NewTool("delete_job",
			mcp.WithDescription("Delete a job from the workspace"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID of the job to delete"),
			),
		)

		s.AddTool(deleteJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := request.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("job ID is required"), nil
			}

			result, err := handler.DeleteJob(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
