package jobs

import (
	"context"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
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
func RegisterJobTools(s *server.MCPServer, handler JobHandler, cfg *config.Config) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(JobHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List jobs tool
	listJobsTool := mcp.NewTool("list_jobs",
		mcp.WithTitleAnnotation("List Jobs"),
		mcp.WithDescription("List all jobs in the workspace"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("status",
			mcp.Description("Optional filter by job status"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(listJobsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		status := request.GetString("status", "")

		result, err := activeHandler.ListJobs(ctx, status)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get job tool
	getJobTool := mcp.NewTool("get_job",
		mcp.WithTitleAnnotation("Get Job"),
		mcp.WithDescription("Get details of a specific job"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("ID of the job to retrieve"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		id := request.GetString("id", "")
		if id == "" {
			return mcp.NewToolResultError("job ID is required"), nil
		}

		result, err := activeHandler.GetJob(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Delete job tool
		deleteJobTool := mcp.NewTool("delete_job",
			mcp.WithTitleAnnotation("Delete Job"),
			mcp.WithDescription("Delete a job from the workspace"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID of the job to delete"),
			),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(deleteJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			id := request.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("job ID is required"), nil
			}

			result, err := activeHandler.DeleteJob(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
