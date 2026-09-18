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
		mcp.WithToolTitle("List Jobs"),
		mcp.WithTitleAnnotation("List Jobs"),
		mcp.WithDescription("List the Blaxel batch jobs in the workspace with each job's status, container image and resources. Pass status to keep only jobs in one state, using the exact status string that get_job reports. See https://docs.blaxel.ai/Jobs/Overview."),
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
		mcp.WithToolTitle("Get Job"),
		mcp.WithTitleAnnotation("Get Job"),
		mcp.WithDescription("Get one Blaxel batch job, identified by the id argument, which is the job name that list_jobs reports. Returns its full definition: container image, resources, environment, triggers and current status. This describes the deployed job, not the outcome of an individual execution. See https://docs.blaxel.ai/Jobs/Overview."),
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
			mcp.WithToolTitle("Delete Job"),
			mcp.WithTitleAnnotation("Delete Job"),
			mcp.WithDescription("Permanently delete a Blaxel batch job, identified by the id argument, which is the job name that list_jobs reports. Executions already running are not waited for. See https://docs.blaxel.ai/Jobs/Overview."),
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
