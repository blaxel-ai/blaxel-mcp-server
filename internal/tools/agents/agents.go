package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for type safety
type ListAgentsRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListAgentsResponse json.RawMessage

type GetAgentRequest struct {
	Name string `json:"name"`
}

type GetAgentResponse json.RawMessage

type DeleteAgentRequest struct {
	Name string `json:"name"`
}

type DeleteAgentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all agent-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List agents tool
	listAgentsTool := mcp.NewTool("list_agents",
		mcp.WithDescription("List all agents in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string to match agent names"),
		),
	)

	s.AddTool(listAgentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListAgentsRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get filter directly for backward compatibility
			req.Filter = request.GetString("filter", "")
		}

		result, err := listAgentsHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
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
		var req GetAgentRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("agent name is required"), nil
		}

		result, err := getAgentHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	// Delete agent tool (only if not in readonly mode)
	if !cfg.ReadOnly {
		deleteAgentTool := mcp.NewTool("delete_agent",
			mcp.WithDescription("Delete an agent from the workspace"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the agent to delete"),
			),
		)

		s.AddTool(deleteAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeleteAgentRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("agent name is required"), nil
			}

			result, err := deleteAgentHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listAgentsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListAgentsRequest) (*ListAgentsResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := sdkClient.ListAgentsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list agents failed with status %d", resp.StatusCode())
	}

	agents := []sdk.Agent{}
	if resp.JSON200 != nil {
		agents = *resp.JSON200
	}

	// Apply optional filter
	if req.Filter != "" {
		var filtered []sdk.Agent
		for _, agent := range agents {
			if agent.Metadata != nil && agent.Metadata.Name != nil &&
				tools.ContainsString(*agent.Metadata.Name, req.Filter) {
				filtered = append(filtered, agent)
			}
		}
		agents = filtered
	}

	// Format the agents using the new formatter
	formatted := tools.FormatAgents(agents)
	jsonData, err := json.Marshal(formatted)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal formatted agents: %w", err)
	}

	response := ListAgentsResponse(jsonData)
	return &response, nil
}

func getAgentHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetAgentRequest) (*GetAgentResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := sdkClient.GetAgentWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get agent failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("agent not found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(resp.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format agent data: %w", err)
	}

	response := GetAgentResponse(jsonData)
	return &response, nil
}

func deleteAgentHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteAgentRequest) (*DeleteAgentResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := sdkClient.DeleteAgentWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete agent: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return nil, fmt.Errorf("delete agent failed with status %d", resp.StatusCode())
	}

	return &DeleteAgentResponse{
		Success: true,
		Message: fmt.Sprintf("Agent '%s' deleted successfully", req.Name),
	}, nil
}
