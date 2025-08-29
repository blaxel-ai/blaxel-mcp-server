package modelapis

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/formatter"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/logger"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/utils"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements ModelAPIHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based model API handler
func NewSDKHandler(cfg *config.Config) (ModelAPIHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// ListModelAPIs implements ModelAPIHandler.ListModelAPIs
func (h *SDKHandler) ListModelAPIs(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.ListModelsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list model APIs: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list model APIs failed with status %d", resp.StatusCode())
	}

	models := []sdk.Model{}
	if resp.JSON200 != nil {
		models = *resp.JSON200
	}

	// Apply filter if requested
	if filter != "" {
		var filtered []sdk.Model
		for _, model := range models {
			if model.Metadata != nil && model.Metadata.Name != nil &&
				tools.ContainsString(*model.Metadata.Name, filter) {
				filtered = append(filtered, model)
			}
		}
		models = filtered
	}

	// Convert SDK models to simple models
	modelModels := make([]formatter.ModelAPI, len(models))
	for i, model := range models {
		modelModels[i] = convertToModelAPIModel(model)
	}

	// Format the models using the formatter
	formatted := formatter.FormatModels(modelModels)
	return []byte(formatted), nil
}

// GetModelAPI implements ModelAPIHandler.GetModelAPI
func (h *SDKHandler) GetModelAPI(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	model, err := h.sdkClient.GetModelWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get model API: %w", err)
	}

	if model.JSON200 == nil {
		return nil, fmt.Errorf("no model API found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(*model.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format model API data: %w", err)
	}

	return jsonData, nil
}

