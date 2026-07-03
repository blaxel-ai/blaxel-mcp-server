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

type createdServiceAccountResponse struct {
	ClientID     *string `json:"client_id,omitempty"`
	ClientSecret *string `json:"client_secret,omitempty"`
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

// resolveHandler returns a handler for the given workspace override.
func resolveHandler(defaultHandler ServiceAccountHandler, cfg *config.Config, workspace string) (ServiceAccountHandler, error) {
	overriddenCfg := cfg.WithWorkspace(workspace)
	if overriddenCfg == cfg {
		return defaultHandler, nil
	}
	return NewSDKHandler(overriddenCfg)
}

func parseCreatedServiceAccount(account *sdk.CreateWorkspaceServiceAccountResponse) (*createdServiceAccountResponse, error) {
	if account.JSON200 != nil {
		return &createdServiceAccountResponse{
			ClientID:     account.JSON200.ClientId,
			ClientSecret: account.JSON200.ClientSecret,
		}, nil
	}

	if account.StatusCode() < 200 || account.StatusCode() >= 300 {
		return nil, fmt.Errorf("failed to create service account with status %d", account.StatusCode())
	}

	if len(account.Body) == 0 {
		return nil, fmt.Errorf("no service account created")
	}

	var createdAccount createdServiceAccountResponse
	if err := json.Unmarshal(account.Body, &createdAccount); err != nil {
		return nil, fmt.Errorf("failed to parse service account creation response: %w", err)
	}

	return &createdAccount, nil
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
	fmt.Fprintf(&formattedResult, "Found %d service account(s):\n\n", len(*serviceAccounts.JSON200))

	count := 0
	for _, account := range *serviceAccounts.JSON200 {
		// Apply filter if requested
		if filter != "" {
			if account.Name == nil || !tools.ContainsString(*account.Name, filter) {
				continue
			}
		}
		count++

		fmt.Fprintf(&formattedResult, "Service Account #%d:\n", count)

		if account.Name != nil {
			fmt.Fprintf(&formattedResult, "  Name: %s\n", *account.Name)
		}

		if account.ClientId != nil {
			fmt.Fprintf(&formattedResult, "  Client ID: %s\n", *account.ClientId)
		}

		if account.Description != nil && *account.Description != "" {
			fmt.Fprintf(&formattedResult, "  Description: %s\n", *account.Description)
		}

		if account.CreatedAt != nil {
			fmt.Fprintf(&formattedResult, "  Created: %s\n", *account.CreatedAt)
		}

		formattedResult.WriteString("\n")
	}

	if count == 0 && filter != "" {
		formattedResult.Reset()
		fmt.Fprintf(&formattedResult, "No service accounts found matching filter: %s", filter)
	}

	return []byte(formattedResult.String()), nil
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

	createdAccount, err := parseCreatedServiceAccount(account)
	if err != nil {
		return nil, err
	}

	serviceAccount := map[string]interface{}{
		"name":      name,
		"client_id": "",
	}
	result := map[string]interface{}{
		"success":         true,
		"message":         fmt.Sprintf("Service account '%s' created successfully", name),
		"service_account": serviceAccount,
	}

	if createdAccount.ClientID != nil {
		serviceAccount["client_id"] = *createdAccount.ClientID
		if createdAccount.ClientSecret != nil {
			serviceAccount["client_secret"] = *createdAccount.ClientSecret
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
