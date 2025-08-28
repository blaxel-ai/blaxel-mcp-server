package serviceaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements ServiceAccountHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based service account handler
func NewSDKHandler(cfg *config.Config) (ServiceAccountHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// ListServiceAccounts implements ServiceAccountHandler.ListServiceAccounts
func (h *SDKHandler) ListServiceAccounts(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	serviceAccounts, err := h.sdkClient.GetWorkspaceServiceAccountsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts: %w", err)
	}

	if serviceAccounts.JSON200 == nil {
		return nil, fmt.Errorf("no service accounts found")
	}

	// Convert service accounts for formatting
	var formattedResult strings.Builder
	formattedResult.WriteString(fmt.Sprintf("Found %d service account(s):\n\n", len(*serviceAccounts.JSON200)))

	count := 0
	for _, account := range *serviceAccounts.JSON200 {
		// Apply filter if requested
		if filter != "" {
			if account.Name == nil || !tools.ContainsString(*account.Name, filter) {
				continue
			}
		}
		count++

		formattedResult.WriteString(fmt.Sprintf("Service Account #%d:\n", count))

		if account.Name != nil {
			formattedResult.WriteString(fmt.Sprintf("  Name: %s\n", *account.Name))
		}

		if account.ClientId != nil {
			formattedResult.WriteString(fmt.Sprintf("  Client ID: %s\n", *account.ClientId))
		}

		if account.Description != nil && *account.Description != "" {
			formattedResult.WriteString(fmt.Sprintf("  Description: %s\n", *account.Description))
		}

		if account.CreatedAt != nil {
			formattedResult.WriteString(fmt.Sprintf("  Created: %s\n", *account.CreatedAt))
		}

		formattedResult.WriteString("\n")
	}

	if count == 0 && filter != "" {
		formattedResult.Reset()
		formattedResult.WriteString(fmt.Sprintf("No service accounts found matching filter: %s", filter))
	}

	return []byte(formattedResult.String()), nil
}

// GetServiceAccount implements ServiceAccountHandler.GetServiceAccount
func (h *SDKHandler) GetServiceAccount(ctx context.Context, clientID string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// List all service accounts and find the one with matching client ID
	serviceAccounts, err := h.sdkClient.GetWorkspaceServiceAccountsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get service account: %w", err)
	}

	if serviceAccounts.JSON200 == nil {
		return nil, fmt.Errorf("no service accounts found")
	}

	for _, account := range *serviceAccounts.JSON200 {
		if account.ClientId != nil && *account.ClientId == clientID {
			jsonData, err := json.MarshalIndent(account, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format service account data: %w", err)
			}
			return jsonData, nil
		}
	}

	return nil, fmt.Errorf("service account with client ID '%s' not found", clientID)
}

// CreateServiceAccount implements ServiceAccountHandler.CreateServiceAccount
func (h *SDKHandler) CreateServiceAccount(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	serviceAccountData := sdk.CreateWorkspaceServiceAccountJSONRequestBody{
		Name: name,
	}

	account, err := h.sdkClient.CreateWorkspaceServiceAccountWithResponse(ctx, serviceAccountData)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	if account.JSON200 == nil {
		return nil, fmt.Errorf("no service account created")
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Service account '%s' created successfully", name),
		"service_account": map[string]interface{}{
			"name":      name,
			"client_id": "",
		},
	}

	if account.JSON200 != nil && account.JSON200.ClientId != nil {
		result["service_account"].(map[string]interface{})["client_id"] = *account.JSON200.ClientId
		if account.JSON200.ClientSecret != nil {
			result["service_account"].(map[string]interface{})["client_secret"] = *account.JSON200.ClientSecret
			result["message"] = fmt.Sprintf("Service account '%s' created successfully. Save the client_secret as it won't be shown again.", name)
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// DeleteServiceAccount implements ServiceAccountHandler.DeleteServiceAccount
func (h *SDKHandler) DeleteServiceAccount(ctx context.Context, clientID string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := h.sdkClient.DeleteWorkspaceServiceAccountWithResponse(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete service account: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Service account with client ID '%s' deleted successfully", clientID),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// UpdateServiceAccount implements ServiceAccountHandler.UpdateServiceAccount
func (h *SDKHandler) UpdateServiceAccount(ctx context.Context, clientID, description string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Build update request
	updateData := sdk.UpdateWorkspaceServiceAccountJSONRequestBody{
		Description: &description,
	}

	// Update the service account
	resp, err := h.sdkClient.UpdateWorkspaceServiceAccountWithResponse(ctx, clientID, updateData)
	if err != nil {
		return nil, fmt.Errorf("failed to update service account: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to update service account with status %d", resp.StatusCode())
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Service account '%s' updated successfully", clientID),
		"service_account": map[string]interface{}{
			"clientId":    clientID,
			"description": description,
		},
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements ServiceAccountHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}