// CreateModelAPI implements ModelAPIHandler.CreateModelAPI
func (h *SDKHandler) CreateModelAPI(ctx context.Context, name, model, endpoint, integrationConnectionName, provider, apiKey, waitForCompletion string, config map[string]interface{}) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Check for integration parameters
	hasExisting := integrationConnectionName != ""
	hasProvider := provider != ""
	hasApiKey := apiKey != ""

	// Build model API request
	modelData := sdk.CreateModelJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &name,
		},
		Spec: &sdk.ModelSpec{
			Runtime: &sdk.Runtime{
				Model: &model,
			},
		},
	}

	// Handle integration configuration
	var integrationName string
	if hasExisting {
		// Use existing integration connection
		if integrationConnectionName == "" {
			return nil, fmt.Errorf("integrationConnectionName cannot be empty")
		}
		integrationName = integrationConnectionName
	} else if hasProvider {
		if !hasApiKey {
			return nil, fmt.Errorf("api key is required when specifying provider")
		}

		// Generate a unique name for the integration
		integrationName = fmt.Sprintf("%s-%s-integration", name, provider)

		// Create the integration
		integrationData := sdk.CreateIntegrationConnectionJSONRequestBody{
			Metadata: &sdk.Metadata{
				Name: &integrationName,
			},
			Spec: &sdk.IntegrationConnectionSpec{
				Integration: &provider, // Provider is the integration type
			},
		}

		// Add API key to secrets
		secrets := map[string]string{
			"apiKey": apiKey,
		}
		integrationData.Spec.Secret = &secrets

		// Create the integration
		integrationResp, err := h.sdkClient.CreateIntegrationConnectionWithResponse(ctx, integrationData)
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
		if provider != "" {
			modelData.Spec.Runtime.Type = &provider
		} else {
			response, err := h.sdkClient.GetIntegrationConnectionWithResponse(ctx, integrationName)
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
	modelResp, err := h.sdkClient.CreateModelWithResponse(ctx, modelData)
	if err != nil {
		return nil, fmt.Errorf("failed to create model API: %w", err)
	}

	if modelResp.JSON200 == nil {
		if modelResp.StatusCode() == 409 {
			return nil, fmt.Errorf("model API with name '%s' already exists", name)
		}
		return nil, fmt.Errorf("failed to create model API with status %d", modelResp.StatusCode())
	}

	// Check if we should wait for completion
	waitForCompletionBool := true // default to true
	if waitForCompletion != "" {
		waitForCompletionBool = waitForCompletion == "true"
	}

	// Wait for the model API to reach a final status if requested
	if waitForCompletionBool {
		logger.Printf("Waiting for model API '%s' to deploy...", name)
		checker := NewModelAPIStatusChecker(h.sdkClient)
		err = utils.WaitForResourceStatus(ctx, name, checker)
		if err != nil {
			// Even if status waiting fails, we still created the model API
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: Model API created but status check failed: %v", err)
			result := map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Model API '%s' created successfully (status check failed: %v)", name, err),
				"model_api": map[string]interface{}{
					"name": name,
				},
			}

			// Add details to result
			if integrationName != "" {
				result["model_api"].(map[string]interface{})["integrationConnection"] = integrationName
				if hasProvider {
					result["message"] = fmt.Sprintf("Model API '%s' created successfully with inline integration '%s' (status check failed: %v)", name, integrationName, err)
					result["model_api"].(map[string]interface{})["provider"] = provider
				}
			}

			if model != "" {
				result["model_api"].(map[string]interface{})["model"] = model
			}

			if endpoint != "" {
				result["model_api"].(map[string]interface{})["endpoint"] = endpoint
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format response: %w", err)
			}

			return jsonData, nil
		}
	} else {
		logger.Printf("Skipping status wait for model API '%s'", name)
	}

	// Model API successfully created (and deployed if we waited)
	deploymentStatus := "created"
	if waitForCompletionBool {
		deploymentStatus = "created and deployed"
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Model API '%s' %s successfully", name, deploymentStatus),
		"model_api": map[string]interface{}{
			"name": name,
		},
	}

	// Add details to result
	if integrationName != "" {
		result["model_api"].(map[string]interface{})["integrationConnection"] = integrationName
		if hasProvider {
			result["message"] = fmt.Sprintf("Model API '%s' %s successfully with inline integration '%s'", name, deploymentStatus, integrationName)
			result["model_api"].(map[string]interface{})["provider"] = provider
		}
	}

	if model != "" {
		result["model_api"].(map[string]interface{})["model"] = model
	}

	if endpoint != "" {
		result["model_api"].(map[string]interface{})["endpoint"] = endpoint
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// DeleteModelAPI implements ModelAPIHandler.DeleteModelAPI
func (h *SDKHandler) DeleteModelAPI(ctx context.Context, name, waitForCompletion string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Delete the model API
	_, err := h.sdkClient.DeleteModelWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete model API: %w", err)
	}

	// Check if we should wait for completion
	waitForCompletionBool := true // default to true
	if waitForCompletion != "" {
		waitForCompletionBool = waitForCompletion == "true"
	}

	// Wait for the model API to be fully deleted if requested
	if waitForCompletionBool {
		logger.Printf("Waiting for model API '%s' to be fully deleted...", name)
		checker := NewModelAPIStatusChecker(h.sdkClient)
		err = utils.WaitForResourceDeletion(ctx, name, checker)
		if err != nil {
			// Even if deletion polling fails, we still initiated the deletion
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: Model API deletion initiated but status check failed: %v", err)
			result := map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Model API '%s' deletion initiated (status check failed: %v)", name, err),
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format response: %w", err)
			}

			return jsonData, nil
		}
	} else {
		logger.Printf("Skipping deletion wait for model API '%s'", name)
	}

	// Model API successfully deleted (or deletion initiated)
	deletionStatus := "deleted"
	if !waitForCompletionBool {
		deletionStatus = "deletion initiated"
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Model API '%s' %s successfully", name, deletionStatus),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements ModelAPIHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}

// ModelAPIStatusChecker implements StatusChecker for model APIs
type ModelAPIStatusChecker struct {
	sdkClient *sdk.ClientWithResponses
}

// NewModelAPIStatusChecker creates a new model API status checker
func NewModelAPIStatusChecker(sdkClient *sdk.ClientWithResponses) *ModelAPIStatusChecker {
	return &ModelAPIStatusChecker{sdkClient: sdkClient}
}

// GetResource gets the model API resource
func (m *ModelAPIStatusChecker) GetResource(ctx context.Context, name string) (interface{}, error) {
	return m.sdkClient.GetModelWithResponse(ctx, name)
}

// ExtractStatus extracts status from model API response
func (m *ModelAPIStatusChecker) ExtractStatus(resource interface{}) string {
	// Type assertion to get the model response
	if modelResp, ok := resource.(*sdk.GetModelResponse); ok {
		if modelResp.JSON200 != nil {
			if modelResp.JSON200.Status == nil {
				return "DEPLOYING"
			}
			return *modelResp.JSON200.Status
		}
	}
	logger.Printf("Model API could not be extracted: %+v", resource)
	return "DEPLOYING" // Default assumption
}

// GetResourceType returns the resource type
func (m *ModelAPIStatusChecker) GetResourceType() utils.ResourceType {
	return "model_api"
}

// convertToModelAPIModel converts an SDK model to a simple model API model
func convertToModelAPIModel(model sdk.Model) formatter.ModelAPI {
	modelAPI := formatter.ModelAPI{
		Name:   "",
		Status: "",
		Labels: make(map[string]string),
	}

	// Extract name
	if model.Metadata != nil && model.Metadata.Name != nil {
		modelAPI.Name = *model.Metadata.Name
	}

	// Extract status
	if model.Status != nil {
		modelAPI.Status = *model.Status
	}

	// Extract labels
	if model.Metadata != nil && model.Metadata.Labels != nil {
		modelAPI.Labels = *model.Metadata.Labels
	}

	// Extract runtime spec
	if model.Spec != nil && model.Spec.Runtime != nil {
		if model.Spec.Runtime.Type != nil {
			modelAPI.Type = model.Spec.Runtime.Type
		}
		if model.Spec.Runtime.Model != nil {
			modelAPI.ModelName = model.Spec.Runtime.Model
		}
		if model.Spec.Runtime.Memory != nil {
			modelAPI.Memory = model.Spec.Runtime.Memory
		}
	}

	// Extract creation time
	if model.Metadata != nil && model.Metadata.CreatedAt != nil {
		// Parse the time string to time.Time
		if createdAt, err := time.Parse(time.RFC3339, *model.Metadata.CreatedAt); err == nil {
			modelAPI.CreatedAt = &createdAt
		}
	}

	return modelAPI
}
