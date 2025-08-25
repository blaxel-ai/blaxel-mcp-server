package agents

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

// RegisterTools registers all agent-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List agents tool
	listAgentsSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"filter": {
				"type": "string",
				"description": "Optional filter string to match agent names"
			}
		}
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("list_agents", "List all agents in the workspace", listAgentsSchema),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse arguments
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}

			result, err := listAgentsHandler(ctx, sdkClient, args)
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

	// Get agent tool
	getAgentSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Name of the agent to retrieve"
			}
		},
		"required": ["name"]
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("get_agent", "Get details of a specific agent", getAgentSchema),
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

			result, err := getAgentHandler(ctx, sdkClient, args)
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

	// Delete agent tool (only if not in readonly mode)
	if !cfg.ReadOnly {
		deleteAgentSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "Name of the agent to delete"
				}
			},
			"required": ["name"]
		}`)

		s.AddTool(
			mcp.NewToolWithRawSchema("delete_agent", "Delete an agent from the workspace", deleteAgentSchema),
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

				result, err := deleteAgentHandler(ctx, sdkClient, args)
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

func listAgentsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
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
	filter, _ := params["filter"].(string)

	var filteredAgents []map[string]interface{}
	for _, agent := range agents {
		// Check if filter matches
		if filter != "" {
			name := ""
			if agent.Metadata != nil && agent.Metadata.Name != nil {
				name = *agent.Metadata.Name
			}
			// Skip if name doesn't contain filter
			if name == "" || !contains(name, filter) {
				continue
			}
		}

		// Build agent info
		agentInfo := map[string]interface{}{
			"name": "",
		}

		if agent.Metadata != nil && agent.Metadata.Name != nil {
			agentInfo["name"] = *agent.Metadata.Name
		}

		if agent.Spec != nil {
			// Add any relevant spec fields here
		}

		filteredAgents = append(filteredAgents, agentInfo)
	}

	return map[string]interface{}{
		"agents": filteredAgents,
		"count":  len(filteredAgents),
	}, nil
}

func getAgentHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	resp, err := sdkClient.GetAgentWithResponse(ctx, name)
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

	return json.RawMessage(jsonData), nil
}

func deleteAgentHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	resp, err := sdkClient.DeleteAgentWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete agent: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return nil, fmt.Errorf("delete agent failed with status %d", resp.StatusCode())
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent '%s' deleted successfully", name),
	}, nil
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
