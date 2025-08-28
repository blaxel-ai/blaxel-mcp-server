package agents

import (
	"context"

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
func RegisterAgentTools(s *server.MCPServer, handler AgentHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(AgentHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()
	// List agents tool
	listAgentsTool := mcp.NewTool("list_agents",
		mcp.WithDescription("List all agents in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string to match agent names"),
		),
	)

	s.AddTool(listAgentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := request.GetString("filter", "")

		result, err := handler.ListAgents(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get agent tool
	getAgentTool := mcp.NewTool("get_agent",
		mcp.WithDescription("Get details of a specific agent"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the agent to retrieve"),
		),
	)

	s.AddTool(getAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("agent name is required"), nil
		}

		result, err := handler.GetAgent(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Delete agent tool (only if not in readonly mode)
	if !isReadOnly {
		deleteAgentTool := mcp.NewTool("delete_agent",
			mcp.WithDescription("Delete an agent from the workspace"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the agent to delete"),
			),
		)

		s.AddTool(deleteAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("agent name is required"), nil
			}

			result, err := handler.DeleteAgent(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
