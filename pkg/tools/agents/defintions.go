package agents

import (
	"context"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AgentHandler defines the interface for agent operations
type AgentHandler interface {
	ListAgents(ctx context.Context, filter string) ([]byte, error)
	GetAgent(ctx context.Context, name string) ([]byte, error)
	DeleteAgent(ctx context.Context, name string) ([]byte, error)
}

// AgentHandlerWithReadOnly extends AgentHandler with readonly capability
type AgentHandlerWithReadOnly interface {
	AgentHandler
	IsReadOnly() bool
}

// RegisterAgentTools registers agent tools with the given handler
func RegisterAgentTools(s *server.MCPServer, handler AgentHandler, cfg *config.Config) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(AgentHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()
	// List agents tool
	listAgentsTool := mcp.NewTool("list_agents",
		mcp.WithToolTitle("List Agents"),
		mcp.WithTitleAnnotation("List Agents"),
		mcp.WithDescription("List the Blaxel agents deployed in the workspace with each agent's status and runtime settings. Pass filter to keep only agents whose name contains a case-insensitive substring. Use this to discover the agent name that run_agent and get_agent need. See https://docs.blaxel.ai/Agents/Overview."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("filter",
			mcp.Description("Optional filter string to match agent names"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(listAgentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		filter := request.GetString("filter", "")

		result, err := activeHandler.ListAgents(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get agent tool
	getAgentTool := mcp.NewTool("get_agent",
		mcp.WithToolTitle("Get Agent"),
		mcp.WithTitleAnnotation("Get Agent"),
		mcp.WithDescription("Get one Blaxel agent by name, returning its full definition: runtime, container image, resources, environment and current deployment status. Call this before run_agent when you need the agent's configuration or to confirm it is deployed. See https://docs.blaxel.ai/Agents/Overview."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the agent to retrieve"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("agent name is required"), nil
		}

		result, err := activeHandler.GetAgent(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Delete agent tool (only if not in readonly mode)
	if !isReadOnly {
		deleteAgentTool := mcp.NewTool("delete_agent",
			mcp.WithToolTitle("Delete Agent"),
			mcp.WithTitleAnnotation("Delete Agent"),
			mcp.WithDescription("Permanently delete a Blaxel agent by name. Calls to run_agent for that name stop working. This removes the deployed agent, not the source code it was deployed from. See https://docs.blaxel.ai/Agents/Overview."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the agent to delete"),
			),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(deleteAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("agent name is required"), nil
			}

			result, err := activeHandler.DeleteAgent(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
