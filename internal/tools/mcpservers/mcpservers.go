package mcpservers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all MCP server-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List MCP servers
	s.AddTool(
		mcp.NewToolWithRawSchema("list_mcp_servers",
			"List all MCP servers (functions) in the workspace",
			json.RawMessage(`{"type": "object", "properties": {"filter": {"type": "string", "description": "Optional filter string"}}}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := listMCPServersHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(err.Error())},
					IsError: true,
				}, nil
			}
			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.NewTextContent(string(jsonResult))},
			}, nil
		},
	)

	// Get MCP server
	s.AddTool(
		mcp.NewToolWithRawSchema("get_mcp_server",
			"Get details of a specific MCP server (function)",
			json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "Name of the MCP server"}}, "required": ["name"]}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := getMCPServerHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(err.Error())},
					IsError: true,
				}, nil
			}
			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.NewTextContent(string(jsonResult))},
			}, nil
		},
	)

	if !cfg.ReadOnly {
		// Create MCP server
		s.AddTool(
			mcp.NewToolWithRawSchema("create_mcp_server",
				"Create an MCP server (function) with flexible integration options",
				json.RawMessage(`{
					"type": "object",
					"properties": {
						"name": {"type": "string", "description": "Name for the MCP server"},
						"integrationConnectionName": {"type": "string", "description": "Existing integration to use"},
						"integrationType": {"type": "string", "description": "Type for new integration (e.g., github)"},
						"secret": {"type": "object", "description": "Secrets for new integration"},
						"config": {"type": "object", "description": "Config for new integration"}
					},
					"required": ["name"]
				}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := createMCPServerHandler(ctx, sdkClient, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				jsonResult, _ := json.MarshalIndent(result, "", "  ")
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(string(jsonResult))},
				}, nil
			},
		)

		// Delete MCP server
		s.AddTool(
			mcp.NewToolWithRawSchema("delete_mcp_server",
				"Delete an MCP server (function) by name",
				json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "Name of the MCP server to delete"}}, "required": ["name"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := deleteMCPServerHandler(ctx, sdkClient, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				jsonResult, _ := json.MarshalIndent(result, "", "  ")
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(string(jsonResult))},
				}, nil
			},
		)
	}
}

func listMCPServersHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	servers, err := sdkClient.ListFunctionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}
	if servers.JSON200 == nil {
		return nil, fmt.Errorf("no MCP servers found")
	}

	// Apply optional filter
	filter, _ := params["filter"].(string)
	var filtered []map[string]interface{}

	for _, server := range *servers.JSON200 {
		if filter != "" {
			name := ""
			if server.Metadata != nil && server.Metadata.Name != nil {
				name = *server.Metadata.Name
			}
			if name == "" || !containsString(name, filter) {
				continue
			}
		}

		item := map[string]interface{}{
			"name": "",
		}

		if server.Metadata != nil && server.Metadata.Name != nil {
			item["name"] = *server.Metadata.Name
		}

		filtered = append(filtered, item)
	}

	return map[string]interface{}{
		"mcp_servers": filtered,
		"count":       len(filtered),
	}, nil
}

func getMCPServerHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("MCP server name is required")
	}

	server, err := sdkClient.GetFunction(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	jsonData, _ := json.MarshalIndent(server, "", "  ")
	return map[string]interface{}{
		"mcp_server": json.RawMessage(jsonData),
	}, nil
}

func createMCPServerHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("MCP server name is required")
	}

	// Check for integration parameters
	integrationConnectionName, hasExisting := params["integrationConnectionName"].(string)
	integrationType, hasNewType := params["integrationType"].(string)

	// Validate integration parameters
	if hasExisting && hasNewType {
		return nil, fmt.Errorf("specify either integrationConnectionName or integrationType, not both")
	}

	if !hasExisting && !hasNewType {
		return nil, fmt.Errorf("must provide either integrationConnectionName (for existing) or integrationType (for new)")
	}

	// If creating new integration, validate required fields
	if hasNewType {
		if integrationType == "" {
			return nil, fmt.Errorf("integrationType cannot be empty")
		}
		// Note: secret and config are optional and their structure depends on the integration type
	}

	// If using existing integration, validate name
	if hasExisting && integrationConnectionName == "" {
		return nil, fmt.Errorf("integrationConnectionName cannot be empty")
	}

	// For now, return not implemented after validation
	return nil, fmt.Errorf("MCP server creation not yet fully implemented")
}

func deleteMCPServerHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("MCP server name is required")
	}

	_, err := sdkClient.DeleteFunctionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete MCP server: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("MCP server '%s' deleted successfully", name),
	}, nil
}

func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && s == substr)
}
