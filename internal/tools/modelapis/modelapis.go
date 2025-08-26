package modelapis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for type safety
type ListModelAPIsRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListModelAPIsResponse struct {
	ModelAPIs []ModelAPIInfo `json:"model_apis"`
	Count     int            `json:"count"`
}

type ModelAPIInfo struct {
	Name string `json:"name"`
}

type GetModelAPIRequest struct {
	Name string `json:"name"`
}

type GetModelAPIResponse struct {
	ModelAPI json.RawMessage `json:"model_api"`
}

type CreateModelAPIRequest struct {
	Name                      string `json:"name"`
	Model                     string `json:"model,omitempty"`
	Endpoint                  string `json:"endpoint,omitempty"`
	IntegrationConnectionName string `json:"integrationConnectionName,omitempty"`
	Provider                  string `json:"provider,omitempty"`
	APIKey                    string `json:"apiKey,omitempty"`
}

type CreateModelAPIResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	ModelAPI map[string]interface{} `json:"model_api"`
}

type DeleteModelAPIRequest struct {
	Name string `json:"name"`
}

type DeleteModelAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all model API-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List model APIs
	listModelAPIsTool := mcp.NewTool("list_model_apis",
		mcp.WithDescription("List all model APIs in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listModelAPIsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListModelAPIsRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get filter directly for backward compatibility
			req.Filter = request.GetString("filter", "")
		}

		result, err := listModelAPIsHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// Get model API
	getModelAPITool := mcp.NewTool("get_model_api",
		mcp.WithDescription("Get details of a specific model API"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the model API"),
		),
	)

	s.AddTool(getModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req GetModelAPIRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("model API name is required"), nil
		}

		result, err := getModelAPIHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	if !cfg.ReadOnly {
		// Create model API
		createModelAPITool := mcp.NewTool("create_model_api",
			mcp.WithDescription("Create a model API with flexible integration options"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the model API"),
			),
			mcp.WithString("integrationConnectionName",
				mcp.Description("Existing integration to use"),
			),
			mcp.WithString("provider",
				mcp.Description("Provider for new integration (e.g., openai)"),
			),
			mcp.WithString("apiKey",
				mcp.Description("API key for new integration"),
			),
			mcp.WithString("model",
				mcp.Description("Model identifier"),
			),
			mcp.WithString("endpoint",
				mcp.Description("Optional endpoint URL"),
			),
			mcp.WithObject("config",
				mcp.Description("Additional configuration"),
			),
		)

		s.AddTool(createModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateModelAPIRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("model API name is required"), nil
			}

			result, err := createModelAPIHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Delete model API
		deleteModelAPITool := mcp.NewTool("delete_model_api",
			mcp.WithDescription("Delete a model API by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the model API to delete"),
			),
		)

		s.AddTool(deleteModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeleteModelAPIRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("model API name is required"), nil
			}

			result, err := deleteModelAPIHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listModelAPIsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListModelAPIsRequest) (*ListModelAPIsResponse, error) {
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
	var filtered []ModelAPIInfo

	for _, model := range *models.JSON200 {
		if req.Filter != "" {
			name := ""
			if model.Metadata != nil && model.Metadata.Name != nil {
				name = *model.Metadata.Name
			}
			if name == "" || !containsString(name, req.Filter) {
				continue
			}
		}

		item := ModelAPIInfo{}
		if model.Metadata != nil && model.Metadata.Name != nil {
			item.Name = *model.Metadata.Name
		}

		filtered = append(filtered, item)
	}

	return &ListModelAPIsResponse{
		ModelAPIs: filtered,
		Count:     len(filtered),
	}, nil
}

func getModelAPIHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetModelAPIRequest) (*GetModelAPIResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	model, err := sdkClient.GetModelWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get model API: %w", err)
	}
	if model.JSON200 == nil {
		return nil, fmt.Errorf("no model API found")
	}

	jsonData, _ := json.MarshalIndent(*model.JSON200, "", "  ")
	return &GetModelAPIResponse{
		ModelAPI: json.RawMessage(jsonData),
	}, nil
}

func createModelAPIHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req CreateModelAPIRequest) (*CreateModelAPIResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Check for integration parameters
	hasExisting := req.IntegrationConnectionName != ""
	hasProvider := req.Provider != ""
	hasApiKey := req.APIKey != ""

	// Build model API request
	modelData := sdk.CreateModelJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &req.Name,
		},
		Spec: &sdk.ModelSpec{},
	}

	// Handle integration configuration
	if hasExisting {
		// Use existing integration connection
		if req.IntegrationConnectionName == "" {
			return nil, fmt.Errorf("integrationConnectionName cannot be empty")
		}
		connections := sdk.IntegrationConnectionsList{req.IntegrationConnectionName}
		modelData.Spec.IntegrationConnections = &connections
	} else if hasProvider {
		// For new integration with provider, we need to create it first, then reference it
		// For now, return an informative error
		if req.Provider == "" {
			return nil, fmt.Errorf("provider cannot be empty")
		}
		if !hasApiKey || req.APIKey == "" {
			return nil, fmt.Errorf("api key is required when specifying provider")
		}
		return nil, fmt.Errorf("inline integration creation is not supported. Please create the integration first using 'create_integration', then reference it by name")
	} else {
		return nil, fmt.Errorf("must provide integrationConnectionName to reference an existing integration")
	}

	// Create the model API
	modelResp, err := sdkClient.CreateModelWithResponse(ctx, modelData)
	if err != nil {
		return nil, fmt.Errorf("failed to create model API: %w", err)
	}

	if modelResp.JSON200 == nil {
		if modelResp.StatusCode() == 409 {
			return nil, fmt.Errorf("model API with name '%s' already exists", req.Name)
		}
		return nil, fmt.Errorf("failed to create model API with status %d", modelResp.StatusCode())
	}

	result := &CreateModelAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Model API '%s' created successfully", req.Name),
		ModelAPI: map[string]interface{}{
			"name": req.Name,
		},
	}

	// Add details to result
	if hasExisting {
		result.ModelAPI["integrationConnection"] = req.IntegrationConnectionName
	} else if hasProvider {
		result.ModelAPI["provider"] = req.Provider
	}

	if req.Model != "" {
		result.ModelAPI["model"] = req.Model
	}

	return result, nil
}

func deleteModelAPIHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteModelAPIRequest) (*DeleteModelAPIResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := sdkClient.DeleteModelWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete model API: %w", err)
	}

	return &DeleteModelAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Model API '%s' deleted successfully", req.Name),
	}, nil
}

func containsString(s, substr string) bool {
	return len(substr) == 0 || strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
