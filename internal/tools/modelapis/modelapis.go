package modelapis

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

// RegisterTools registers all model API-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List model APIs
	s.AddTool(
		mcp.NewToolWithRawSchema("list_model_apis",
			"List all model APIs in the workspace",
			json.RawMessage(`{"type": "object", "properties": {"filter": {"type": "string", "description": "Optional filter string"}}}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := listModelAPIsHandler(ctx, sdkClient, args)
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

	// Get model API
	s.AddTool(
		mcp.NewToolWithRawSchema("get_model_api",
			"Get details of a specific model API",
			json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "Name of the model API"}}, "required": ["name"]}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := getModelAPIHandler(ctx, sdkClient, args)
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
		// Create model API
		s.AddTool(
			mcp.NewToolWithRawSchema("create_model_api",
				"Create a model API with flexible integration options",
				json.RawMessage(`{
					"type": "object",
					"properties": {
						"name": {"type": "string", "description": "Name for the model API"},
						"integrationConnectionName": {"type": "string", "description": "Existing integration to use"},
						"provider": {"type": "string", "description": "Provider for new integration (e.g., openai)"},
						"apiKey": {"type": "string", "description": "API key for new integration"},
						"model": {"type": "string", "description": "Model identifier"},
						"endpoint": {"type": "string", "description": "Optional endpoint URL"}
					},
					"required": ["name"]
				}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := createModelAPIHandler(ctx, sdkClient, args)
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

		// Delete model API
		s.AddTool(
			mcp.NewToolWithRawSchema("delete_model_api",
				"Delete a model API by name",
				json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "description": "Name of the model API to delete"}}, "required": ["name"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := deleteModelAPIHandler(ctx, sdkClient, args)
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

func listModelAPIsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	models, err := sdkClient.ListModelsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list model APIs: %w", err)
	}
	if models.JSON200 == nil {
		return nil, fmt.Errorf("no model APIs found")
	}

	// Apply optional filter
	filter, _ := params["filter"].(string)
	var filtered []map[string]interface{}

	for _, model := range *models.JSON200 {
		if filter != "" {
			name := ""
			if model.Metadata != nil && model.Metadata.Name != nil {
				name = *model.Metadata.Name
			}
			if name == "" || !containsString(name, filter) {
				continue
			}
		}

		item := map[string]interface{}{
			"name": "",
		}

		if model.Metadata != nil && model.Metadata.Name != nil {
			item["name"] = *model.Metadata.Name
		}

		filtered = append(filtered, item)
	}

	return map[string]interface{}{
		"model_apis": filtered,
		"count":      len(filtered),
	}, nil
}

func getModelAPIHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("model API name is required")
	}

	model, err := sdkClient.GetModelWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get model API: %w", err)
	}
	if model.JSON200 == nil {
		return nil, fmt.Errorf("no model API found")
	}

	jsonData, _ := json.MarshalIndent(*model.JSON200, "", "  ")
	return map[string]interface{}{
		"model_api": json.RawMessage(jsonData),
	}, nil
}

func createModelAPIHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("model API name is required")
	}

	// Check for integration parameters
	integrationConnectionName, hasExisting := params["integrationConnectionName"].(string)
	provider, hasProvider := params["provider"].(string)
	apiKey, hasApiKey := params["apiKey"].(string)

	// If using new integration with provider, apiKey is required
	if hasProvider {
		if provider == "" {
			return nil, fmt.Errorf("provider cannot be empty")
		}
		if !hasApiKey || apiKey == "" {
			return nil, fmt.Errorf("api key is required when creating new integration")
		}
	}

	// If using existing integration, validate name
	if hasExisting && integrationConnectionName == "" {
		return nil, fmt.Errorf("integrationConnectionName cannot be empty")
	}

	// For now, return not implemented after validation
	return nil, fmt.Errorf("model API creation not yet fully implemented")
}

func deleteModelAPIHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("model API name is required")
	}

	_, err := sdkClient.DeleteModelWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete model API: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Model API '%s' deleted successfully", name),
	}, nil
}

func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && s == substr)
}
