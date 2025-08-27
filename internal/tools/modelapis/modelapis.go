package modelapis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/logger"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/utils"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for type safety
type ListModelAPIsRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListModelAPIsResponse json.RawMessage

type GetModelAPIRequest struct {
	Name string `json:"name"`
}

type GetModelAPIResponse json.RawMessage

type CreateModelAPIRequest struct {
	Name                      string `json:"name"`
	Model                     string `json:"model,omitempty"`
	Endpoint                  string `json:"endpoint,omitempty"`
	IntegrationConnectionName string `json:"integrationConnectionName,omitempty"`
	Provider                  string `json:"provider,omitempty"`
	APIKey                    string `json:"apiKey,omitempty"`
	WaitForCompletion         string `json:"waitForCompletion,omitempty"`
}

type CreateModelAPIResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	ModelAPI map[string]interface{} `json:"model_api"`
}

type DeleteModelAPIRequest struct {
	Name              string `json:"name"`
	WaitForCompletion string `json:"waitForCompletion,omitempty"`
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
		logger.Warnf("Failed to initialize SDK client: %v", err)
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

		return mcp.NewToolResultText(string(*result)), nil
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

		return mcp.NewToolResultText(string(*result)), nil
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
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the model API to reach a final status (true/false, default: true)"),
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
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the model API to be fully deleted (true/false, default: true)"),
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

	resp, err := sdkClient.ListModelsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list model APIs: %w", err)
	}

	models := []sdk.Model{}
	if resp.JSON200 != nil {
		models = *resp.JSON200
	}

	// Apply filter if requested
	if req.Filter != "" {
		var filtered []sdk.Model
		for _, model := range models {
			if model.Metadata != nil && model.Metadata.Name != nil &&
				tools.ContainsString(*model.Metadata.Name, req.Filter) {
				filtered = append(filtered, model)
			}
		}
		models = filtered
	}

	// Format the models using the new formatter
	formatted := tools.FormatModels(models)
	jsonData, err := json.Marshal(formatted)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal formatted models: %w", err)
	}

	response := ListModelAPIsResponse(jsonData)
	return &response, nil
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
	response := GetModelAPIResponse(jsonData)
	return &response, nil
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
		Spec: &sdk.ModelSpec{
			Runtime: &sdk.Runtime{
				Model: &req.Model,
			},
		},
	}

	// Handle integration configuration
	var integrationName string
	if hasExisting {
		// Use existing integration connection
		if req.IntegrationConnectionName == "" {
			return nil, fmt.Errorf("integrationConnectionName cannot be empty")
		}
		integrationName = req.IntegrationConnectionName
	} else if hasProvider {
		if !hasApiKey {
			return nil, fmt.Errorf("api key is required when specifying provider")
		}

		// Generate a unique name for the integration
		integrationName = fmt.Sprintf("%s-%s-integration", req.Name, req.Provider)

		// Create the integration
		integrationData := sdk.CreateIntegrationConnectionJSONRequestBody{
			Metadata: &sdk.Metadata{
				Name: &integrationName,
			},
			Spec: &sdk.IntegrationConnectionSpec{
				Integration: &req.Provider, // Provider is the integration type
			},
		}

		// Add API key to secrets
		secrets := map[string]string{
			"apiKey": req.APIKey,
		}
		integrationData.Spec.Secret = &secrets

		// Create the integration
		integrationResp, err := sdkClient.CreateIntegrationConnectionWithResponse(ctx, integrationData)
		if err != nil {
			return nil, fmt.Errorf("failed to create inline integration: %w", err)
		}

		if integrationResp.StatusCode() >= 400 {
			if integrationResp.StatusCode() == 409 {
				// Integration might already exist, try to use it
				logger.Printf("Integration '%s' already exists, will attempt to use it", integrationName)
			} else {
				return nil, fmt.Errorf("failed to create integration with status %d", integrationResp.StatusCode())
			}
		}
	} else {
		return nil, fmt.Errorf("must provide either integrationConnectionName to reference an existing integration or provider with apiKey to create a new one")
	}

	// Set the integration connection on the model
	if integrationName != "" {
		connections := sdk.IntegrationConnectionsList{integrationName}

		modelData.Spec.IntegrationConnections = &connections
		if req.Provider != "" {
			modelData.Spec.Runtime.Type = &req.Provider
		} else {
			response, err := sdkClient.GetIntegrationConnectionWithResponse(ctx, integrationName)
			if err != nil {
				return nil, fmt.Errorf("failed to get integration connection: %w", err)
			}
			if response.JSON200 == nil {
				return nil, fmt.Errorf("no integration connection found")
			}
			modelData.Spec.Runtime.Type = response.JSON200.Spec.Integration
		}
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

	// Check if we should wait for completion
	waitForCompletion := true // default to true
	if req.WaitForCompletion != "" {
		waitForCompletion = req.WaitForCompletion == "true"
	}

	// Wait for the model API to reach a final status if requested
	if waitForCompletion {
		logger.Printf("Waiting for model API '%s' to deploy...", req.Name)
		checker := utils.NewModelAPIStatusChecker(sdkClient)
		err = utils.WaitForResourceStatus(ctx, req.Name, checker)
		if err != nil {
			// Even if status waiting fails, we still created the model API
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: Model API created but status check failed: %v", err)
			result := &CreateModelAPIResponse{
				Success: true,
				Message: fmt.Sprintf("Model API '%s' created successfully (status check failed: %v)", req.Name, err),
				ModelAPI: map[string]interface{}{
					"name": req.Name,
				},
			}

			// Add details to result
			if integrationName != "" {
				result.ModelAPI["integrationConnection"] = integrationName
				if hasProvider {
					result.Message = fmt.Sprintf("Model API '%s' created successfully with inline integration '%s' (status check failed: %v)", req.Name, integrationName, err)
					result.ModelAPI["provider"] = req.Provider
				}
			}

			if req.Model != "" {
				result.ModelAPI["model"] = req.Model
			}

			if req.Endpoint != "" {
				result.ModelAPI["endpoint"] = req.Endpoint
			}

			return result, nil
		}
	} else {
		logger.Printf("Skipping status wait for model API '%s'", req.Name)
	}

	// Model API successfully created (and deployed if we waited)
	deploymentStatus := "created"
	if waitForCompletion {
		deploymentStatus = "created and deployed"
	}

	result := &CreateModelAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Model API '%s' %s successfully", req.Name, deploymentStatus),
		ModelAPI: map[string]interface{}{
			"name": req.Name,
		},
	}

	// Add details to result
	if integrationName != "" {
		result.ModelAPI["integrationConnection"] = integrationName
		if hasProvider {
			result.Message = fmt.Sprintf("Model API '%s' %s successfully with inline integration '%s'", req.Name, deploymentStatus, integrationName)
			result.ModelAPI["provider"] = req.Provider
		}
	}

	if req.Model != "" {
		result.ModelAPI["model"] = req.Model
	}

	if req.Endpoint != "" {
		result.ModelAPI["endpoint"] = req.Endpoint
	}

	return result, nil
}

func deleteModelAPIHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteModelAPIRequest) (*DeleteModelAPIResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Delete the model API
	_, err := sdkClient.DeleteModelWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete model API: %w", err)
	}

	// Check if we should wait for completion
	waitForCompletion := true // default to true
	if req.WaitForCompletion != "" {
		waitForCompletion = req.WaitForCompletion == "true"
	}

	// Wait for the model API to be fully deleted if requested
	if waitForCompletion {
		logger.Printf("Waiting for model API '%s' to be fully deleted...", req.Name)
		checker := utils.NewModelAPIStatusChecker(sdkClient)
		err = utils.WaitForResourceDeletion(ctx, req.Name, checker)
		if err != nil {
			// Even if deletion polling fails, we still initiated the deletion
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: Model API deletion initiated but status check failed: %v", err)
			return &DeleteModelAPIResponse{
				Success: true,
				Message: fmt.Sprintf("Model API '%s' deletion initiated (status check failed: %v)", req.Name, err),
			}, nil
		}
	} else {
		logger.Printf("Skipping deletion wait for model API '%s'", req.Name)
	}

	// Model API successfully deleted (or deletion initiated)
	deletionStatus := "deleted"
	if !waitForCompletion {
		deletionStatus = "deletion initiated"
	}

	return &DeleteModelAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Model API '%s' %s successfully", req.Name, deletionStatus),
	}, nil
}
