package integrations

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

// RegisterTools registers all integration-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List integrations
	s.AddTool(
		mcp.NewToolWithRawSchema("list_integrations",
			"List all integration connections in the workspace",
			json.RawMessage(`{"type": "object", "properties": {"filter": {"type": "string", "description": "Optional filter string"}}}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := listIntegrationsHandler(ctx, sdkClient, args)
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

	// Get integration
	s.AddTool(
		mcp.NewToolWithRawSchema("get_integration",
			"Get details of a specific integration connection",
			json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "Name of the integration"}}, "required": ["name"]}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := getIntegrationHandler(ctx, sdkClient, args)
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
		// Delete integration
		s.AddTool(
			mcp.NewToolWithRawSchema("delete_integration",
				"Delete an integration connection by name",
				json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "Name of the integration to delete"}}, "required": ["name"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := deleteIntegrationHandler(ctx, sdkClient, args)
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

func listIntegrationsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	integrations, err := sdkClient.ListIntegrationConnectionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}
	if integrations.JSON200 == nil {
		return nil, fmt.Errorf("no integrations found")
	}

	// Apply optional filter
	filter, _ := params["filter"].(string)
	var filtered []map[string]interface{}

	for _, integration := range *integrations.JSON200 {
		if filter != "" {
			name := ""
			if integration.Metadata != nil && integration.Metadata.Name != nil {
				name = *integration.Metadata.Name
			}
			if name == "" || !containsString(name, filter) {
				continue
			}
		}

		item := map[string]interface{}{
			"name": "",
			"type": "",
		}

		if integration.Metadata != nil && integration.Metadata.Name != nil {
			item["name"] = *integration.Metadata.Name
		}

		if integration.Spec != nil && integration.Spec.Integration != nil {
			item["type"] = *integration.Spec.Integration
		}

		filtered = append(filtered, item)
	}

	return map[string]interface{}{
		"integrations": filtered,
		"count":        len(filtered),
	}, nil
}

func getIntegrationHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("integration name is required")
	}

	integration, err := sdkClient.GetIntegrationConnectionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration: %w", err)
	}

	jsonData, _ := json.MarshalIndent(integration, "", "  ")
	return map[string]interface{}{
		"integration": json.RawMessage(jsonData),
	}, nil
}

func deleteIntegrationHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("integration name is required")
	}

	_, err := sdkClient.DeleteIntegrationConnectionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete integration: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Integration '%s' deleted successfully", name),
	}, nil
}

func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && s == substr)
}
