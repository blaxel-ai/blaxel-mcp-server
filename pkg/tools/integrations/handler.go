package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/formatter"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements IntegrationHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based integration handler
func NewSDKHandler(cfg *config.Config) (IntegrationHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// ListIntegrations implements IntegrationHandler.ListIntegrations
func (h *SDKHandler) ListIntegrations(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.ListIntegrationConnectionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list integrations failed with status %d", resp.StatusCode())
	}

	integrations := []sdk.IntegrationConnection{}
	if resp.JSON200 != nil {
		integrations = *resp.JSON200
	}

	// Apply filter if requested
	if filter != "" {
		var filtered []sdk.IntegrationConnection
		for _, integration := range integrations {
			if integration.Metadata != nil && integration.Metadata.Name != nil &&
				tools.ContainsString(*integration.Metadata.Name, filter) {
				filtered = append(filtered, integration)
			}
		}
		integrations = filtered
	}

	// Format the integrations using the formatter
	formatted := formatter.FormatIntegrations(integrations)
	return []byte(formatted), nil
}

// GetIntegration implements IntegrationHandler.GetIntegration
func (h *SDKHandler) GetIntegration(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	integration, err := h.sdkClient.GetIntegrationConnectionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration: %w", err)
	}

	if integration.JSON200 == nil {
		return nil, fmt.Errorf("no integration found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(*integration.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format integration data: %w", err)
	}

	return jsonData, nil
}

// CreateIntegration implements IntegrationHandler.CreateIntegration
func (h *SDKHandler) CreateIntegration(ctx context.Context, name, integrationType string, secret, config map[string]string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Build integration request
	integrationData := sdk.CreateIntegrationConnectionJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &name,
		},
		Spec: &sdk.IntegrationConnectionSpec{
			Integration: &integrationType,
		},
	}

	// Add secret if provided
	if len(secret) > 0 {
		integrationData.Spec.Secret = &secret
	}

	// Add config if provided
	if len(config) > 0 {
		integrationData.Spec.Config = &config
	}

	integration, err := h.sdkClient.CreateIntegrationConnectionWithResponse(ctx, integrationData)
	if err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	if integration.JSON200 == nil {
		if integration.StatusCode() == 409 {
			return nil, fmt.Errorf("integration with name '%s' already exists", name)
		}
		return nil, fmt.Errorf("failed to create integration with status %d", integration.StatusCode())
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Integration '%s' created successfully", name),
		"integration": map[string]interface{}{
			"name": name,
			"type": integrationType,
		},
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// DeleteIntegration implements IntegrationHandler.DeleteIntegration
func (h *SDKHandler) DeleteIntegration(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := h.sdkClient.DeleteIntegrationConnectionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete integration: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Integration '%s' deleted successfully", name),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements IntegrationHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}
