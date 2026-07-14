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

// resolveHandler returns a handler for the given workspace override.
func resolveHandler(defaultHandler ModelAPIHandler, cfg *config.Config, workspace string) (ModelAPIHandler, error) {
	overriddenCfg := cfg.WithWorkspace(workspace)
	if overriddenCfg == cfg {
		return defaultHandler, nil
	}
	return NewSDKHandler(overriddenCfg)
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

	// The tool supports exactly one integration mode: reference an existing
	// connection, or create an inline connection from provider + apiKey. Config
	// belongs to the inline integration, not the model runtime.
	hasExisting := integrationConnectionName != ""
	hasProvider := provider != ""
	hasApiKey := apiKey != ""
	if hasExisting && (hasProvider || hasApiKey || len(config) > 0) {
		return nil, fmt.Errorf("integrationConnectionName cannot be combined with provider, apiKey, or config")
	}
	if !hasExisting {
		if !hasProvider {
			if hasApiKey || len(config) > 0 {
				return nil, fmt.Errorf("provider is required when specifying apiKey or config")
			}
			return nil, fmt.Errorf("must provide either integrationConnectionName or provider with apiKey")
		}
		if !hasApiKey {
			return nil, fmt.Errorf("apiKey is required when specifying provider")
		}
	}

	inlineConfig := make(map[string]string, len(config))
	for key, value := range config {
		stringValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("config.%s must be a string", key)
		}
		inlineConfig[key] = stringValue
	}

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
	if endpoint != "" {
		modelData.Spec.Runtime.EndpointName = &endpoint
	}

	// Handle integration configuration.
	var integrationName string
	inlineIntegrationOwned := false
	if hasExisting {
		integrationName = integrationConnectionName
	} else {
		// Generate a deterministic name. A collision is an error: this invocation
		// must never claim or overwrite a connection it did not create.
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

		// Add API key and optional provider configuration to the inline connection.
		secrets := map[string]string{"apiKey": apiKey}
		integrationData.Spec.Secret = &secrets
		if len(inlineConfig) > 0 {
			integrationData.Spec.Config = &inlineConfig
		}

		// Create the integration.
		integrationResp, err := h.sdkClient.CreateIntegrationConnectionWithResponse(ctx, integrationData)
		if err != nil {
			return nil, fmt.Errorf("failed to create inline integration: %w", err)
		}

		if integrationResp.StatusCode() < http.StatusOK || integrationResp.StatusCode() >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("failed to create inline integration with status %d", integrationResp.StatusCode())
		}
		inlineIntegrationOwned = true
	}

	// Set the integration connection on the model
	if integrationName != "" {
		connections := sdk.IntegrationConnectionsList{integrationName}

		modelData.Spec.IntegrationConnections = &connections
		if hasProvider {
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
		createErr := fmt.Errorf("failed to create model API: %w", err)
		return nil, h.withInlineIntegrationRollback(ctx, integrationName, inlineIntegrationOwned, createErr)
	}

	if modelResp.JSON200 == nil {
		var createErr error
		if modelResp.StatusCode() == http.StatusConflict {
			createErr = fmt.Errorf("model API with name '%s' already exists", name)
		} else {
			createErr = fmt.Errorf("failed to create model API with status %d", modelResp.StatusCode())
		}
		return nil, h.withInlineIntegrationRollback(ctx, integrationName, inlineIntegrationOwned, createErr)
	}

	// Check if we should wait for completion
	waitForCompletionBool := true // default to true
	if waitForCompletion != "" {
		waitForCompletionBool = waitForCompletion == "true"
	}

	// Wait for the model API to reach a final status if requested.
	deploymentFinalStatus := ""
	if waitForCompletionBool {
		logger.Printf("Waiting for model API '%s' to deploy...", name)
		checker := NewModelAPIStatusChecker(h.sdkClient)
		err = utils.WaitForResourceStatus(ctx, name, checker)
		// FAILED is a terminal deployment result, so the wait is complete even
		// though the generic status helper reports it as an unsuccessful deploy.
		deploymentFinalStatus = checker.LastStatus()
		if err != nil && deploymentFinalStatus != "FAILED" {
			return nil, fmt.Errorf("model API '%s' status wait failed: %w", name, err)
		}
	} else {
		logger.Printf("Skipping status wait for model API '%s'", name)
	}

	// The model API was created and, when requested, reached a terminal state.
	message := fmt.Sprintf("Model API '%s' created successfully", name)
	switch deploymentFinalStatus {
	case "DEPLOYED":
		message = fmt.Sprintf("Model API '%s' created and deployed successfully", name)
	case "FAILED":
		message = fmt.Sprintf("Model API '%s' created; deployment reached terminal status FAILED", name)
	}

	result := map[string]interface{}{
		"success": true,
		"message": message,
		"model_api": map[string]interface{}{
			"name": name,
		},
	}
	if deploymentFinalStatus != "" {
		result["model_api"].(map[string]interface{})["status"] = deploymentFinalStatus
	}

	// Add details to result
	if integrationName != "" {
		result["model_api"].(map[string]interface{})["integrationConnection"] = integrationName
		if hasProvider {
			result["message"] = fmt.Sprintf("%s with inline integration '%s'", message, integrationName)
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

func (h *SDKHandler) withInlineIntegrationRollback(ctx context.Context, name string, owned bool, createErr error) error {
	if !owned {
		return createErr
	}
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	response, err := h.sdkClient.DeleteIntegrationConnectionWithResponse(rollbackCtx, name)
	if err != nil {
		return fmt.Errorf("%w; cleanup of inline integration '%s' failed", createErr, name)
	}
	if status := response.StatusCode(); status != http.StatusNotFound && (status < http.StatusOK || status >= http.StatusMultipleChoices) {
		return fmt.Errorf("%w; cleanup of inline integration '%s' failed with status %d", createErr, name, status)
	}
	return createErr
}

// DeleteModelAPI implements ModelAPIHandler.DeleteModelAPI
func (h *SDKHandler) DeleteModelAPI(ctx context.Context, name, waitForCompletion string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Delete the model API
	resp, err := h.sdkClient.DeleteModelWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete model API: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("model API '%s' not found", name)
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("failed to delete model API '%s': status %d", name, resp.StatusCode())
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
		if err = utils.WaitForResourceDeletion(ctx, name, checker); err != nil {
			return nil, fmt.Errorf("model API '%s' deletion wait failed: %w", name, err)
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
	sdkClient  *sdk.ClientWithResponses
	lastStatus string
}

// NewModelAPIStatusChecker creates a new model API status checker
func NewModelAPIStatusChecker(sdkClient *sdk.ClientWithResponses) *ModelAPIStatusChecker {
	return &ModelAPIStatusChecker{sdkClient: sdkClient}
}

// GetResource gets the model API resource
func (m *ModelAPIStatusChecker) GetResource(ctx context.Context, name string) (interface{}, error) {
	resp, err := m.sdkClient.GetModelWithResponse(ctx, name)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("model API '%s' not found (status 404)", name)
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get model API '%s' failed with status %d", name, resp.StatusCode())
	}
	return resp, nil
}

// ExtractStatus extracts status from model API response
func (m *ModelAPIStatusChecker) ExtractStatus(resource interface{}) string {
	// Type assertion to get the model response
	if modelResp, ok := resource.(*sdk.GetModelResponse); ok {
		if modelResp.JSON200 != nil {
			if modelResp.JSON200.Status == nil {
				m.lastStatus = "DEPLOYING"
				return m.lastStatus
			}
			m.lastStatus = *modelResp.JSON200.Status
			return m.lastStatus
		}
	}
	logger.Printf("Model API could not be extracted: %+v", resource)
	return "DEPLOYING" // Default assumption
}

// GetResourceType returns the resource type
func (m *ModelAPIStatusChecker) GetResourceType() utils.ResourceType {
	return "model_api"
}

// LastStatus returns the latest status observed while polling.
func (m *ModelAPIStatusChecker) LastStatus() string {
	return m.lastStatus
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
